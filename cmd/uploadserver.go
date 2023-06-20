/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// uploadserverCmd represents the uploadserver command
var uploadserverCmd = &cobra.Command{
	Use:   "uploadserver",
	Short: "启动一个提供上传功能的http服务",
	Long: `uploadserver: 
  启动一个提供上传功能的http服务: 
  本命令存在一个参数 -p 允许用户指定端口, 格式为-p port`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("uploadserver called")
	},
}

func init() {
	rootCmd.AddCommand(uploadserverCmd)
}
