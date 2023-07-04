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

// constraintSvcCmd represents the constraintSvc command
var constraintSvcCmd = &cobra.Command{
	Use:   "constraintSvc",
	Short: "约束GeoGlobe Server 服务到初次投递节点",
	Long: `constraintSvc: 
  约束GeoGlobe Server 服务到初次投递节点, 可以预防Overlay跨节点通信存在问题时服务无法访问:
  第一次运行时程序会在当前目录初始化一个 config/config.yml 和 config/db.yml 文件, 
  用户需完善相关配置才能再次运行. `,
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

		dbConf := config.GetDBConfig(true)
		if dbConf == nil {
			os.Exit(0)
		}
		dbs := &swarmopt.Databases{}
		dbs.Globe, err = dbs.InitDB(&dbConf.Globe)
		if err != nil {
			logger.SugarLogger.Panicln(err)
		}

		swarmopt.RecordSvc(ctx, dockerClient, hostConfig, true, dbs, "services.json")

		serviceConfig := config.GetSvcConfig("services.json")
		if serviceConfig == nil {
			logger.SugarLogger.Panicln("读取service配置失败")
		}
		swarmopt.UnConstraitAll(ctx, dockerClient)
		swarmopt.ConstraitService(ctx, dockerClient, serviceConfig)
	},
}

func init() {
	rootCmd.AddCommand(constraintSvcCmd)
}
