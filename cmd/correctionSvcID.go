/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"myTool/config"
	"myTool/operation"
	"os"

	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

// correctionSvcIDCmd represents the correctionSvcID command
var correctionSvcIDCmd = &cobra.Command{
	Use:   "correctionSvcID",
	Short: "更正GeoGlobe 服务ID",
	Long: `correctionSvcID:
根据当前swarm中现有service, 更正数据库记录中的DOCKERID, 
以解决服务编辑和删除相关问题`,
	Run: func(cmd *cobra.Command, args []string) {
		var ctx = context.Background()
		dockerClient, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation(), client.WithVersion(""))
		if err != nil {
			panic(err)
		}

		dbConf := config.GetDBConfig()
		if dbConf == nil {
			os.Exit(0)
		}
		operation.UpdateDBServiceID(ctx, dockerClient, dbConf)
	},
}

func init() {
	rootCmd.AddCommand(correctionSvcIDCmd)
}
