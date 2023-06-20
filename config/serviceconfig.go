package config

import (
	"encoding/json"
	"myTool/logger"
	"os"
)

var serviceConfig *[]ServiceConfig

func GetSvcConfig() *[]ServiceConfig {
	data, err := os.ReadFile("services.json")
	if err != nil {
		logger.SugarLogger.Errorln(err)
		os.Exit(1)
	}
	err = json.Unmarshal(data, &serviceConfig)
	if err != nil {
		logger.SugarLogger.Errorln(err)
	}

	if serviceConfig == nil {
		logger.SugarLogger.Warnln("未在service.json中解析到服务配置, 程序将退出 ")
		return nil
	}
	return serviceConfig
}
