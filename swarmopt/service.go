package swarmopt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"myTool/config"
	"myTool/logger"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

// RecordSvc 记录swarm中service的信息
func RecordSvc(ctx context.Context, dockerClient *client.Client, hostConfig *config.Config, isGlobe bool, dbs *Databases, svcConf string) {
	var svcStructs []config.ServiceConfig

	serviceList, err := dockerClient.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		panic(err)
	}

	networks, err := dockerClient.NetworkList(ctx, types.NetworkListOptions{})
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
		if isGlobe {
			sqlStrGeoGlobeQuery := `SELECT
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

			rows, err := dbs.Query("Globe", sqlStrGeoGlobeQuery, service.Spec.Name)
			if err != nil {
				logger.SugarLogger.Errorln(err)
			}
			defer rows.Close()
			var urls []string
			var url string
			for rows.Next() {
				err = rows.Scan(&url)
				if err != nil {
					logger.SugarLogger.Errorln(err)
				}
				urls = append(urls, url)
			}
			if len(urls) == 0 {
				logger.SugarLogger.Infoln("结果为0行, 服务未找到. ")
			} else if len(urls) == 1 {
				url = urls[0]
			} else {
				logger.SugarLogger.Errorln("数据库查询结果不唯一, 语句可能存在问题. ")
			}

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

		}
		svcStruct.Name = service.Spec.Name
		svcStruct.RawSvcID = service.ID
		svcStruct.Labels = service.Spec.Labels
		svcStruct.Image = service.Spec.TaskTemplate.ContainerSpec.Image
		svcStruct.TargetPort = service.Endpoint.Ports[0].TargetPort
		svcStruct.PublishPort = service.Endpoint.Ports[0].PublishedPort
		svcStruct.Env = service.Spec.TaskTemplate.ContainerSpec.Env
		svcStruct.Replicas = *service.Spec.Mode.Replicated.Replicas
		if len(service.Endpoint.VirtualIPs) > 0 {
			for _, vip := range service.Endpoint.VirtualIPs {
				for _, net := range networks {
					if vip.NetworkID == net.ID {
						svcStruct.Network = append(svcStruct.Network, net.Name)
					}
				}
			}
		}
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
	networks, err := dockerClient.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		panic(err)
	}

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
		serviceSpec.Name = service.Name
		serviceSpec.Labels = service.Labels
		serviceSpec.TaskTemplate.ContainerSpec = svcContainerspec
		serviceSpec.EndpointSpec = svcEndpoint
		if len(service.Network) > 0 {
			for _, net := range service.Network {
				for _, network := range networks {
					if net == network.Name {
						serviceSpec.Networks = append(serviceSpec.Networks, swarm.NetworkAttachmentConfig{Target: network.ID})
					}
				}
			}
		}

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
				constraint(ctx, dockerClient, service, svc.NodeID, svc.Host)
			}
		}
	}
}

// UnConstraitService 反约束服务
func UnConstraitAll(ctx context.Context, dockerClient *client.Client) {
	// 获取服务的列表
	serviceList, err := dockerClient.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		panic(err)
	}

	// 遍历服务列表，更新每个服务
	for _, service := range serviceList {
		// 取消约束服务
		unConstraitService(ctx, dockerClient, service)
	}
}

// 根据服务名更换服务节点
func ChangeSvcNode(ctx context.Context, dockerClient *client.Client, dbs *Databases, serviceName string, nodeTarget string) error {
	serviceList, err := dockerClient.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		panic(err)
	}
	nodeList, err := dockerClient.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		panic(err)
	}

	sqlStrGeoGlobeQuery := `SELECT
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

	rows, err := dbs.Query("Globe", sqlStrGeoGlobeQuery, serviceName)
	if err != nil {
		logger.SugarLogger.Errorln(err)
	}
	defer rows.Close()
	var urls []string
	var url string
	for rows.Next() {
		err = rows.Scan(&url)
		if err != nil {
			logger.SugarLogger.Errorln(err)
		}
		urls = append(urls, url)
	}
	if len(urls) == 0 {
		logger.SugarLogger.Panicln("结果为0行, 服务未找到. ")
	} else if len(urls) == 1 {
		url = urls[0]
	} else {
		logger.SugarLogger.Panicln("数据库查询结果不唯一. ")
	}

	newUrl := ""
	if len(strings.Split(url, ":")) >= 2 {
		for k, v := range strings.Split(url, ":") {
			if k == 1 {
				v = "//" + nodeTarget
			}
			newUrl = newUrl + v
			if k == len(strings.Split(url, ":"))-1 {
				// 最后一次循环,不需要再加":"了
			} else {
				newUrl = newUrl + ":"
			}
		}
	} else {
		logger.SugarLogger.Errorln("查询到的服务地址存在问题", url)
	}

	for _, svc := range serviceList {
		if svc.Spec.Name == serviceName {
			for _, node := range nodeList {
				if node.Status.Addr == nodeTarget {
					unConstraitService(ctx, dockerClient, svc)
					constraint(ctx, dockerClient, svc, node.ID, nodeTarget)

					sqlStrServiceCenter := `update sc_service_cluster set service_address = replace(service_address, ?, ?) where service_address = ?`
					sqlStrServiceProxy := `update proxy_cluster set physical_address = replace(physical_address, ?, ?) where physical_address = ?`
					sqlStrGeoGlobeExec := `update ggs_sr_serviceinfo set paramvalue = replace(paramvalue, ?, ?) where paramvalue = ?`

					logger.SugarLogger.Infoln("数据库记录服务地址: ", url, "将被替换成: ", newUrl)
					res, err := dbs.ServiceCenter.Exec(sqlStrServiceCenter, url, newUrl, url+"/")
					if err != nil {
						logger.SugarLogger.Errorln(err)
					}
					rows, _ := res.RowsAffected()
					logger.SugarLogger.Infof("update sc_service_cluster , %d rows affected. ", rows)

					res, err = dbs.ServiceProxy.Exec(sqlStrServiceProxy, url, newUrl, url+"/")
					if err != nil {
						logger.SugarLogger.Errorln(err)
					}
					rows, _ = res.RowsAffected()
					logger.SugarLogger.Infof("update proxy_cluster , %d rows affected. ", rows)

					res, err = dbs.Globe.Exec(sqlStrGeoGlobeExec, url, newUrl, url)
					if err != nil {
						logger.SugarLogger.Errorln(err)
					}
					rows, _ = res.RowsAffected()
					logger.SugarLogger.Infof("update ggs_sr_serviceinfo , %d rows affected. ", rows)
					return nil
				}
			}
			return errors.New("节点" + nodeTarget + "没有找到!")
		}
	}
	return errors.New("服务" + serviceName + "未找到")
}

// 取消约束单个服务
func unConstraitService(ctx context.Context, dockerClient *client.Client, service swarm.Service) {
	placement := &swarm.Placement{
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

// 约束单个服务
func constraint(ctx context.Context, dockerClient *client.Client, service swarm.Service, nodeID string, nodeIP string) {
	service, _, err := dockerClient.ServiceInspectWithRaw(ctx, service.ID, types.ServiceInspectOptions{})
	if err != nil {
		logger.SugarLogger.Errorln(err)
	}

	placement := &swarm.Placement{
		Constraints: []string{
			"node.id==" + nodeID,
		},
	}
	serviceSpec := swarm.ServiceSpec{}
	serviceSpec = service.Spec
	serviceSpec.TaskTemplate.Placement = placement
	rsp, err := dockerClient.ServiceUpdate(ctx, service.ID, service.Version, serviceSpec, types.ServiceUpdateOptions{})
	if err != nil {
		logger.SugarLogger.Errorln(rsp, err)
	} else {
		logger.SugarLogger.Infoln("服务", service.Spec.Name, "已被约束到", nodeIP)
	}
}
