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

var svcConfPath1 string

// rebuildSvcCmd represents the rebuildSvc command
var rebuildSvcCmd = &cobra.Command{
	Use:   "rebuildSvc",
	Short: "重建service",
	Long: `rebuildSvc: 
  根据当前目录的services.json文件
  重建service`,
	Run: func(cmd *cobra.Command, args []string) {
		var ctx = context.Background()
		dockerClient, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation(), client.WithVersion(""))
		if err != nil {
			panic(err)
		}
		if len(args) == 1 {
			serviceConfig := config.GetSvcConfig(svcConfPath1)
			if serviceConfig == nil {
				logger.SugarLogger.Panicln("读取service配置失败")
			}
			swarmopt.RebuildSvc(ctx, dockerClient, serviceConfig)
		} else {
			logger.SugarLogger.Errorln("不允许指定多个配置文件! ")
		}
	},
}

func init() {
	rootCmd.AddCommand(rebuildSvcCmd)
	recordeSvcCmd.Flags().StringVar(&svcConfPath1, "config", "services.json", "服务配置文件, 默认为当前目录 services.json")
}
