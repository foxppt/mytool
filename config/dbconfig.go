package config

import (
	"myTool/logger"
	"os"

	"gopkg.in/yaml.v2"
)

var DBConf *DBConfig

func GetDBConfig(globeOnly bool) *DBConfig {
	if _, err := os.Stat("config/db.yml"); os.IsNotExist(err) {
		if _, err := os.Stat("config"); os.IsNotExist(err) {
			os.Mkdir("config", 0775)
		}
		logger.SugarLogger.Infoln("数据库配置文件不存在")
		if globeOnly {
			initDBConf(globeOnly)
		} else {
			initDBConf(!globeOnly)
		}

		logger.SugarLogger.Infoln("示例配置文件 config/db.yml 已经生成, 请根据实际情况修改. ")
		logger.SugarLogger.Infoln("本次运行将直接退出, 修改正确后再次运行本程序. ")
		return nil
	}
	content, err := os.ReadFile("./config/db.yml")
	if err != nil {
		panic(err)
	}
	// 解析配置文件
	err = yaml.Unmarshal(content, &DBConf)
	if err != nil {
		panic(err)
	}
	return DBConf
}

func initDBConf(globeOnly bool) {
	var examDBConf DBConfig
	examConn := Mysql{
		Host:   "数据库ip",
		Port:   "数据库端口",
		DBName: "数据库库名",
		User:   "数据库用户名",
		Passwd: "数据库密码",
	}
	if globeOnly {
		examDBConf.Globe = examConn
		examDBConf.ServiceCenter = Mysql{}
		examDBConf.ServiceProxy = Mysql{}
	} else {
		examDBConf.Globe = examConn
		examDBConf.ServiceCenter = examConn
		examDBConf.ServiceProxy = examConn
	}

	confStr, err := yaml.Marshal(examDBConf)
	if err != nil {
		panic(err)
	}
	f, err := os.Create("config/db.yml")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	_, err = f.Write([]byte(confStr))
	if err != nil {
		panic(err)
	}
}
