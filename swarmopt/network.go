package swarmopt

import (
	"context"
	"myTool/config"
	"myTool/logger"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

// DelIngress 删除Ingress网络
func DelIngress(ctx context.Context, dockerClient *client.Client, config *config.Config) {
	networks, err := dockerClient.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		panic(err)
	}
	for _, network := range networks {
		if network.Name == "ingress" {
			logger.SugarLogger.Infoln("ingress 网络存在 ")
			err = dockerClient.NetworkRemove(ctx, "ingress")
			if err != nil {
				panic(err)
			}
			logger.SugarLogger.Infoln("ingress 已经被移除 ")
		}
	}
	logger.SugarLogger.Infoln("正在重启dockerd, 请等待...")
	reloadDocker(config)
}

// RebuildIngress 根据配置文件重建Ingress网络
func RebuildIngress(ctx context.Context, dockerClient *client.Client, config *config.Config) {
	// 创建一个overlay网络，名为ingress，subnet为10.0.0.1/20
	networkName := "ingress"
	ipaconfig := network.IPAMConfig{
		Subnet:  config.Ingress.Subnet,
		Gateway: config.Ingress.Gateway,
	}
	networkcrt := types.NetworkCreate{
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
	resp, err := dockerClient.NetworkCreate(ctx, networkName, networkcrt)
	if err != nil {
		panic(err)
	}
	logger.SugarLogger.Infof("Ingress网络%s创建成功 ", resp.ID)
	logger.SugarLogger.Infoln("正在重启dockerd, 请等待...")
	reloadDocker(config)
}
