package swarmopt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"myTool/config"
	"myTool/logger"
	"strings"

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
		_, err = execCMD(host.IP, host.Port, host.Username, host.Password, cmd)
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
		_, err = execCMD(host.IP, host.Port, host.Username, host.Password, cmd)
		if err != nil {
			logger.SugarLogger.Panicln(host.IP, "重启dockerd失败:", err)
		}
		logger.SugarLogger.Infoln(host.IP, "重启dockerd成功. ")
	}
}

// 修改docker配置bip
func EditBipConf(host config.HostConf, bipStr string) {
	logger.SugarLogger.Infoln("修改", host.IP, "主机的docker的bip配置: ")
	var cmd string
	// 判断文件是否存在
	cmd = `[ -f /etc/docker/daemon.json ] && echo true || echo false`
	fileExists, err := execCMD(host.IP, host.Port, host.Username, host.Password, cmd)
	if err != nil {
		logger.SugarLogger.Panicln(host.IP, "获取/etc/docker/daemon.json文件状态失败. ")
	}

	if strings.Contains(fileExists, "false") {
		// 文件不存在,创建文件
		logger.SugarLogger.Infoln("/etc/docker/daemon.json不存在, 将被创建. ")
		cmd = "touch /etc/docker/daemon.json && " + "echo " + "{\\\"bip\\\": " + "\\\"" + bipStr + "\\\"" + "} " + "> /etc/docker/daemon.json"
		// logger.SugarLogger.Infoln("创建命令为: ", cmd)
		resp, err := execCMD(host.IP, host.Port, host.Username, host.Password, cmd)
		if err != nil {
			logger.SugarLogger.Panicln(host.IP, "重启docker失败", resp)
		}
		logger.SugarLogger.Infoln("/etc/docker/daemon.json创建成功. ")

		logger.SugarLogger.Infoln("正在重启dockerd, 请等待...")
		_, err = execCMD(host.IP, host.Port, host.Username, host.Password, "systemctl restart docker")
		if err != nil {
			logger.SugarLogger.Panicln(host.IP, "重启docker失败")
		}

	} else if strings.Contains(fileExists, "true") {
		// 文件存在,解析文件
		logger.SugarLogger.Infoln("/etc/docker/daemon.json存在")
		cmd = "cat /etc/docker/daemon.json"
		resp, err := execCMD(host.IP, host.Port, host.Username, host.Password, cmd)
		if err != nil {
			logger.SugarLogger.Panicln("获取/etc/docker/daemon.json 内容失败")
		}
		content := []byte(resp)
		var config map[string]interface{}
		err = json.Unmarshal(content, &config)
		if err != nil {
			logger.SugarLogger.Panicln(err)
		}

		// 判断bip key是否存在
		_, ok := config["bip"]
		if !ok {
			// bip key不存在,追加bip key
			logger.SugarLogger.Infoln("/etc/docker/daemon.json中bip配置不存在")
			config["bip"] = bipStr
			content, err := json.Marshal(config)
			if err != nil {
				logger.SugarLogger.Panicln(err)
			}

			content = bytes.TrimSpace(content)
			content = bytes.Replace(content, []byte("\n"), []byte("\n    "), -1)

			cmd = fmt.Sprintf("echo '%s' > /etc/docker/daemon.json", string(content))
			_, err = execCMD(host.IP, host.Port, host.Username, host.Password, cmd)
			if err != nil {
				logger.SugarLogger.Panicln(host.IP, "修改/etc/docker/daemon.json失败")
			}

			logger.SugarLogger.Infoln("正在重启dockerd, 请等待...")
			_, err = execCMD(host.IP, host.Port, host.Username, host.Password, "systemctl restart docker")
			if err != nil {
				logger.SugarLogger.Panicln(host.IP, "重启docker失败")
			}
			logger.SugarLogger.Infoln(`bip配置为"bip:"`, bipStr)

		} else {
			// bip key存在,判断value
			logger.SugarLogger.Infoln("/etc/docker/daemon.json中bip配置存在")
			bip := config["bip"].(string)
			if bip == bipStr {
				// value等于172.31.1.1/24,不做任何处理
				logger.SugarLogger.Infoln("/etc/docker/daemon.json中bip配置与config/config.yml中指定一致. ")
				logger.SugarLogger.Infoln("正在重启dockerd, 请等待...")
				_, err = execCMD(host.IP, host.Port, host.Username, host.Password, "systemctl restart docker")
				if err != nil {
					logger.SugarLogger.Panicln(host.IP, "重启docker失败")
				}
			} else {
				// value不等于172.31.1.1/24,修改value
				logger.SugarLogger.Infoln("/etc/docker/daemon.json中bip配置与config/config.yml中指定不一致. ")
				config["bip"] = bipStr
				content, err := json.Marshal(config)
				if err != nil {
					logger.SugarLogger.Panicln(err)
				}
				content = bytes.TrimSpace(content)
				content = bytes.Replace(content, []byte("\n"), []byte("\n    "), -1)

				cmd = fmt.Sprintf("echo '%s' > /etc/docker/daemon.json", string(content))
				_, err = execCMD(host.IP, host.Port, host.Username, host.Password, cmd)
				if err != nil {
					logger.SugarLogger.Panicln(host.IP, "修改/etc/docker/daemon.json失败")
				}

				logger.SugarLogger.Infoln("正在重启dockerd, 请等待...")
				_, err = execCMD(host.IP, host.Port, host.Username, host.Password, "systemctl restart docker")
				if err != nil {
					logger.SugarLogger.Panicln(host.IP, "重启docker失败")
				}
				logger.SugarLogger.Infoln("/etc/docker/daemon.json中bip配置已被修改. ")
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
