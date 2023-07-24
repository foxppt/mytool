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

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

// expandIngressCmd represents the expandIngress command
var expandIngressCmd = &cobra.Command{
	Use:   "expandIngress",
	Short: "扩展GeoGlobe Server Swarm 的 Ingress网络",
	Long: `expandIngress: 
  扩展GeoGlobe Server Swarm 的 Ingress网络: 
  第一次运行时程序会在当前目录初始化一个config/config.yml和config/db.yml文件, 
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
		dbConf := config.GetDBConfig()
		if dbConf == nil {
			os.Exit(0)
		}
		dbs := &operation.Databases{}
		dbs.Globe, err = dbs.InitDB(&dbConf.Globe)
		if err != nil {
			logger.SugarLogger.Panicln(err)
		}

		ipaconfig := network.IPAMConfig{
			Subnet:  hostConfig.Ingress.Subnet,
			Gateway: hostConfig.Ingress.Gateway,
		}
		netConf := types.NetworkCreate{
			Driver:     "overlay",
			Scope:      "swarm",
			EnableIPv6: false,
			IPAM: &network.IPAM{
				Driver: "default",
				Config: []network.IPAMConfig{
					ipaconfig,
				},
			},
			Internal:   false,
			Attachable: false,
			Ingress:    true,
			ConfigOnly: false,
			ConfigFrom: &network.ConfigReference{
				Network: "",
			},
			Options: map[string]string{
				"com.docker.network.driver.overlay.vxlanid_list": "4098",
				"com.docker.network.mtu":                         "1400",
			},
			Labels: map[string]string{},
		}

		operation.RecordSvc(ctx, dockerClient, hostConfig, true, dbs, "services.json")
		operation.DelService(ctx, dockerClient)
		operation.DelNetwork(ctx, dockerClient, hostConfig, "ingress", true)
		operation.BuildNetwork(ctx, dockerClient, hostConfig, "ingress", netConf, false)

		serviceConfig := config.GetSvcConfig("services.json")
		if serviceConfig == nil {
			logger.SugarLogger.Panicln("读取service配置失败")
		}
		operation.RebuildSvc(ctx, dockerClient, serviceConfig)
	},
}

func init() {
	rootCmd.AddCommand(expandIngressCmd)
}
