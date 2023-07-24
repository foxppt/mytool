/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"myTool/config"
	"myTool/logger"
	"myTool/operation"
	"os"

	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

var svcConf string
var isGlobe string

// recordeSvcCmd represents the recodeSvc command
var recordeSvcCmd = &cobra.Command{
	Use:   "recordeSvc",
	Short: "记录当前环境GeoGlobe Server的服务信息",
	Long: `recordeSvc: 
  记录当前环境GeoGlobe Server的服务信息: 
  这个子命令没有任何其他参数; 
  程序默认会在同级目录生成一个services.json文件(也可以指定), 如果同名文件存在会被备份. `,
	Run: func(cmd *cobra.Command, args []string) {
		var ctx = context.Background()
		dockerClient, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation(), client.WithVersion(""))
		if err != nil {
			panic(err)
		}

		if isGlobe == "true" {
			hostConfig := config.GetHostConfig()
			if hostConfig == nil {
				os.Exit(0)
			}

			dbConf := config.GetDBConfig()
			if dbConf == nil {
				os.Exit(0)
			}
			dbs := &operation.Databases{}
			dbs.Globe, err = dbs.InitDB(&dbConf.Globe)
			if err != nil {
				logger.SugarLogger.Panicln(err)
			}
			operation.RecordSvc(ctx, dockerClient, hostConfig, true, dbs, svcConf)
		} else {
			hostConfig := config.GetHostConfig()
			if hostConfig == nil {
				os.Exit(0)
			}
			operation.RecordSvc(ctx, dockerClient, hostConfig, false, nil, svcConf)
		}
	},
}

func init() {
	rootCmd.AddCommand(recordeSvcCmd)
	recordeSvcCmd.Flags().StringVarP(&svcConf, "config", "c", "services.json", "服务配置文件存储路径")
	recordeSvcCmd.Flags().StringVarP(&isGlobe, "isgeoglobe", "g", "", "是否存在geoglobe服务")
}
