package operation

import (
	"myTool/config"
	"myTool/logger"
	"os/exec"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
)

// 如果 &hostConf 传入为空值则代表bash命令在本地执行
func execCMD(hostConf *config.HostConf, command string) (string, error) {
	logger.SugarLogger.Infoln("执行bash命令: ", command)
	if hostConf == nil {
		args := strings.Split(command, " ")
		cmd := exec.Command(args[0], args[1:]...)
		resp, err := cmd.CombinedOutput()
		if err != nil {
			return string(resp), err
		}
		return string(resp), nil
	} else {
		sshConfig := &ssh.ClientConfig{
			User: hostConf.Username,
			Auth: []ssh.AuthMethod{
				ssh.Password(hostConf.Password),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}

		// 连接到远程 SSH 服务器
		conn, err := ssh.Dial("tcp", hostConf.Username+":"+strconv.Itoa(hostConf.Port), sshConfig)
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
		output, err := session.CombinedOutput(command)
		if err != nil {
			return string(output), err
		}

		return string(output), nil
	}
}
