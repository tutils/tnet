package cmd

import (
	"log"
	"net/http"

	"github.com/spf13/cobra"
)

// httpsrvCmd represents the agent command
var httpsrvCmd = &cobra.Command{
	Use:   "httpsrv",
	Short: "HTTP Server",
	Long: `Start HTTP Server, For example:
  tnet httpsrv --listen 8080`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fileServer := http.FileServer(http.Dir("."))
		http.Handle("/", fileServer)
		log.Println("listening on", listenAddress)
		return http.ListenAndServe(listenAddress, nil)
	},
}

func init() {
	rootCmd.AddCommand(httpsrvCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// httpsrvCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	flags := httpsrvCmd.Flags()
	flags.StringVarP(&listenAddress, "listen", "l", "0.0.0.0:8080", "http server listen address")
}
