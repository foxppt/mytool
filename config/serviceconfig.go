package config

import (
	"encoding/json"
	"myTool/logger"
	"os"
)

var serviceConfig *[]ServiceConfig

func GetSvcConfig(configPath string) *[]ServiceConfig {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		logger.SugarLogger.Errorln(err)
		os.Exit(1)
	}
	err = json.Unmarshal(data, &serviceConfig)
	if err != nil {
		logger.SugarLogger.Errorln(err)
	}

	if serviceConfig == nil {
		logger.SugarLogger.Warnln("未在", configPath, "中解析到服务配置, 程序将退出 ")
		return nil
	}
	return serviceConfig
}
