package swarmopt

import (
	"context"
	"errors"
	"myTool/config"
	"myTool/logger"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

// LeaveSwarm 离开swarm
func LeaveSwarm(ctx context.Context, dockerClient *client.Client, host config.HostConf, isLeader bool) {
	if !isLeader {
		err := execCMD(host.IP, host.Port, host.Username, host.Password, "docker swarm leave -f")
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
func JoinSwarm(host config.HostConf, joinToken string) {
	err := execCMD(host.IP, host.Port, host.Username, host.Password, joinToken)
	if err != nil {
		logger.SugarLogger.Errorln(host.IP, "加入swarm失败: ", err)
	}
	logger.SugarLogger.Infoln(host.IP, "加入swarm成功. ")
}

// GetSwarmLeader 获取Swarm节点角色
func GetSwarmNodeRole(ctx context.Context, dockerClient *client.Client, nodeAddr string) (bool, error) {
	nodeList, err := dockerClient.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		logger.SugarLogger.Panicln(err)
	}
	for _, node := range nodeList {
		if node.Status.Addr == nodeAddr {
			return node.ManagerStatus.Leader, nil
		}
	}
	err = errors.New("节点信息中没有匹配到" + nodeAddr + "这个IP的相关信息. ")
	return false, err
}

// GetSwarmJoinTK 获取swarm join-token
func GetSwarmJoinTK(ctx context.Context, dockerClient *client.Client) (joinTokenLeader string, joinTokenWorker string, err error) {
	swarm, err := dockerClient.SwarmInspect(ctx)
	if err != nil {
		return "", "", err
	}
	return swarm.JoinTokens.Manager, swarm.JoinTokens.Worker, nil
}
