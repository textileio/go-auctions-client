package main

import (
	"time"

	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multibase"
	"github.com/spf13/cobra"
	"github.com/textileio/cli"
	"github.com/textileio/go-auctions-client/buildinfo"
	"github.com/textileio/go-auctions-client/localwallet"
	"github.com/textileio/go-auctions-client/propsigner"
	"github.com/textileio/go-auctions-client/relaymgr"
)

var walletCmd = &cobra.Command{
	Use:   "wallet",
	Short: "wallet provides remote signing capabilities to run auctions",
	Long:  `wallet provides remote signing capabilities to run auctions`,
	Args:  cobra.ExactArgs(0),
}

var walletDaemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run a remote wallet signer for auctions",
	Long:  "Run a remote wallet signer for auctions",
	Args:  cobra.ExactArgs(0),
	PersistentPreRun: func(c *cobra.Command, args []string) {
		cli.ExpandEnvVars(v, v.AllSettings())
		err := cli.ConfigureLogging(v, nil)
		cli.CheckErrf("setting log levels: %v", err)
	},
	Run: func(c *cobra.Command, args []string) {
		log.Infof("auc %s", buildinfo.Summary())

		settings, err := cli.MarshalConfig(v, !v.GetBool("log-json"), "wallet-keys", "auth-token", "private-key")
		cli.CheckErrf("marshaling config: %v", err)
		log.Infof("loaded config from %s: %s", v.ConfigFileUsed(), string(settings))

		authToken := v.GetString("auth-token")
		walletKeys := v.GetStringSlice("wallet-keys")
		wallet, err := localwallet.New(walletKeys)
		cli.CheckErrf("creating local wallet: %s", err)
		addrs := wallet.GetAddresses()
		for _, addr := range addrs {
			log.Infof("Loaded wallet: %s", addr)
		}

		_, key, err := multibase.Decode(v.GetString("private-key"))
		cli.CheckErrf("decoding private key: %v", err)
		sk, err := crypto.UnmarshalPrivateKey(key)
		cli.CheckErrf("unmarshaling private key: %v", err)
		conmgr, err := connmgr.NewConnManager(10, 20, connmgr.WithGracePeriod(time.Minute))
		cli.CheckErrf("creating conn manager: %s", err)
		opts := []libp2p.Option{
			libp2p.ConnectionManager(conmgr),
			libp2p.Identity(sk),
		}
		if v.GetString("listen-maddr") != "" {
			listenMaddr, err := multiaddr.NewMultiaddr(v.GetString("listen-maddr"))
			cli.CheckErrf("parsing listen multiaddr: %s", err)
			opts = append(opts, libp2p.ListenAddrs(listenMaddr))
		}

		h, err := libp2p.New(opts...)
		cli.CheckErrf("creating libp2p host: %s", err)
		printHostInfo(h)

		var rlymgr *relaymgr.RelayManager
		if v.GetString("relay-maddr") != "" {
			rlymgr, err = relaymgr.New(c.Context(), h, v.GetString("relay-maddr"))
			cli.CheckErrf("connecting with relay: %s", err)

			maddrcircuit := v.GetString("relay-maddr") + "/p2p-circuit/" + h.ID().String()
			log.Infof("Relayed multiaddr: %s", maddrcircuit)
		} else {
			log.Warnf("libp2p relaying is disabled")
		}

		err = propsigner.NewDealSignerService(h, authToken, wallet)
		cli.CheckErrf("creating deal signer service: %s", err)

		cli.HandleInterrupt(func() {
			if rlymgr != nil {
				if err := rlymgr.Close(); err != nil {
					log.Errorf("closing relay manager: %s", err)
				}
			}
			if err := h.Close(); err != nil {
				log.Errorf("closing libp2p host: %s", err)
			}
		})
	},
}

func printHostInfo(h host.Host) {
	log.Infof("libp2p peer-id: %s", h.ID())
	for _, maddr := range h.Addrs() {
		log.Infof("Listen multiaddr: %s", maddr)
	}
}
