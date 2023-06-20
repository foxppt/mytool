package swarmopt

import (
	"database/sql"
	"myTool/config"

	_ "github.com/go-sql-driver/mysql"
)

func getSvcUriFromMySQL(db *sql.DB, name string) (string, error) {
	var svcUri string
	sqlStr := `SELECT
	             info1.PARAMVALUE 
               FROM
	             GGS_SR_SERVICEINFO AS info1
	             JOIN GGS_SR_SERVICEINFO AS info2 ON info1.PARENTID = info2.ID
	             JOIN GGS_SR_SERVICEINFO AS info3 ON info2.PARENTID = info3.PARENTID 
               WHERE
	             info3.PARAMKEY = 'name' 
	             AND info3.PARAMVALUE = ? 
	             AND info2.PARAMKEY = 'settings' 
	             AND info1.PARAMKEY = 'DOCKERSERVICEURL'`
	err := db.QueryRow(sqlStr, name).Scan(&svcUri)
	if err != nil {
		return "", err
	}
	return svcUri, nil
}

// 初始化数据库
func InitDB(config *config.Config) (db *sql.DB, err error) {
	dsn := config.Mysql.User + ":" + config.Mysql.Passwd + "@tcp(" + config.Mysql.Host + ":" + config.Mysql.Port + ")" + "/" + config.Mysql.DBName
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return db, nil
}
