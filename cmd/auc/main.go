package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/go-auctions-client/common"
	logger "github.com/textileio/go-log/v2"
)

var (
	cliName           = "auc"
	envPrefix         = strings.ToUpper(cliName)
	defaultConfigPath = filepath.Join(os.Getenv("HOME"), "."+cliName)
	log               = logger.Logger(cliName)
	v                 = viper.New()
)

func init() {
	// Configuration.
	configPath := os.Getenv("AUC_PATH")
	if configPath == "" {
		configPath = defaultConfigPath
	}

	cobra.OnInitialize(func() {
		v.SetConfigType("json")
		v.SetConfigName("config")
		v.AddConfigPath(os.Getenv(envPrefix + "_PATH"))
		v.AddConfigPath(defaultConfigPath)
		_ = v.ReadInConfig()
	})

	// Commands.
	rootCmd.AddCommand(walletCmd)
	common.ConfigureCLI(v, envPrefix, []common.Flag{
		{Name: "log-debug", DefValue: false, Description: "Enable debug level log"},
		{Name: "log-json", DefValue: false, Description: "Enable structured logging"},
	}, rootCmd.PersistentFlags())

	walletCmd.AddCommand(walletDaemonCmd)
	common.ConfigureCLI(v, envPrefix, []common.Flag{
		{Name: "wallet-keys", DefValue: []string{}, Description: "Wallet address keys"},
		{Name: "auth-token", DefValue: "", Description: "Authorization token to validate signing requests"},
	}, walletDaemonCmd.Flags())

}

var rootCmd = &cobra.Command{
	Use:   cliName,
	Short: "Auctions client provides a CLI to run auctions",
	Long:  `Auctions client provides a CLI to run auctions.`,
	Args:  cobra.ExactArgs(0),
}

func main() {
	common.CheckErr(rootCmd.Execute())

}
