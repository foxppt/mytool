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

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

var isGeoglobe string

// editDockerNetCmd represents the editdockernet command
var editDockerNetCmd = &cobra.Command{
	Use:   "editDockerNet",
	Short: "重新指定docker相关网络占用的ip段",
	Long: `重新指定docker相关网络占用的ip段, 以规避网段冲突.
  第一次运行时程序会在当前目录初始化一个 config/config.yml 和 config/db.yml 文件, 
  用户需完善相关配置才能再次运行. `,
	Run: func(cmd *cobra.Command, args []string) {
		var ctx = context.Background()
		dockerClient, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation(), client.WithVersion(""))
		if err != nil {
			panic(err)
		}

		var nodeRoles []struct {
			nodeAddr  string
			isManager bool
		}

		hostConfig := config.GetHostConfig()
		if hostConfig == nil {
			os.Exit(0)
		}

		if isGeoglobe == "true" {
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
		} else if isGeoglobe == "false" {
			swarmopt.RecordSvc(ctx, dockerClient, hostConfig, false, nil, "services.json")
		}

		ipaconfig := network.IPAMConfig{
			Subnet:  hostConfig.Gwbridge.Subnet,
			Gateway: hostConfig.Gwbridge.Gateway,
		}
		netConf := types.NetworkCreate{
			Driver:     "bridge",
			Scope:      "local",
			EnableIPv6: false,
			IPAM: &network.IPAM{
				Driver: "default",
				Config: []network.IPAMConfig{
					ipaconfig,
				},
			},
			Internal:   false,
			Attachable: false,
			Ingress:    false,
			ConfigOnly: false,
			ConfigFrom: &network.ConfigReference{
				Network: "",
			},
			Options: map[string]string{
				"com.docker.network.bridge.enable_icc":           "false",
				"com.docker.network.bridge.enable_ip_masquerade": "true",
				"com.docker.network.bridge.name":                 "docker_gwbridge",
			},
			Labels: map[string]string{},
		}

		swarmopt.DelService(ctx, dockerClient)
		swarmopt.RecordNet(ctx, dockerClient, "userDefinedNet.json")
		for _, host := range hostConfig.Host {
			swarmopt.EditBipConf(host, hostConfig.BIP, true)
			// 判断主从节点
			isLeader, err := swarmopt.GetSwarmNodeRole(ctx, dockerClient, host)
			if err != nil {
				logger.SugarLogger.Panicln("获取节点角色失败:", err)
			}
			nodeRoles = append(nodeRoles, struct {
				nodeAddr  string
				isManager bool
			}{nodeAddr: host.IP, isManager: isLeader})
			// 依次退出swarm
			swarmopt.LeaveSwarm(ctx, dockerClient, host, isLeader)
			// 依次删除docker_gwbridge
			swarmopt.DelNetwork(ctx, dockerClient, hostConfig, "docker_gwbridge", true)
			// 依次创建docker_gwbridge
			swarmopt.BuildNetwork(ctx, dockerClient, hostConfig, "docker_gwbridge", netConf, false)
		}

		// 主节点创建swarm
		swarmopt.InitSwarm(ctx, dockerClient, ":2377")

		// 获取join token
		managerTK, workerTK, err := swarmopt.GetSwarmJoinTK(ctx, dockerClient)
		if err != nil {
			logger.SugarLogger.Panicln(err)
		}
		logger.SugarLogger.Infoln("主节点加入的token为: ", managerTK)
		logger.SugarLogger.Infoln("从节点加入的token为: ", workerTK)
		// 加入swarm
		for _, host := range hostConfig.Host {
			for _, noderole := range nodeRoles {
				if host.IP == noderole.nodeAddr && !noderole.isManager {
					swarmopt.JoinSwarm(host, workerTK)
				} else if host.IP == noderole.nodeAddr && noderole.isManager {
					swarmopt.JoinSwarm(host, managerTK)
				}
			}
		}

		userDefinedNets := config.GetUserNetConf("userDefinedNet.json")
		for _, userDefinedNet := range *userDefinedNets {
			userNetConf := types.NetworkCreate{
				CheckDuplicate: true,
				Driver:         userDefinedNet.Driver,
				Scope:          userDefinedNet.Scope,
				EnableIPv6:     userDefinedNet.EnableIPv6,
				IPAM:           &userDefinedNet.IPAM,
				Internal:       userDefinedNet.Internal,
				Attachable:     userDefinedNet.Attachable,
				Ingress:        userDefinedNet.Ingress,
				ConfigOnly:     userDefinedNet.ConfigOnly,
				ConfigFrom:     &userDefinedNet.ConfigFrom,
				Options:        userDefinedNet.Options,
				Labels:         userDefinedNet.Labels,
			}
			swarmopt.BuildNetwork(ctx, dockerClient, hostConfig, userDefinedNet.Name, userNetConf, false)
		}

		// 重建service
		serviceConfig := config.GetSvcConfig("services.json")
		if serviceConfig != nil {
			swarmopt.RebuildSvc(ctx, dockerClient, serviceConfig)
		}
	},
}

func init() {
	rootCmd.AddCommand(editDockerNetCmd)
	editDockerNetCmd.Flags().StringVarP(&isGeoglobe, "isgeoglobe", "g", "", "是否存在geoglobe服务")
}
