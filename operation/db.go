package operation

import (
	"myTool/config"
	"myTool/logger"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectionInit(dbConf config.DB) *gorm.DB {
	switch dbConf.DBType {
	case "mysql":
		dsn := dbConf.User + ":" + dbConf.Passwd + "@tcp(" + dbConf.Host + ":" + dbConf.Port + ")/" + dbConf.DBName + "?charset=utf8mb4&parseTime=True&loc=Local"
		logger.SugarLogger.Infoln("连接到", dbConf.DBType, "数据库: ", dbConf.User, "@", dbConf.Host, ":", dbConf.Port, "/", dbConf.DBName)
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			logger.SugarLogger.Panicln(err)
		}
		return db
	case "postgres":
		dsn := "host=" + dbConf.Host + " user=" + dbConf.User + " password=" + dbConf.Passwd + " dbname=" + dbConf.DBName + " port=" + dbConf.Port + " sslmode=disable TimeZone=Asia/Shanghai"
		logger.SugarLogger.Infoln("连接到", dbConf.DBType, "数据库: ", dbConf.User, "@", dbConf.Host, ":", dbConf.Port, "/", dbConf.DBName)
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			logger.SugarLogger.Panicln(err)
		}
		return db
	}
	return nil
}
