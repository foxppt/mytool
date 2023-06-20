package config

import (
	"myTool/logger"
	"os"

	"gopkg.in/yaml.v2"
)

var HostConfig *Config

// initConfig 初始化配置文件
func initConfig() {
	confStr := `# 集群主机连接信息
host:
  - ip: 示例节点1-IP
    port: 节点SSH端口
    username: 节点SSH用户名
    password: 节点SSH密码
  - ip: 示例节点2-IP
    port: 节点SSH端口
    username: 节点SSH用户名
    password: 节点SSH密码
  # 如果有多个节点可以继续添加, 相反, 如果只有一个节点, 删除第二节(示例节点2)

# Ingress CIDR定义
ingress:
  subnet: 172.29.0.1/20 # Ingress网络CIDR定义, 可以自行修改
  gateway: 172.29.0.254 # Ingress网络网关

# 数据库连接信息
mysql:
  host: 数据库ip
  port: 数据库端口
  dbname: 数据库库名
  user: 数据库用户名
  passwd: 数据库密码`

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
