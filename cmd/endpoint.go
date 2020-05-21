/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tutils/tnet/proxy"
	"github.com/tutils/tnet/tun"
	"log"
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
	flags.StringVarP(&connectAddress, "connect", "c", "127.0.0.1:3128", "connect remote proxied application address")
	flags.StringVarP(&tunServerListenAddress, "tunnel-listen", "", "", "server tunnel listening address")
	flags.Int64VarP(&xorCryptSeed, "crypt-key", "k", 98545715754651, "crypt key")
}