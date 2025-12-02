package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tutils/tnet/endpoint/server"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start tnet management server",
	Long: `Start tnet management server with web interface, For example:
  tnet server --listen=0.0.0.0:8080`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return server.StartServer(serverListenAddress)
	},
}

var (
	serverListenAddress string
)

func init() {
	rootCmd.AddCommand(serverCmd)

	// Here you will define your flags and configuration settings.
	flags := serverCmd.Flags()
	flags.StringVarP(&serverListenAddress, "listen", "l", "0.0.0.0:8080", "server listen address")
}
