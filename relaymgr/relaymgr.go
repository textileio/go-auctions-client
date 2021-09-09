package relaymgr

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	logger "github.com/textileio/go-log/v2"
)

const (
	connProtectTag = "auction-relay"
)

var (
	log = logger.Logger("relaymgr")

	pollFrequency = time.Second * 10
)

// RelayManager connects a libp2p host to an external relay and do a best-effort
// in keeping the connection healthy.
type RelayManager struct {
	host        host.Host
	relayAddr   peer.AddrInfo
	connNotifee *connNotifee

	closeOnce   sync.Once
	closeCtx    context.Context
	closeSignal context.CancelFunc
	closed      chan struct{}
}

// New connects the provided host to the remote relay, doing a best-effort
// in keeping the connection healthy. The provided context is only used for
// the initial connection to the relay. To shutdown call Close().
func New(ctx context.Context, h host.Host, relayMultiaddress string) (*RelayManager, error) {
	relayAddr, err := multiaddr.NewMultiaddr(relayMultiaddress)
	if err != nil {
		return nil, fmt.Errorf("parsing relay multiaddr: %s", err)
	}
	addrInfo, err := peer.AddrInfoFromP2pAddr(relayAddr)
	if err != nil {
		return nil, fmt.Errorf("get addr-info from relay multiaddr: %s", err)
	}

	closeCtx, closeSignal := context.WithCancel(context.Background())
	rm := &RelayManager{
		host:      h,
		relayAddr: *addrInfo,

		closeCtx:    closeCtx,
		closeSignal: closeSignal,
		closed:      make(chan struct{}),
	}
	rm.connNotifee = &connNotifee{rm: rm}
	h.Network().Notify(rm.connNotifee)

	if err := rm.connect(); err != nil {
		return nil, fmt.Errorf("connecting to relay: %s", err)
	}

	go rm.keepHealthy()

	return rm, nil
}

func (rm *RelayManager) Close() error {
	rm.closeOnce.Do(func() {
		log.Infof("closing relay manager")

		rm.host.ConnManager().Unprotect(rm.relayAddr.ID, connProtectTag)
		rm.host.Network().StopNotify(rm.connNotifee)
		rm.closeSignal()
		<-rm.closed
		log.Infof("relay manager closed")
	})
	return nil
}

func (rm *RelayManager) keepHealthy() {
	defer close(rm.closed)
	for {
		select {
		case <-rm.closeCtx.Done():
			log.Debugf("closing healthy checker")
			return
		case <-time.After(pollFrequency):
			isProtected := rm.host.ConnManager().IsProtected(rm.relayAddr.ID, "")
			connStatus := rm.host.Network().Connectedness(rm.relayAddr.ID)

			if !isProtected || connStatus != network.Connected {
				log.Warnf("detected unhealthy status of connection (protected: %t, connStatus: %s)", isProtected, connStatus)
				if err := rm.connect(); err != nil {
					log.Errorf("poller reconnect: %s", err)
					continue
				}
			}
			log.Debugf("relay connection is healthy")
		}
	}
}

func (rm *RelayManager) connect() error {
	log.Infof("connecting with relay...")
	err := rm.host.Connect(rm.closeCtx, rm.relayAddr)
	if err != nil {
		return fmt.Errorf("connecting to relay: %s", err)
	}
	rm.host.ConnManager().Protect(rm.relayAddr.ID, connProtectTag)
	log.Infof("connected with relay")

	return nil
}

type connNotifee struct {
	rm *RelayManager
}

func (n *connNotifee) Connected(_ network.Network, ne network.Conn) {
	if ne.RemotePeer() == n.rm.relayAddr.ID {
		log.Debugf("connected with remote relay")
	}
}
func (n *connNotifee) Disconnected(_ network.Network, ne network.Conn) {
	if ne.RemotePeer() == n.rm.relayAddr.ID {
		log.Warnf("disconnected from remote relay")
		if err := n.rm.connect(); err != nil {
			log.Errorf("notifee reconnect: %s", err)
		}
	}
}
func (n *connNotifee) OpenedStream(_ network.Network, s network.Stream)     {}
func (n *connNotifee) ClosedStream(_ network.Network, s network.Stream)     {}
func (n *connNotifee) ListenClose(_ network.Network, _ multiaddr.Multiaddr) {}
func (n *connNotifee) Listen(net network.Network, ma multiaddr.Multiaddr)   {}
