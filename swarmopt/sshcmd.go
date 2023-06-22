package swarmopt

import (
	"myTool/logger"
	"strconv"

	"golang.org/x/crypto/ssh"
)

func execCMD(hostIP string, hostPort int, userName, userPass, command string) error {
	sshConfig := &ssh.ClientConfig{
		User: userName,
		Auth: []ssh.AuthMethod{
			ssh.Password(userPass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// 连接到远程 SSH 服务器
	conn, err := ssh.Dial("tcp", hostIP+":"+strconv.Itoa(hostPort), sshConfig)
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

	// 运行命令
	cmd := command
	if err := session.Run(cmd); err != nil {
		return err
	}
	return nil
}
