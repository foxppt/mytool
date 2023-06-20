package swarmopt

import (
	"myTool/config"
	"myTool/logger"
	"strconv"

	"golang.org/x/crypto/ssh"
)

func reloadDocker(config *config.Config) {
	for _, host := range config.Host {
		// 设置 SSH 客户端配置
		sshConfig := &ssh.ClientConfig{
			User: host.Username,
			Auth: []ssh.AuthMethod{
				ssh.Password(host.Password),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}

		// 连接到远程 SSH 服务器
		conn, err := ssh.Dial("tcp", host.IP+":"+strconv.Itoa(host.Port), sshConfig)
		if err != nil {
			logger.SugarLogger.Fatalf("SSH远程连接失败 : %s", err)
		}
		defer conn.Close()

		// 在 SSH 连接上创建一个新会话
		session, err := conn.NewSession()
		if err != nil {
			logger.SugarLogger.Fatalf("创建 session 失败: %s", err)
		}
		defer session.Close()

		// 运行命令重启Docker服务
		cmd := "systemctl restart docker"
		if err := session.Run(cmd); err != nil {
			logger.SugarLogger.Fatalf("执行CMD命令失败: %s", err)
		}
		logger.SugarLogger.Infoln(host.IP, "docker已重启 ")
	}
}
