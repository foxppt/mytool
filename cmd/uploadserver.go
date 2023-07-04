/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var port uint16
var downloadPath string
var uploadPath string

// uploadserverCmd represents the uploadserver command
var simpleserverCmd = &cobra.Command{
	Use:   "simpleserver",
	Short: "启动一个提供上传功能的http服务",
	Long: `simpleserver: 
  启动一个提供上传功能的http服务: 
  本命令存在一个参数 -p 允许用户指定端口, 格式为-p port`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("空函数, 还没来得及写. ")
	},
}

func init() {
	rootCmd.AddCommand(simpleserverCmd)
	simpleserverCmd.Flags().Uint16VarP(&port, "prot", "p", 23333, "启动可以指定端口")
	simpleserverCmd.Flags().StringVarP(&downloadPath, "downloadpath", "d", "./downloud", "指定http的下载文件路径")
	simpleserverCmd.Flags().StringVarP(&uploadPath, "uploadpath", "u", "./upload", "指定http的上传文件路径")
}
