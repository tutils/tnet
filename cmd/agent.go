package cmd

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/spf13/cobra"
	"github.com/tutils/tnet/crypt/xor"
	"github.com/tutils/tnet/endpoint/agent"
	"github.com/tutils/tnet/tun"
)

// agentCmd represents the agent command
var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "TCP tunnel agent",
	Long: `Start TCP tunnel agent, For example:
  tnet agent --tunnel-listen=ws://0.0.0.0:8080/stream --crypt-key=816559
  tnet agent --tunnel-connect=ws://proxy-server:8080/stream --crypt-key=816559`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if tunClientConnectAddress != "" && tunServerListenAddress != "" {
			return fmt.Errorf("cannot specify both --tunnel-connect and --tunnel-listen")
		}
		if tunClientConnectAddress == "" && tunServerListenAddress == "" {
			return fmt.Errorf("must specify either --tunnel-connect or --tunnel-listen")
		}
		var epOpt agent.Option
		var a *agent.Agent
		if tunServerListenAddress != "" {
			// Normal mode: agent waits for proxy to connect
			epOpt = agent.WithTunServer(
				tun.NewServer(
					tun.WithListenAddress(tunServerListenAddress),
				),
			)
		} else {
			// Reverse mode: agent actively connects to proxy
			epOpt = agent.WithTunClient(
				tun.NewClient(
					tun.WithConnectAddress(tunClientConnectAddress),
				),
			)
		}

		a = agent.New(
			epOpt,
			agent.WithTunHandlerNewer(agent.NewTCPAgentTunHandler),
			agent.WithTunCrypt(xor.NewCrypt(xorCryptSeed)),
			agent.WithEnabledExecute(enabledExecute),
		)

		// backoff
		var tempDelay time.Duration
		for {
			// Create a new context for each Serve call
			ctx := context.Background()
			if err := a.Serve(ctx); err != nil {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				log.Println(err)
				time.Sleep(tempDelay)
				continue
			}
		}
	},
}

var (
	enabledExecute bool
)

const defaultXorCryptSeed = 98545715754651

func init() {
	rootCmd.AddCommand(agentCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// agentCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	flags := agentCmd.Flags()
	flags.BoolVarP(&enabledExecute, "enabled-execute", "e", false, "enable remote command execution (SECURITY WARNING: only use with trusted input)")
	flags.StringVarP(&tunServerListenAddress, "tunnel-listen", "", "", "tunnel server listening address")
	flags.StringVarP(&tunClientConnectAddress, "tunnel-connect", "", "", "tunnel client connect address (for reverse mode)")
	flags.Int64VarP(&xorCryptSeed, "crypt-key", "k", defaultXorCryptSeed, "crypt key")

	agentCmd.MarkFlagsMutuallyExclusive("tunnel-connect", "tunnel-listen")
	agentCmd.MarkFlagsOneRequired("tunnel-connect", "tunnel-listen")
}
