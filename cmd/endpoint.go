package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/tutils/tnet/proxy"
	"github.com/tutils/tnet/tun"
)

var (
	tunServerListenAddress string
)

// endpointCmd represents the endpoint command
var endpointCmd = &cobra.Command{
	Use:   "endpoint",
	Short: "TCP tunnel endpoint",
	Long: `Start TCP tunnel endpoint, For example:
  tnet endpoint --tunnel-listen=ws://0.0.0.0:8080/stream --crypt-key=816559`,
	RunE: func(cmd *cobra.Command, args []string) error {
		e := proxy.NewEndpoint(
			proxy.WithTunServer(
				tun.NewServer(
					tun.WithListenAddress(tunServerListenAddress),
					tun.WithServerHandler(proxy.NewTunServerHandler()),
				),
			),
			proxy.WithTunServerCrypt(proxy.DefaultTunCrypt),
		)
		if err := e.ListenAndServe(); err != nil {
			log.Fatalln(err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(endpointCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// endpointCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	flags := endpointCmd.Flags()
	flags.StringVarP(&tunServerListenAddress, "tunnel-listen", "", "", "server tunnel listening address")
	flags.Int64VarP(&xorCryptSeed, "crypt-key", "k", 98545715754651, "crypt key")
}
