package cmd

import (
	"log"
	"os"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string

	// Shared flags
	xorCryptSeed            int64
	tunServerListenAddress  string
	tunClientConnectAddress string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tnet",
	Short: "Network utils.",
	Long: `Network utils.
Repo: https://github.com/tutils/tnet
Start proxy or agent to setup a TCP tunnel, For example:
  tnet proxy --listen=0.0.0.0:56080 --connect=127.0.0.1:3128 --tunnel-connect=ws://123.45.67.89:8080/stream --crypt-key=816559
  tnet agent --tunnel-listen=ws://0.0.0.0:8080/stream --crypt-key=816559`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

const (
	prefix = "@"
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if len(os.Args) == 2 && strings.HasPrefix(os.Args[1], prefix) {
		decodeCmdline(os.Args[1][1:])
	} else if len(os.Args) >= 2 {
		if s, err := encodeCmdline(); err == nil {
			log.Println(prefix + s)
		}
	}

	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize( /*initConfig*/ )

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	//rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.tcptun.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	//flags := rootCmd.Flags()
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".tcptun" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".conf")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Println("Using config file:", viper.ConfigFileUsed())
	}
}
