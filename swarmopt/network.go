package swarmopt

import (
	"context"
	"encoding/json"
	"myTool/config"
	"myTool/logger"
	"os"

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
	cmd := "systemctl restart docker"
	for _, host := range config.Host {
		err = execCMD(host.IP, host.Port, host.Username, host.Password, cmd)
		if err != nil {
			logger.SugarLogger.Panicln(host.IP, "重启dockerd失败:", err)
		}
		logger.SugarLogger.Infoln(host.IP, "重启dockerd成功. ")
	}
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
	cmd := "systemctl restart docker"
	for _, host := range config.Host {
		err = execCMD(host.IP, host.Port, host.Username, host.Password, cmd)
		if err != nil {
			logger.SugarLogger.Panicln(host.IP, "重启dockerd失败:", err)
		}
		logger.SugarLogger.Infoln(host.IP, "重启dockerd成功. ")
	}
}

// 修改docker配置bip
func EditBipConf(host config.HostConf, bip string) {
	if _, err := os.Stat("/etc/docker/daemon.json"); os.IsNotExist(err) {
		// 配置文件不存在,新建文件
		file, err := os.Create("/etc/docker/daemon.json")
		if err != nil {
			logger.SugarLogger.DPanicln(err)
		}
		defer file.Close()
		// 添加bip配置
		_, err = file.WriteString(`{"bip": ` + bip + `}`)
		if err != nil {
			logger.SugarLogger.DPanicln(err)
		}

		err = execCMD(host.IP, host.Port, host.Username, host.Password, "systemctl restart docker")
		if err != nil {
			logger.SugarLogger.Panicln(host.IP, "重启docker失败")
		}
	} else {
		// 文件存在,解析文件
		file, err := os.Open("/etc/docker/daemon.json")
		if err != nil {
			logger.SugarLogger.DPanicln(err)
		}
		defer file.Close()

		decoder := json.NewDecoder(file)
		var dockerConfig map[string]string
		err = decoder.Decode(&dockerConfig)
		if err != nil {
			logger.SugarLogger.DPanicln(err)
		}

		// 判断bip配置是否存在
		_, ok := dockerConfig["bip"]
		if ok {
			// bip配置存在,判断值是否为"172.28.1.1/24"
			if dockerConfig["bip"] != bip {
				// 值不为"172.28.1.1/24",修改为"172.28.1.1/24"
				dockerConfig["bip"] = bip
				output, err := json.MarshalIndent(dockerConfig, "", "    ")
				if err != nil {
					logger.SugarLogger.DPanicln(err)
				}

				err = os.WriteFile("/etc/docker/daemon.json", output, 0644)
				if err != nil {
					logger.SugarLogger.DPanicln(err)
				}

				err = execCMD(host.IP, host.Port, host.Username, host.Password, "systemctl restart docker")
				if err != nil {
					logger.SugarLogger.Panicln(host.IP, "重启docker失败")
				}
			} else {
				err = execCMD(host.IP, host.Port, host.Username, host.Password, "systemctl restart docker")
				if err != nil {
					logger.SugarLogger.Panicln(host.IP, "重启docker失败")
				}
			}
		} else {
			// 若bip的配置不存在,追加"bip": "172.28.1.1/24"
			dockerConfig["bip"] = bip
			output, err := json.MarshalIndent(dockerConfig, "", "    ")
			if err != nil {
				logger.SugarLogger.DPanicln(err)
			}
			err = os.WriteFile("/etc/docker/daemon.json", output, 0644)
			if err != nil {
				logger.SugarLogger.DPanicln(err)
			}

			err = execCMD(host.IP, host.Port, host.Username, host.Password, "systemctl restart docker")
			if err != nil {
				logger.SugarLogger.Panicln(host.IP, "重启docker失败")
			}
		}
	}
}

// 删除docker_gwbridge
func Delgwbr(ctx context.Context, dockerClient *client.Client, config *config.Config) {
	networks, err := dockerClient.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		panic(err)
	}
	for _, network := range networks {
		if network.Name == "docker_gwbridge" {
			logger.SugarLogger.Infoln("docker_gwbridge 网络存在 ")
			err = dockerClient.NetworkRemove(ctx, "docker_gwbridge")
			if err != nil {
				panic(err)
			}
			logger.SugarLogger.Infoln("docker_gwbridge 已经被移除 ")
		}
	}
	logger.SugarLogger.Infoln("正在重启dockerd, 请等待...")
}

// 重新创建docker_gwbridge
func Rebuildgwbr(ctx context.Context, dockerClient *client.Client, config *config.Config) {
	// 创建一个overlay网络，名为ingress，subnet为10.0.0.1/20
	networkName := "docker_gwbridge"
	ipaconfig := network.IPAMConfig{
		Subnet:  config.Gwbridge.Subnet,
		Gateway: config.Gwbridge.Gateway,
	}
	networkcrt := types.NetworkCreate{
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
	resp, err := dockerClient.NetworkCreate(ctx, networkName, networkcrt)
	if err != nil {
		panic(err)
	}
	logger.SugarLogger.Infof("docker_gwbridge网络%s创建成功 ", resp.ID)
}
