package config

import (
	"html/template"
	"myTool/logger"
	"os"

	"gopkg.in/yaml.v2"
)

var DBConf *DBConfig

const tmplDB string = `# 数据库配置文件
# servicemgr配置
globe:
{{- with .Globe }}
  dbtype: {{ .DBType }} # 数据库类型mysql/postgres
  host:   {{ .Host }}   # 数据库主机
  port:   {{ .Port }}   # 数据库端口
  dbname: {{ .DBName }} # 数据库库名
  schema: {{ .Schema }} # 数据库schema, 如果数据库是Mysql冒号后面含注释都删除
  user:   {{ .User }}   # 数据库用户名
  passwd: {{ .Passwd }} # 数据库密码
{{- end }}

# 服务中心配置
serviceCenter:
{{- with .ServiceCenter }}
  dbtype: {{ .DBType }} # 数据库类型mysql/postgres
  host:   {{ .Host }}   # 数据库主机
  port:   {{ .Port }}   # 数据库端口
  dbname: {{ .DBName }} # 数据库库名
  schema: {{ .Schema }} # 数据库schema, 如果数据库是Mysql冒号后面含注释都删除
  user:   {{ .User }}   # 数据库用户名
  passwd: {{ .Passwd }} # 数据库密码
{{- end }}

# 服务网关配置
serviceProxy:
{{- with .ServiceProxy }}
  dbtype: {{ .DBType }} # 数据库类型mysql/postgres
  host:   {{ .Host }}   # 数据库主机
  port:   {{ .Port }}   # 数据库端口
  dbname: {{ .DBName }} # 数据库库名
  schema: {{ .Schema }} # 数据库schema, 如果数据库是Mysql冒号后面含注释都删除
  user:   {{ .User }}   # 数据库用户名
  passwd: {{ .Passwd }} # 数据库密码
{{- end }}`

func GetDBConfig() *DBConfig {
	if _, err := os.Stat("config/db.yml"); os.IsNotExist(err) {
		if _, err := os.Stat("config"); os.IsNotExist(err) {
			os.Mkdir("config", 0775)
		}
		logger.SugarLogger.Infoln("数据库配置文件不存在")

		initDBConf()

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

func initDBConf() {
	var examDBConf DBConfig
	examConn := DB{
		DBType: "数据库类型",
		Host:   "数据库IP或域名",
		Port:   "数据库端口",
		DBName: "数据库库名",
		Schema: "数据库模式",
		User:   "数据库用户名",
		Passwd: "数据库密码",
	}
	examDBConf.Globe = examConn
	examDBConf.ServiceCenter = examConn
	examDBConf.ServiceProxy = examConn

	tmplParser := template.Must(template.New("dbConfig").Parse(tmplDB))

	f, err := os.Create("config/db.yml")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	err = tmplParser.Execute(f, examDBConf)
	if err != nil {
		panic(err)
	}
}
