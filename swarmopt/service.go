package swarmopt

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"myTool/config"
	"myTool/logger"
	"os"
	"regexp"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

// RecordSvc 记录swarm中service的信息
func RecordSvc(ctx context.Context, dockerClient *client.Client, hostConfig *config.Config, db *sql.DB, svcConf string) {
	var svcStructs []config.ServiceConfig

	serviceList, err := dockerClient.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		panic(err)
	}

	nodeList, err := dockerClient.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		logger.SugarLogger.Fatalln(err)
	}
	ipRegex := regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)
	// 遍历服务并将相应的信息附加到 svcStructs
	for _, service := range serviceList {
		var svcStruct config.ServiceConfig
		url, err := getSvcUriFromMySQL(db, service.Spec.Name)
		if err != nil {
			logger.SugarLogger.Warnln("服务", service.Spec.Name, "数据库查询结果为", err)
			// 跳过本次循环，开始下一次
			continue
		}
		logger.SugarLogger.Infoln("查询服务到", service.Spec.Name, "的相关信息")
		match := ipRegex.FindStringSubmatch(url)
		if len(match) == 0 {
			logger.SugarLogger.Errorln("未解析到到IP! ")
		}
		for _, node := range nodeList {
			if node.Status.Addr == match[0] {
				svcStruct.NodeID = node.ID
			}
		}
		svcStruct.Host = match[0]
		svcStruct.Name = service.Spec.Name
		svcStruct.RawSvcID = service.ID
		svcStruct.Labels = service.Spec.Labels
		svcStruct.Image = service.Spec.TaskTemplate.ContainerSpec.Image
		svcStruct.TargetPort = service.Endpoint.Ports[0].TargetPort
		svcStruct.PublishPort = service.Endpoint.Ports[0].PublishedPort
		svcStruct.Env = service.Spec.TaskTemplate.ContainerSpec.Env
		svcStructs = append(svcStructs, svcStruct)
	}

	// 将 svcStructs 编码为JSON并将其写入services.json
	jsonBytes, err := json.MarshalIndent(svcStructs, "", "  ")
	if err != nil {
		panic(err)
	}
	// 判断services.json是否存在，如果存在就备份，备份文件名为services.json_时间戳
	timestamp := time.Now().Unix()
	if _, err := os.Stat(svcConf); !os.IsNotExist(err) {
		// 备份文件名为services.json_时间戳
		logger.SugarLogger.Infoln(svcConf, "已经存在")
		err := os.Rename(svcConf, fmt.Sprintf(svcConf+"_%d", timestamp))
		if err != nil {
			logger.SugarLogger.Errorln(svcConf, "备份文件失败：", err)
			return
		}
		logger.SugarLogger.Infoln(svcConf, "备份, 文件名为: ", fmt.Sprintf(svcConf+"_%d", timestamp))
	}
	f, err := os.Create(svcConf)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	_, err = f.Write(jsonBytes)
	if err != nil {
		panic(err)
	}
	logger.SugarLogger.Infoln("服务创建信息已被保存到", svcConf)
}

// DelService 清空当前swarm的service
func DelService(ctx context.Context, dockerClient *client.Client) {
	// 删除docker swarm中所有的service
	serviceList, err := dockerClient.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		panic(err)
	}

	for _, service := range serviceList {
		err = dockerClient.ServiceRemove(ctx, service.ID)
		if err != nil {
			panic(err)
		}
		logger.SugarLogger.Infoln("服务", service.Spec.Name, "已被删除 ")
	}
	logger.SugarLogger.Infoln("所有服务已被移除 ")
}

// RebuildSvc 根据已记录信息重新投递service
func RebuildSvc(ctx context.Context, dockerClient *client.Client, serviceConfig *[]config.ServiceConfig) error {
	// networkList, err := dockerClient.NetworkList(ctx, types.NetworkListOptions{})
	// if err != nil {
	// 	panic(err)
	// }

	// var servicemgrNetID string
	// for _, net := range networkList {
	// 	if net.Name == "servicemgr" {
	// 		logger.SugarLogger.Infoln("servicemgr存在, geoglobe服务将使用该网络!")
	// 		servicemgrNetID = net.ID
	// 	}
	// }

	// 创建服务
	for _, service := range *serviceConfig {
		svcContainerspec := &swarm.ContainerSpec{
			Image: service.Image,
			Env:   service.Env,
		}
		svcEndpoint := &swarm.EndpointSpec{
			Ports: []swarm.PortConfig{
				{
					Protocol:      "tcp",
					TargetPort:    service.TargetPort,
					PublishedPort: service.PublishPort,
					PublishMode:   "ingress",
				},
			},
		}
		var serviceSpec swarm.ServiceSpec
		// if servicemgrNetID != "" {
		// 	serviceSpec = swarm.ServiceSpec{
		// 		Networks: []swarm.NetworkAttachmentConfig{
		// 			{
		// 				Target: servicemgrNetID,
		// 			},
		// 		},
		// 	}
		// }
		serviceSpec.Networks = []swarm.NetworkAttachmentConfig{}
		serviceSpec.Name = service.Name
		serviceSpec.Labels = service.Labels
		serviceSpec.TaskTemplate.ContainerSpec = svcContainerspec
		serviceSpec.EndpointSpec = svcEndpoint

		resp, err := dockerClient.ServiceCreate(ctx, serviceSpec, types.ServiceCreateOptions{})
		if err != nil {
			panic(err)
		}

		logger.SugarLogger.Infoln("服务被创建:", resp.ID)
		if resp.Warnings != nil {
			logger.SugarLogger.Warnln(resp.Warnings)
		}
	}

	return nil
}

// ConstraitService 约束服务到服务初次投递的节点
func ConstraitService(ctx context.Context, dockerClient *client.Client, serviceConfig *[]config.ServiceConfig) {
	// 获取服务的列表
	time.Sleep(5 * time.Second)
	serviceList, err := dockerClient.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		panic(err)
	}

	// 遍历服务列表，更新每个服务
	for _, svc := range *serviceConfig {
		for _, service := range serviceList {
			if svc.Name == service.Spec.Name {
				// 约束这个服务
				placement := &swarm.Placement{
					Constraints: []string{
						"node.id==" + svc.NodeID,
					},
				}
				serviceSpec := swarm.ServiceSpec{}
				serviceSpec = service.Spec
				serviceSpec.TaskTemplate.Placement = placement
				rsp, err := dockerClient.ServiceUpdate(ctx, service.ID, service.Version, serviceSpec, types.ServiceUpdateOptions{})
				if err != nil {
					logger.SugarLogger.Errorln(rsp, err)
				} else {
					logger.SugarLogger.Infoln("服务", service.Spec.Name, "已被约束到", svc.Host)
				}
			}
		}
	}
}

// UnConstraitService 反约束服务
func UnConstraitService(ctx context.Context, dockerClient *client.Client) {
	// 获取服务的列表
	serviceList, err := dockerClient.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		panic(err)
	}

	// 遍历服务列表，更新每个服务
	for _, service := range serviceList {
		// 取消约束服务
		placement := &swarm.Placement{
			// Constraints: []string{
			// 	"",
			// },
			Constraints: nil,
		}
		serviceSpec := swarm.ServiceSpec{}
		serviceSpec = service.Spec
		serviceSpec.TaskTemplate.Placement = placement
		rsp, err := dockerClient.ServiceUpdate(ctx, service.ID, service.Version, serviceSpec, types.ServiceUpdateOptions{})
		if err != nil {
			logger.SugarLogger.Errorln(rsp, err)
		} else {
			logger.SugarLogger.Infoln("服务", service.Spec.Name, "已被取消约束")
		}
	}
}
