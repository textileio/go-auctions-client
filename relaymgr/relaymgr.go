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

	pollFrequency = time.Second * 30
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
		host:        h,
		relayAddr:   *addrInfo,
		connNotifee: &connNotifee{peerID: addrInfo.ID},

		closeCtx:    closeCtx,
		closeSignal: closeSignal,
		closed:      make(chan struct{}),
	}

	err = h.Connect(ctx, *addrInfo)
	if err != nil {
		return nil, fmt.Errorf("connecting to relay: %s", err)
	}
	h.ConnManager().Protect(addrInfo.ID, connProtectTag)

	go rm.keepHealthy()

	return rm, nil
}

func (rm *RelayManager) Close() error {
	rm.closeOnce.Do(func() {
		log.Infof("closing relay manager")

		rm.host.ConnManager().Unprotect(rm.relayAddr.ID, connProtectTag)
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
		}
	}
}

type connNotifee struct {
	peerID peer.ID
}

func (n *connNotifee) Connected(_ network.Network, ne network.Conn) {
	if ne.RemotePeer() == n.peerID {
		log.Debugf("connected with remote relay")
	}
}
func (n *connNotifee) Disconnected(_ network.Network, ne network.Conn) {
	if ne.RemotePeer() == n.peerID {
		log.Debugf("disconnected from remote relay")
	}
}
func (n *connNotifee) OpenedStream(_ network.Network, s network.Stream)     {}
func (n *connNotifee) ClosedStream(_ network.Network, s network.Stream)     {}
func (n *connNotifee) ListenClose(_ network.Network, _ multiaddr.Multiaddr) {}
func (n *connNotifee) Listen(net network.Network, ma multiaddr.Multiaddr)   {}
