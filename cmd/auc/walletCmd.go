package main

import (
	"github.com/libp2p/go-libp2p"
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
		err := cli.ConfigureLogging(v, []string{
			cliName,
		})
		cli.CheckErrf("setting log levels: %v", err)
	},
	Run: func(c *cobra.Command, args []string) {
		log.Infof("auc %s", buildinfo.Summary())

		settings, err := cli.MarshalConfig(v, !v.GetBool("log-json"), "wallet-keys", "auth-token")
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

		h, err := libp2p.New(c.Context())
		cli.CheckErrf("creating libp2p host: %s", err)
		log.Infof("Remote Wallet PeerID: %s", h.ID())

		rlymgr, err := relaymgr.New(c.Context(), h, v.GetString("relay-addr"))
		cli.CheckErrf("connecting with relay: %s", err)

		err = propsigner.NewDealSignerService(h, authToken, wallet)
		cli.CheckErrf("creating deal signer service: %s", err)

		cli.HandleInterrupt(func() {
			if err := rlymgr.Close(); err != nil {
				log.Errorf("closing relay manager: %s", err)
			}
			if err := h.Close(); err != nil {
				log.Errorf("closing libp2p host: %s", err)
			}
		})
	},
}
