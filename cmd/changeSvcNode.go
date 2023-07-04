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

var svcName string
var nodeTarget string

// changeSvcNodeCmd represents the changeSvcNode command
var changeSvcNodeCmd = &cobra.Command{
	Use:   "changeSvcNode",
	Short: "更换GeoGlobe Server服务的节点",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		var ctx = context.Background()
		dockerClient, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation(), client.WithVersion(""))
		if err != nil {
			panic(err)
		}
		dbConf := config.GetDBConfig(false)
		if dbConf == nil {
			os.Exit(0)
		}
		dbs := &swarmopt.Databases{}
		dbs.Globe, err = dbs.InitDB(&dbConf.Globe)
		if err != nil {
			logger.SugarLogger.Panicln(err)
		}

		dbs.ServiceCenter, err = dbs.InitDB(&dbConf.ServiceCenter)
		if err != nil {
			logger.SugarLogger.Panicln(err)
		}

		dbs.ServiceProxy, err = dbs.InitDB(&dbConf.ServiceProxy)
		if err != nil {
			logger.SugarLogger.Panicln(err)
		}

		if svcName != "" && nodeTarget != "" {
			err = swarmopt.ChangeSvcNode(ctx, dockerClient, dbs, svcName, nodeTarget)
			if err != nil {
				logger.SugarLogger.DPanicln(err)
			}
		} else {
			logger.SugarLogger.Infoln("请输入正确的服务名和节点IP")
		}
	},
}

func init() {
	rootCmd.AddCommand(changeSvcNodeCmd)
	changeSvcNodeCmd.Flags().StringVarP(&svcName, "servicename", "s", "", "服务名, 与docker service ls对应的Name字段一致")
	changeSvcNodeCmd.Flags().StringVarP(&nodeTarget, "nodeTarget", "n", "", "节点IP, 与docker node inspect 中对应的Addr字段一致")
}
