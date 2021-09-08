package main

import (
	"fmt"

	"github.com/libp2p/go-libp2p"
	"github.com/spf13/cobra"
	"github.com/textileio/bidbot/buildinfo"
	"github.com/textileio/bidbot/lib/common"
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

		walletMap := v.GetStringMapString("wallet-keys")
		for k, v := range walletMap {
			fmt.Printf("%s: %s\n", k, v)
		}

		//fin := finalizer.NewFinalizer()

		h, err := libp2p.New(c.Context())
		common.CheckErrf("creating libp2p host: %s", err)

		err = propsigner.HandleV1(h)
		common.CheckErrf("configuring proposal signer v1: %s", err)

		///////////////////////
		/*
			config := service.Config{
				Peer: pconfig,
				BidParams: service.BidParams{
					StorageProviderID:       storageProviderID,
					WalletAddrSig:           walletAddrSig,
					AskPrice:                v.GetInt64("ask-price"),
					VerifiedAskPrice:        v.GetInt64("verified-ask-price"),
					FastRetrieval:           v.GetBool("fast-retrieval"),
					DealStartWindow:         v.GetUint64("deal-start-window"),
					DealDataDirectory:       dealDataDirectory,
					DealDataFetchAttempts:   v.GetUint32("deal-data-fetch-attempts"),
					DealDataFetchTimeout:    v.GetDuration("deal-data-fetch-timeout"),
					DiscardOrphanDealsAfter: v.GetDuration("discard-orphan-deals-after"),
				},
				AuctionFilters: service.AuctionFilters{
					DealDuration: service.MinMaxFilter{
						Min: v.GetUint64("deal-duration-min"),
						Max: v.GetUint64("deal-duration-max"),
					},
					DealSize: service.MinMaxFilter{
						Min: v.GetUint64("deal-size-min"),
						Max: v.GetUint64("deal-size-max"),
					},
				},
				BytesLimiter:        bytesLimiter,
				ConcurrentImports:   v.GetInt("concurrent-imports-limit"),
				SealingSectorsLimit: v.GetInt("sealing-sectors-limit"),
			}
			serv, err := service.New(config, store, lc, fc)
			common.CheckErrf("starting service: %v", err)
			fin.Add(serv)
		*/

		common.HandleInterrupt(func() {
			//common.CheckErr(fin.Cleanupf("closing service: %v", nil))
		})
	},
}
