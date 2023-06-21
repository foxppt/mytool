package config

import (
	"encoding/json"
	"myTool/logger"
	"os"
)

var serviceConfig *[]ServiceConfig

func GetSvcConfig(configpath string) *[]ServiceConfig {
	data, err := os.ReadFile(configpath)
	if err != nil {
		logger.SugarLogger.Errorln(err)
		os.Exit(1)
	}
	err = json.Unmarshal(data, &serviceConfig)
	if err != nil {
		logger.SugarLogger.Errorln(err)
	}

	if serviceConfig == nil {
		logger.SugarLogger.Warnln("未在", configpath, "中解析到服务配置, 程序将退出 ")
		return nil
	}
	return serviceConfig
}
