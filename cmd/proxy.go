package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/spf13/cobra"
	"github.com/tutils/tnet/counter/period"
	"github.com/tutils/tnet/crypt/xor"
	"github.com/tutils/tnet/endpoint/proxy"
	"github.com/tutils/tnet/tun"
)

// proxyCmd represents the proxy command
var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "TCP tunnel proxy",
	Long: `Start TCP tunnel proxy, For example:
  tnet proxy --listen=0.0.0.0:56080 --connect=127.0.0.1:3128 --tunnel-connect=ws://123.45.67.89:8080/stream --crypt-key=816559
  tnet proxy --tunnel-listen=ws://0.0.0.0:8080/stream --connect=127.0.0.1:3128 --crypt-key=816559`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if tunClientConnectAddress != "" && tunServerListenAddress != "" {
			return fmt.Errorf("cannot specify both --tunnel-connect and --tunnel-listen")
		}
		if tunClientConnectAddress == "" && tunServerListenAddress == "" {
			return fmt.Errorf("must specify either --tunnel-connect or --tunnel-listen")
		}

		var epOpt proxy.Option
		var p *proxy.Proxy
		if tunClientConnectAddress != "" {
			// Normal mode: proxy actively connects to agent
			epOpt = proxy.WithTunClient(
				tun.NewClient(
					tun.WithConnectAddress(tunClientConnectAddress),
				),
			)
		} else {
			// Reverse mode: proxy waits for agent to connect
			epOpt = proxy.WithTunServer(
				tun.NewServer(
					tun.WithListenAddress(tunServerListenAddress),
				),
			)
		}

		p = proxy.New(
			epOpt,
			proxy.WithTunHandlerNewer(proxy.NewTCPProxyTunHandler),
			proxy.WithListenAddress(listenAddress),
			proxy.WithConnectAddress(connectAddress),
			proxy.WithTunCrypt(xor.NewCrypt(xorCryptSeed)),
			proxy.WithDownloadCounter(period.NewPeriodCounter(time.Second)),
			proxy.WithUploadCounter(period.NewPeriodCounter(time.Second)),
		)

		// backoff
		var tempDelay time.Duration
		for {
			if err := p.Serve(); err != nil {
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
	listenAddress  string
	connectAddress string
)

func init() {
	rootCmd.AddCommand(proxyCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// proxyCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// proxyCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	flags := proxyCmd.Flags()
	flags.StringVarP(&listenAddress, "listen", "l", "0.0.0.0:56080", "proxy listen address")
	flags.StringVarP(&connectAddress, "connect", "c", "127.0.0.1:3128", "agent connect address")
	flags.StringVarP(&tunClientConnectAddress, "tunnel-connect", "", "", "tunnel client connect address")
	flags.StringVarP(&tunServerListenAddress, "tunnel-listen", "", "", "tunnel server listening address (for reverse mode)")
	flags.Int64VarP(&xorCryptSeed, "crypt-key", "k", 98545715754651, "crypt key")
}
