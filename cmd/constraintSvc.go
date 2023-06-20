/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"myTool/config"
	"myTool/logger"
	"myTool/swarmopt"
	"os"

	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

// constraintSvcCmd represents the constraintSvc command
var constraintSvcCmd = &cobra.Command{
	Use:   "constraintSvc",
	Short: "约束GeoGlobe Server 服务到初次投递节点",
	Long: `constraintSvc: 
  约束GeoGlobe Server 服务到初次投递节点, 可以预防Overlay跨节点通信存在问题时服务无法访问:
  第一次运行时程序会在当前目录初始化一个config/config.yml文件, 
  用户需完善相关配置才能再次运行. 
  yaml配置文件格式: 
  # 集群主机连接信息
  host:
    - ip: 示例节点1-IP
      port: 节点SSH端口
      username: 节点SSH端口
      password: 节点SSH密码
    - ip: 示例节点2-IP
      port:节点SSH端口
      username: 节点SSH端口
      password: 节点SSH密码
    # 如果有多个节点可以继续添加, 相反, 如果只有一个节点, 删除第二个节(示例节点2)
  # Ingress CIDR定义
  ingress:
    subnet: 172.29.0.1/20 # Ingress网络CIDR定义, 可以自行修改
    gateway: 172.29.0.254 # Ingress网络网关
  # 数据库连接信息
  mysql:
    host: 数据库ip
    port: 数据库端口
    dbname: 数据库库名
    user: 数据库用户名
    passwd: 数据库密码`,
	Run: func(cmd *cobra.Command, args []string) {
		var ctx = context.Background()
		dockerClient, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation(), client.WithVersion(""))
		if err != nil {
			panic(err)
		}
		hostConfig := config.GetHostConfig()
		if hostConfig == nil {
			os.Exit(0)
		}
		db, err := swarmopt.InitDB(hostConfig)
		if err != nil {
			logger.SugarLogger.Fatalln(err)
		}

		swarmopt.RecordSvc(ctx, dockerClient, hostConfig, db)

		serviceConfig := config.GetSvcConfig()
		if serviceConfig == nil {
			logger.SugarLogger.Panicln("读取service配置失败")
		}

		swarmopt.ConstraitService(ctx, dockerClient, serviceConfig)
	},
}

func init() {
	rootCmd.AddCommand(constraintSvcCmd)
}
