package operation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"myTool/config"
	"myTool/logger"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// 修改docker配置bip
func EditBipConf(host config.HostConf, bipStr string, needRestartDocker bool) {
	logger.SugarLogger.Infoln("修改", host.IP, "主机的docker的bip配置: ")
	var cmd string
	// 判断文件是否存在
	logger.SugarLogger.Infoln("判断", host.IP, "主机的docker的daemon.json文件是否存在")
	cmd = `[ -f /etc/docker/daemon.json ] && echo true || echo false`
	fileExists, err := execCMD(&host, cmd)
	if err != nil {
		logger.SugarLogger.Panicln(host.IP, "获取/etc/docker/daemon.json文件状态失败. ")
	}

	if strings.Contains(fileExists, "false") {
		// 文件不存在,创建文件
		logger.SugarLogger.Infoln("/etc/docker/daemon.json不存在, 将被创建. ")
		cmd = "touch /etc/docker/daemon.json && " + "echo " + "{\\\"bip\\\": " + "\\\"" + bipStr + "\\\"" + "} " + "> /etc/docker/daemon.json"
		// logger.SugarLogger.Infoln("创建命令为: ", cmd)
		resp, err := execCMD(&host, cmd)
		if err != nil {
			logger.SugarLogger.Panicln(host.IP, "重启docker失败", resp)
		}
		logger.SugarLogger.Infoln("/etc/docker/daemon.json创建成功, bip被设置为: ", bipStr)

		if needRestartDocker {
			logger.SugarLogger.Infoln("正在重启dockerd, 请等待...")
			_, err = execCMD(&host, "systemctl restart docker")
			if err != nil {
				logger.SugarLogger.Panicln(host.IP, "重启docker失败")
			}
		}
	} else if strings.Contains(fileExists, "true") {
		// 文件存在,解析文件
		logger.SugarLogger.Infoln("/etc/docker/daemon.json存在")
		cmd = "cat /etc/docker/daemon.json"
		resp, err := execCMD(&host, cmd)
		if err != nil {
			logger.SugarLogger.Panicln("获取/etc/docker/daemon.json 内容失败")
		}
		if resp == "" {
			logger.SugarLogger.Infoln("/etc/docker/daemon.json文件为空")
			cmd = "touch /etc/docker/daemon.json && " + "echo " + "{\\\"bip\\\": " + "\\\"" + bipStr + "\\\"" + "} " + "> /etc/docker/daemon.json"
			resp, err := execCMD(&host, cmd)
			if err != nil {
				logger.SugarLogger.Panicln(host.IP, "重启docker失败", resp)
			}
			logger.SugarLogger.Infoln("bip被设置为: ", bipStr)
			if needRestartDocker {
				logger.SugarLogger.Infoln("正在重启dockerd, 请等待...")
				_, err = execCMD(&host, "systemctl restart docker")
				if err != nil {
					logger.SugarLogger.Panicln(host.IP, "重启docker失败")
				}
				return
			}
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
			_, err = execCMD(&host, cmd)
			if err != nil {
				logger.SugarLogger.Panicln(host.IP, "修改/etc/docker/daemon.json失败")
			}

			if needRestartDocker {
				logger.SugarLogger.Infoln("正在重启dockerd, 请等待...")
				_, err = execCMD(&host, "systemctl restart docker")
				if err != nil {
					logger.SugarLogger.Panicln(host.IP, "重启docker失败")
				}
			}
			logger.SugarLogger.Infoln(`bip配置为"bip:"`, bipStr)

		} else {
			// bip key存在,判断value
			logger.SugarLogger.Infoln("/etc/docker/daemon.json中bip配置存在")
			bip := config["bip"].(string)
			if bip == bipStr {
				// value等于172.31.1.1/24,不做任何处理
				logger.SugarLogger.Infoln("/etc/docker/daemon.json中bip配置与config/config.yml中指定一致. ")
				logger.SugarLogger.Infoln("为确保配置生效, 正在重启dockerd, 请等待...")
				_, err = execCMD(&host, "systemctl restart docker")
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
				_, err = execCMD(&host, cmd)
				if err != nil {
					logger.SugarLogger.Panicln(host.IP, "修改/etc/docker/daemon.json失败")
				}

				if needRestartDocker {
					logger.SugarLogger.Infoln("正在重启dockerd, 请等待...")
					_, err = execCMD(&host, "systemctl restart docker")
					if err != nil {
						logger.SugarLogger.Panicln(host.IP, "重启docker失败")
					}
				}
				logger.SugarLogger.Infoln("/etc/docker/daemon.json中bip配置已被修改, 现在为: ", bipStr)
			}
		}
	}

}

// 删除网络
func DelNetwork(ctx context.Context, dockerClient *client.Client, config *config.Config, netName string, needRestartDocker bool) {
	networks, err := dockerClient.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		panic(err)
	}

	for _, network := range networks {
		if network.Name == netName {
			logger.SugarLogger.Infoln(netName, "网络存在 ")
			err = dockerClient.NetworkRemove(ctx, netName)
			if err != nil {
				panic(err)
			}
			logger.SugarLogger.Infoln(netName, "已经被移除 ")
		}
	}
	if needRestartDocker {
		logger.SugarLogger.Infoln("正在重启dockerd, 请等待...")
		cmd := "systemctl restart docker"
		for _, host := range config.Host {
			_, err = execCMD(&host, cmd)
			if err != nil {
				logger.SugarLogger.Panicln(host.IP, "重启dockerd失败:", err)
			}
			logger.SugarLogger.Infoln(host.IP, "重启dockerd成功. ")
		}
	}
}

// 创建网络
func BuildNetwork(ctx context.Context, dockerClient *client.Client, config *config.Config, netName string, netConf types.NetworkCreate, needRestartDocker bool) {
	resp, err := dockerClient.NetworkCreate(ctx, netName, netConf)
	if err != nil {
		panic(err)
	}
	logger.SugarLogger.Infof("%s网络%s创建成功 ", netName, resp.ID)
	if needRestartDocker {
		logger.SugarLogger.Infoln("正在重启dockerd, 请等待...")
		cmd := "systemctl restart docker"
		for _, host := range config.Host {
			_, err = execCMD(&host, cmd)
			if err != nil {
				logger.SugarLogger.Panicln(host.IP, "重启dockerd失败:", err)
			}
			logger.SugarLogger.Infoln(host.IP, "重启dockerd成功. ")
		}
	}
}

// 记录自定义网络相关的信息
func RecordNet(ctx context.Context, dockerClient *client.Client, netConf string) {
	netStruct := []types.NetworkResource{}
	networks, err := dockerClient.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		logger.SugarLogger.Panicln(err)
	}
	for _, net := range networks {
		if net.Name != "bridge" && net.Name != "docker_gwbridge" && net.Name != "host" && net.Name != "ingress" && net.Name != "none" {
			netStruct = append(netStruct, net)
		}
	}
	// 将 svcStructs 编码为JSON并将其写入userDefinedNet.json
	jsonBytes, err := json.MarshalIndent(netStruct, "", "  ")
	if err != nil {
		panic(err)
	}
	// 判断userDefinedNet.json是否存在，如果存在就备份，备份文件名为userDefinedNet.json_时间戳
	timestamp := time.Now().Unix()
	if _, err := os.Stat(netConf); !os.IsNotExist(err) {
		// 备份文件名为userDefinedNet.json_时间戳
		logger.SugarLogger.Infoln(netConf, "已经存在")
		err := os.Rename(netConf, fmt.Sprintf(netConf+"_%d", timestamp))
		if err != nil {
			logger.SugarLogger.Errorln(netConf, "备份文件失败：", err)
			return
		}
		logger.SugarLogger.Infoln(netConf, "备份, 文件名为: ", fmt.Sprintf(netConf+"_%d", timestamp))
	}

	f, err := os.Create(netConf)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	_, err = f.Write(jsonBytes)
	if err != nil {
		panic(err)
	}
	logger.SugarLogger.Infoln("自定义网络详细信息已被保存到", netConf)
}
