// package operation

// import (
// 	"database/sql"
// 	"errors"
// 	"myTool/config"

// 	_ "github.com/go-sql-driver/mysql"
// )

// type Databases struct {
// 	Globe         *sql.DB
// 	ServiceCenter *sql.DB
// 	ServiceProxy  *sql.DB
// }

// // 初始化数据库
// func (dbs *Databases) InitDB(dbConfig *config.Mysql) (db *sql.DB, err error) {
// 	dsn := dbConfig.User + ":" + dbConfig.Passwd + "@tcp(" + dbConfig.Host + ":" + dbConfig.Port + ")" + "/" + dbConfig.DBName
// 	db, err = sql.Open("mysql", dsn)
// 	if err != nil {
// 		return nil, err
// 	}
// 	err = db.Ping()
// 	if err != nil {
// 		return nil, err
// 	}
// 	return db, nil
// }

// // 公有sql Exec执行类
// func (dbs *Databases) Exec(db string, query string, args ...interface{}) (sql.Result, error) {
// 	switch db {
// 	case "Globe":
// 		res, err := dbs.Globe.Exec(query, args...)
// 		if err != nil {
// 			return nil, err
// 		}
// 		return res, nil
// 	case "ServiceCenter":
// 		res, err := dbs.ServiceCenter.Exec(query, args...)
// 		if err != nil {
// 			return nil, err
// 		}
// 		return res, nil
// 	case "ServiceProxy":
// 		res, err := dbs.ServiceProxy.Exec(query, args...)
// 		if err != nil {
// 			return nil, err
// 		}
// 		return res, nil
// 	}
// 	return nil, errors.New("数据库查询方法未实现")
// }

// func (dbs *Databases) Query(db string, query string, args ...interface{}) (*sql.Rows, error) {
// 	switch db {
// 	case "Globe":
// 		rows, err := dbs.Globe.Query(query, args...)
// 		if err != nil {
// 			return nil, err
// 		}
// 		return rows, nil
// 	case "ServiceCenter":
// 		rows, err := dbs.ServiceCenter.Query(query, args...)
// 		if err != nil {
// 			return nil, err
// 		}
// 		return rows, nil
// 	case "ServiceProxy":
// 		rows, err := dbs.ServiceProxy.Query(query, args...)
// 		if err != nil {
// 			return nil, err
// 		}
// 		return rows, nil
// 	}
// 	return nil, errors.New("数据库查询方法未实现")
// }

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
