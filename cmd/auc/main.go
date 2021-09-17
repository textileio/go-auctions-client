package main

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/multiformats/go-multibase"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/cli"
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
		v.AddConfigPath(configPath)
		if err := initConfigFile(configPath); err != nil {
			log.Infof("config file can't be read, creating one")
		}
		if err := v.ReadInConfig(); err != nil {
			log.Fatalf("reading config file: %s", err)
		}
	})

	// Commands.
	rootCmd.AddCommand(walletCmd)
	cli.ConfigureCLI(v, envPrefix, []cli.Flag{
		{Name: "log-debug", DefValue: false, Description: "Enable debug level log"},
		{Name: "log-json", DefValue: false, Description: "Enable structured logging"},
	}, rootCmd.PersistentFlags())

	walletCmd.AddCommand(walletDaemonCmd)
	cli.ConfigureCLI(v, envPrefix, []cli.Flag{
		{Name: "wallet-keys", DefValue: []string{}, Description: "Wallet address keys"},
		{Name: "auth-token", DefValue: "", Description: "Authorization token to validate signing requests"},
		{
			Name:        "relay-maddr",
			DefValue:    "/ip4/34.105.85.147/tcp/4001/p2p/QmYRDEq8z3Y9hBBAirwMFySuxyCoWwskrD1bxUEYKBiwmU",
			Description: "Multiaddress of libp2p relay",
		},
		{Name: "listen-maddr", DefValue: "", Description: "Libp2p listen multiaddr"},
		{
			Name:        "private-key",
			DefValue:    "",
			Description: "Libp2p private key",
		},
	}, walletDaemonCmd.Flags())
}

var rootCmd = &cobra.Command{
	Use:   cliName,
	Short: "Auctions client provides a CLI to run auctions",
	Long:  `Auctions client provides a CLI to run auctions.`,
	Args:  cobra.ExactArgs(0),
}

func main() {
	cli.CheckErr(rootCmd.Execute())
}

func initConfigFile(configPath string) error {
	path := filepath.Join(configPath, "config")
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		log.Fatalf("create config file path: %s", err)
	}

	if v.GetString("private-key") == "" {
		priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
		if err != nil {
			return fmt.Errorf("generating private key: %v", err)
		}
		key, err := crypto.MarshalPrivateKey(priv)
		if err != nil {
			return fmt.Errorf("marshaling private key: %v", err)
		}
		keystr, err := multibase.Encode(multibase.Base64, key)
		if err != nil {
			return fmt.Errorf("encoding private key: %v", err)
		}
		v.Set("private-key", keystr)
	}
	if err := v.WriteConfigAs(path); err != nil {
		log.Fatalf("creating config file: %s", err)
	}

	return nil
}
