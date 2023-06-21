/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"myTool/config"
	"myTool/logger"
	"myTool/swarmopt"
	"os"

	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

var svcConf string

// recordeSvcCmd represents the recodeSvc command
var recordeSvcCmd = &cobra.Command{
	Use:   "recordeSvc",
	Short: "记录当前环境GeoGlobe Server的服务信息",
	Long: `recordeSvc: 
  记录当前环境GeoGlobe Server的服务信息: 
  这个子命令没有任何其他参数; 
  程序会在同级目录生成一个services.json文件, 如果同名文件存在会被备份. `,
	Run: func(cmd *cobra.Command, args []string) {
		var ctx = context.Background()
		dockerClient, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation(), client.WithVersion(""))
		if err != nil {
			panic(err)
		}
		hostConfig := config.GetHostConfig()
		if hostConfig == nil {
			os.Exit(0)
		}
		db, err := swarmopt.InitDB(hostConfig)
		if err != nil {
			logger.SugarLogger.Fatalln(err)
		}
		swarmopt.RecordSvc(ctx, dockerClient, hostConfig, db, svcConf)
	},
}

func init() {
	rootCmd.AddCommand(recordeSvcCmd)
	recordeSvcCmd.Flags().StringVar(&svcConf, "config", "services.json", "服务配置文件, 默认为当前目录 services.json")
}
