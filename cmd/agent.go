package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/tutils/tnet/crypt/xor"
	"github.com/tutils/tnet/endpoint"
	"github.com/tutils/tnet/tun"
	"github.com/tutils/tnet/tun/mqtt"
)

var (
	tunServerListenAddress string
)

// agentCmd represents the agent command
var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "TCP tunnel agent",
	Long: `Start TCP tunnel agent, For example:
  tnet agent --tunnel-listen=ws://0.0.0.0:8080/stream --crypt-key=816559`,
	RunE: func(cmd *cobra.Command, args []string) error {
		e := endpoint.NewAgent(
			endpoint.WithTunServer(
				mqtt.NewServer(
					tun.WithListenAddress(tunServerListenAddress),
					tun.WithServerHandler(endpoint.NewTunServerHandler()),
				),
			),
			endpoint.WithTunServerCrypt(xor.NewCrypt(xorCryptSeed)),
		)
		if err := e.ListenAndServe(); err != nil {
			log.Fatalln(err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(agentCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// agentCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	flags := agentCmd.Flags()
	flags.StringVarP(&tunServerListenAddress, "tunnel-listen", "", "", "server tunnel listening address")
	flags.Int64VarP(&xorCryptSeed, "crypt-key", "k", 98545715754651, "crypt key")
}
