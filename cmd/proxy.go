package cmd

import (
	"log"
	"time"

	"github.com/spf13/cobra"
	"github.com/tutils/tnet/counter/period"
	"github.com/tutils/tnet/crypt/xor"
	"github.com/tutils/tnet/endpoint"
	"github.com/tutils/tnet/tun"
	"github.com/tutils/tnet/tun/mqtt"
)

var (
	tunClientConnectAddress string
	listenAddress           string
	connectAddress          string
)

// proxyCmd represents the proxy command
var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "TCP tunnel proxy",
	Long: `Start TCP tunnel proxy, For example:
  tnet proxy --listen=0.0.0.0:56080 --connect=127.0.0.1:3128 --tunnel-connect=ws://123.45.67.89:8080/stream --crypt-key=816559`,
	RunE: func(cmd *cobra.Command, args []string) error {
		p := endpoint.NewProxy(
			endpoint.WithTunClient(
				mqtt.NewClient(
					tun.WithConnectAddress(tunClientConnectAddress),
					tun.WithClientHandler(endpoint.NewTunClientHandler()),
				),
			),
			endpoint.WithListenAddress(listenAddress),
			endpoint.WithConnectAddress(connectAddress),
			endpoint.WithTunClientCrypt(xor.NewCrypt(xorCryptSeed)),
			endpoint.WithDownloadCounter(period.NewPeriodCounter(time.Second)),
			endpoint.WithUploadCounter(period.NewPeriodCounter(time.Second)),
		)

		// backoff
		var tempDelay time.Duration
		for {
			if err := p.DialAndServe(); err != nil {
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
	flags.StringVarP(&connectAddress, "connect", "c", "127.0.0.1:3128", "endpoint connect address")
	flags.StringVarP(&tunClientConnectAddress, "tunnel-connect", "", "", "client tunnel connect address")
	flags.Int64VarP(&xorCryptSeed, "crypt-key", "k", 98545715754651, "crypt key")
}
