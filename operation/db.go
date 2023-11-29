package operation

import (
	"errors"
	"myTool/config"
	"myTool/logger"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectionInit(dbConf config.DB) (*gorm.DB, error) {
	switch dbConf.DBType {
	case "mysql":
		dsn := dbConf.User + ":" + dbConf.Passwd + "@tcp(" + dbConf.Host + ":" + dbConf.Port + ")/" + dbConf.DBName + "?charset=utf8mb4&parseTime=True&loc=Local"
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			logger.SugarLogger.Panicln(err)
		}
		return db, nil
	case "postgres":
		dsn := "host=" + dbConf.Host + " user=" + dbConf.User + " password=" + dbConf.Passwd + " dbname=" + dbConf.DBName + " port=" + dbConf.Port + " sslmode=disable TimeZone=Asia/Shanghai"
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			logger.SugarLogger.Panicln(err)
		}
		return db, nil
	}
	return nil, errors.New("数据库类型不受支持, 目前只支持mysql和pg")
}
