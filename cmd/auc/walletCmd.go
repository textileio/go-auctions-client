package main

import (
	"github.com/libp2p/go-libp2p"
	"github.com/spf13/cobra"
	"github.com/textileio/go-auctions-client/buildinfo"
	"github.com/textileio/go-auctions-client/common"
	"github.com/textileio/go-auctions-client/localwallet"
	"github.com/textileio/go-auctions-client/propsigner"
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
		common.ExpandEnvVars(v, v.AllSettings())
		err := common.ConfigureLogging(v, []string{
			cliName,
		})
		common.CheckErrf("setting log levels: %v", err)
	},
	Run: func(c *cobra.Command, args []string) {
		log.Infof("auc %s", buildinfo.Summary())

		settings, err := common.MarshalConfig(v, !v.GetBool("log-json"), "wallet-keys")
		common.CheckErrf("marshaling config: %v", err)
		log.Infof("loaded config from %s: %s", v.ConfigFileUsed(), string(settings))

		authToken := v.GetString("auth-token")
		walletKeys := v.GetStringSlice("wallet-keys")
		wallet, err := localwallet.New(walletKeys)
		common.CheckErrf("creating local wallet: %s", err)

		h, err := libp2p.New(c.Context())
		common.CheckErrf("creating libp2p host: %s", err)

		err = propsigner.NewDealSignerService(h, authToken, wallet)
		common.CheckErrf("creating deal signer service: %s", err)

		common.HandleInterrupt(func() {
			err := h.Close()
			common.CheckErrf("closing libp2p host: %s", err)
		})
	},
}
