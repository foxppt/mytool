package operation

import (
	"fmt"
	"myTool/config"
	"myTool/pkg/dm"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func DBConnectionInit(dbConf config.DB) (*gorm.DB, error) {
	switch dbConf.DBType {
	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			dbConf.User, dbConf.Passwd, dbConf.Host, dbConf.Port, dbConf.DBName)
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
		return db, err
	case "postgres":
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Shanghai",
			dbConf.Host, dbConf.User, dbConf.Passwd, dbConf.DBName, dbConf.Port)
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
		return db, err
	case "dm":
		dsn := fmt.Sprintf("dm://%s:%s@%s:%s",
			dbConf.User, dbConf.Passwd, dbConf.Host, dbConf.Port)
		db, err := gorm.Open(dm.Open(dsn), &gorm.Config{})
		return db, err
	default:
		return nil, fmt.Errorf("数据库类型%s不受支持, 目前只支持mysql、postgres和达梦", dbConf.DBType)
	}
}
