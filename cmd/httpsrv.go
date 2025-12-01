package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tutils/tnet/httpsrv"
)

// httpsrvCmd represents the agent command
var httpsrvCmd = &cobra.Command{
	Use:   "httpsrv",
	Short: "HTTP文件服务器",
	Long: `启动HTTP文件服务器，支持文件浏览、上传和下载功能。例如：
  tnet httpsrv --listen 8080`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 调用httpsrv包中的StartServer函数，使用正确的包导入
		return httpsrv.StartServer(listenAddress)
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
