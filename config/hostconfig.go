package config

import (
	"myTool/logger"
	"os"

	"gopkg.in/yaml.v2"
)

var HostConfig *Config

// initConfig 初始化配置文件
func initConfig() {
	var examHost Config
	examHost.Host = make([]HostConf, 3)
	examHost.Host[0] = HostConf{
		IP:       "示例节点1-IP",
		Port:     22,
		Username: "节点SSH用户名",
		Password: "节点SSH密码",
	}
	examHost.Host[1] = examHost.Host[0]
	examHost.Host[1].IP = "示例节点2-IP"
	examHost.Host[2] = examHost.Host[0]
	examHost.Host[2].IP = "示例节点3-IP"
	examHost.Ingress.Subnet = "172.29.0.1/20"
	examHost.Ingress.Gateway = "172.29.0.254"
	examHost.Gwbridge.Subnet = "172.30.0.1/24"
	examHost.Gwbridge.Gateway = "172.30.0.254"
	examHost.BIP = "172.31.0.1/24"

	confStr, err := yaml.Marshal(examHost)
	if err != nil {
		panic(err)
	}

	f, err := os.Create("config/config.yml")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	_, err = f.Write([]byte(confStr))
	if err != nil {
		panic(err)
	}
}

// LoadHostConfig 加载配置文件
func GetHostConfig() *Config {
	if _, err := os.Stat("config/config.yml"); os.IsNotExist(err) {
		if _, err := os.Stat("config"); os.IsNotExist(err) {
			os.Mkdir("config", 0775)
		}
		// 备份文件名为services.json_时间戳
		logger.SugarLogger.Infoln("配置文件不存在")
		initConfig()
		logger.SugarLogger.Infoln("示例配置文件 config/config.yml 已经生成, 请根据实际情况修改. ")
		logger.SugarLogger.Infoln("本次运行将直接退出, 修改正确后再次运行本程序. ")
		return nil
	}

	content, err := os.ReadFile("./config/config.yml")
	if err != nil {
		panic(err)
	}
	// 解析配置文件
	err = yaml.Unmarshal(content, &HostConfig)
	if err != nil {
		panic(err)
	}
	return HostConfig
}
