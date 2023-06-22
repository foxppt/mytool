/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"myTool/config"
	"myTool/logger"
	"myTool/swarmopt"

	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

var svcConfPath string

// rebuildSvcCmd represents the rebuildSvc command
var rebuildSvcCmd = &cobra.Command{
	Use:   "rebuildSvc",
	Short: "重建service",
	Long: `rebuildSvc: 
  根据指定的服务配置 (不指定则为当前目录的services.json) 文件
  重建service`,
	Run: func(cmd *cobra.Command, args []string) {
		var ctx = context.Background()
		dockerClient, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation(), client.WithVersion(""))
		if err != nil {
			panic(err)
		}
		serviceConfig := config.GetSvcConfig(svcConfPath)
		if serviceConfig == nil {
			logger.SugarLogger.Panicln("读取service配置失败")
		}
		swarmopt.RebuildSvc(ctx, dockerClient, serviceConfig)
	},
}

func init() {
	rootCmd.AddCommand(rebuildSvcCmd)
	rebuildSvcCmd.Flags().StringVar(&svcConfPath, "config", "services.json", "加载服务配置文件")
}
