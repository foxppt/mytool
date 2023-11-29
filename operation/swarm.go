package operation

import (
	"context"
	"errors"
	"myTool/config"
	"myTool/logger"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

// LeaveSwarm 离开swarm
func LeaveSwarm(ctx context.Context, dockerClient *client.Client, host config.HostConf, isLeader bool) {
	if !isLeader {
		_, err := execCMD(&host, "docker swarm leave -f")
		if err != nil {
			logger.SugarLogger.Panicln("从节点", host.IP, "离开swarm失败: ", err)
		}
		logger.SugarLogger.Infoln("从节点", host.IP, "离开swarm成功. ")
	} else {
		if err := dockerClient.SwarmLeave(ctx, true); err != nil {
			logger.SugarLogger.Panicln("主节点离开swarm失败: ", err)
		}
		logger.SugarLogger.Infoln("主节点离开swarm成功. ")
	}
}

// InitSwarm 初始化swarm
func InitSwarm(ctx context.Context, dockerClient *client.Client, listenAddr string) {
	resp, err := dockerClient.SwarmInit(ctx, swarm.InitRequest{ListenAddr: listenAddr})
	if err != nil {
		logger.SugarLogger.Panicln(err)
	}
	logger.SugarLogger.Infoln("swarm ID: ", resp)
}

// JoinSwarm 加入swarm
func JoinSwarm(host config.HostConf, joinToken string) error {
	// 判断如果是本机就continue跳过, 因为本机不能加入第二次
	resp, err := execCMD(&host, "docker node ls")
	if err != nil {
		if strings.Contains(resp, "This node is not a swarm manager") {
			logger.SugarLogger.Infoln(host.IP, "将尝试加入集群")
		}
	}
	if strings.Contains(resp, "Active") && strings.Contains(resp, "Leader") {
		err := errors.New("这个节点是当前主节点, 已经跳过")
		return err
	}
	resp, err = execCMD(&host, joinToken)
	if err != nil {
		logger.SugarLogger.Errorln(host.IP, "加入swarm失败: ", resp, err)
	}
	logger.SugarLogger.Infoln(host.IP, "加入swarm成功. ")
	return err
}

// GetSwarmLeader 获取Swarm节点角色
func GetSwarmNodeRole(ctx context.Context, dockerClient *client.Client, host config.HostConf) (bool, error) {
	cmd := "docker node ls"
	resp, err := execCMD(&host, cmd)
	if strings.Contains(resp, "This node is not a swarm manager") {
		return false, nil
	} else if strings.Contains(resp, "Active") && strings.Contains(resp, "Leader") {
		return true, nil
	} else {
		return false, err
	}
}

// GetSwarmJoinTK 获取swarm join-token
func GetSwarmJoinTK(ctx context.Context, dockerClient *client.Client) (joinTokenLeader string, joinTokenWorker string, err error) {
	swarm, err := dockerClient.SwarmInspect(ctx)
	if err != nil {
		return "", "", err
	}

	nodeList, err := dockerClient.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		return "", "", err
	}
	return "docker swarm join --token " + swarm.JoinTokens.Manager + " " + nodeList[0].ManagerStatus.Addr, "docker swarm join --token " + swarm.JoinTokens.Worker + " " + nodeList[0].ManagerStatus.Addr, nil
}
