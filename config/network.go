package config

import (
	"encoding/json"
	"myTool/logger"
	"os"

	"github.com/docker/docker/api/types"
)

var userDefinedNets *[]types.NetworkResource

func GetUserNetConf(configPath string) *[]types.NetworkResource {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		logger.SugarLogger.Errorln(err)
		os.Exit(1)
	}
	err = json.Unmarshal(data, &userDefinedNets)
	if err != nil {
		logger.SugarLogger.Errorln(err)
	}

	if userDefinedNets == nil {
		logger.SugarLogger.Warnln("未在", configPath, "中解析到用户自定义网络的配置, 程序将退出 ")
		return nil
	}
	return userDefinedNets
}
