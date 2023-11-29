package operation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"myTool/config"
	"myTool/logger"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

// GeoGlobe ggs_sr_serviceinfo 表结构体
type GGSSrServiceinfo struct {
	ID         int    `gorm:"column:id;type:int4"`
	ParamValue string `gorm:"column:paramvalue;type:bytea"`
}

// GeoSmarter sc_service_cluster 表结构体
type ScServiceCluster struct {
	ServiceAddress string `gorm:"column:service_address;type:varchar(255)"`
}

// GeoSmarter proxy_cluster 表结构体
type ProxyCluster struct {
	PhysicalAddress string `gorm:"column:physical_address;type:varchar(255)"`
}

// GeoSmarter sc_service 表结构体
type ScService struct {
	ID string `gorm:"column:id;type:varchar(255)"`
}

// GeoSmarter proxy 表结构体
type Proxy struct {
	ID string `gorm:"column:id;type:varchar(255)"`
}

// RecordSvc 记录swarm中service的信息
func RecordSvc(ctx context.Context, dockerClient *client.Client, hostConfig *config.Config, isGlobe bool, dbConf *config.DBConfig, svcConf string) {
	var svcStructs []config.ServiceConfig
	// 获取服务信息
	serviceList, err := dockerClient.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		panic(err)
	}

	// 获取网络信息
	networks, err := dockerClient.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		panic(err)
	}

	// 获取节点信息
	nodeList, err := dockerClient.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		logger.SugarLogger.Fatalln(err)
	}

	// ip匹配正则
	ipRegex := regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)
	// 数据库连接对象
	logger.SugarLogger.Infof("连接ServiceMgr数据库")
	dbGlobe, err := ConnectionInit(dbConf.Globe)
	if dbGlobe == nil {
		logger.SugarLogger.Infof("连接到%s数据库: %s@%s:%s/%s", dbConf.Globe.DBType, dbConf.Globe.User, dbConf.Globe.Host, dbConf.Globe.Port, dbConf.Globe.DBName)
	} else {
		logger.SugarLogger.Panicln(err)
	}
	var serviceResults GGSSrServiceinfo
	var schema string
	if dbConf.Globe.Schema == "" {
		schema = dbConf.Globe.Schema
	} else {
		schema = dbConf.Globe.Schema + "."
	}
	// 遍历服务并将相应的信息附加到 svcStructs
	for _, service := range serviceList {
		var svcStruct config.ServiceConfig
		if isGlobe {
			dbGlobe.Table(schema+"ggs_sr_serviceinfo as info1").
				Joins("Join "+schema+"ggs_sr_serviceinfo as info2 ON info1.PARENTID = info2.ID").
				Joins("Join "+schema+"ggs_sr_serviceinfo as info3 ON info2.PARENTID = info3.PARENTID").
				Where("info3.PARAMKEY = ? AND info3.PARAMVALUE = ?", "name", service.Spec.Name).
				Where("info2.PARAMKEY = ?", "settings").
				Where("info1.PARAMKEY = ?", "DOCKERSERVICEURL").Find(&serviceResults)
			logger.SugarLogger.Infoln("查询service", service.Spec.Name, "的数据库信息")
			url := serviceResults.ParamValue

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
		if len(service.Endpoint.Ports) > 0 {
			svcStruct.TargetPort = service.Endpoint.Ports[0].TargetPort
			svcStruct.PublishPort = service.Endpoint.Ports[0].PublishedPort
		}
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
	serviceList, err := dockerClient.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		panic(err)
	}

	// 删除docker swarm中所有的service
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
func RebuildSvc(ctx context.Context, dockerClient *client.Client, serviceConfig *[]config.ServiceConfig, dbConf *config.DBConfig) error {
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

		// 如果是Globe的服务就更新下数据库记录，否则可能删除的时候会有资源遗留
		if dbConf != nil {
			logger.SugarLogger.Infof("连接ServiceMgr数据库")
			dbGlobe, err := ConnectionInit(dbConf.Globe)
			if err == nil {
				logger.SugarLogger.Infof("连接到%s数据库: %s@%s:%s/%s", dbConf.Globe.DBType, dbConf.Globe.User, dbConf.Globe.Host, dbConf.Globe.Port, dbConf.Globe.DBName)
			} else {
				logger.SugarLogger.Panicln(err)
			}
			var serviceResults GGSSrServiceinfo
			var dbServiceID string
			var dbRecordID string
			var schema string
			if dbConf.Globe.Schema == "" {
				schema = dbConf.Globe.Schema
			} else {
				schema = dbConf.Globe.Schema + "."
			}
			dbGlobe.Table(schema+"ggs_sr_serviceinfo as info1").
				Joins("Join "+schema+"ggs_sr_serviceinfo as info2 ON info1.PARENTID = info2.ID").
				Joins("Join "+schema+"ggs_sr_serviceinfo as info3 ON info2.PARENTID = info3.PARENTID").
				Where("info3.PARAMKEY = ? AND info3.PARAMVALUE = ?", "name", service.Name).
				Where("info2.PARAMKEY = ?", "settings").
				Where("info1.PARAMKEY = ?", "DOCKERID").Find(&serviceResults)
			dbRecordID = strconv.Itoa(serviceResults.ID)
			dbServiceID = serviceResults.ParamValue
			logger.SugarLogger.Infoln("数据库记录服务ID: ", dbServiceID, "被更新成: ", resp.ID)
			dbGlobe.Table(schema+"ggs_sr_serviceinfo").Where("id = ?", dbRecordID).Find(&serviceResults)
			serviceResults.ParamValue = resp.ID
			tx := dbGlobe.Table(schema+"ggs_sr_serviceinfo").Where("id = ?", dbRecordID).Save(&serviceResults)
			logger.SugarLogger.Infof("update ggs_sr_serviceinfo , %d rows affected. ", tx.RowsAffected)
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
func ChangeSvcNode(ctx context.Context, dockerClient *client.Client, dbConf *config.DBConfig, serviceName string, nodeTarget string) error {
	serviceList, err := dockerClient.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		panic(err)
	}
	nodeList, err := dockerClient.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		panic(err)
	}

	logger.SugarLogger.Infof("连接ServiceMgr数据库")
	dbGlobe, err := ConnectionInit(dbConf.Globe)
	if err == nil {
		logger.SugarLogger.Infof("连接到%s数据库: %s@%s:%s/%s", dbConf.Globe.DBType, dbConf.Globe.User, dbConf.Globe.Host, dbConf.Globe.Port, dbConf.Globe.DBName)
	} else {
		logger.SugarLogger.Panicln(err)
	}
	var serviceResults GGSSrServiceinfo
	var schema string
	if dbConf.Globe.Schema == "" {
		schema = dbConf.Globe.Schema
	} else {
		schema = dbConf.Globe.Schema + "."
	}
	dbGlobe.Table(schema+"ggs_sr_serviceinfo as info1").
		Joins("Join "+schema+"ggs_sr_serviceinfo as info2 ON info1.PARENTID = info2.ID").
		Joins("Join "+schema+"ggs_sr_serviceinfo as info3 ON info2.PARENTID = info3.PARENTID").
		Where("info3.PARAMKEY = ? AND info3.PARAMVALUE = ?", "name", serviceName).
		Where("info2.PARAMKEY = ?", "settings").
		Where("info1.PARAMKEY = ?", "DOCKERSERVICEURL").Find(&serviceResults)
	url := serviceResults.ParamValue
	if url == "" {
		logger.SugarLogger.Panicln("结果为0行, 服务未找到. ")
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
	var dbServiceID string
	var dbRecordID string
	serviceResults = GGSSrServiceinfo{}
	dbGlobe.Table(schema+"ggs_sr_serviceinfo as info1").
		Joins("Join "+schema+"ggs_sr_serviceinfo as info2 ON info1.PARENTID = info2.ID").
		Joins("Join "+schema+"ggs_sr_serviceinfo as info3 ON info2.PARENTID = info3.PARENTID").
		Where("info3.PARAMKEY = ? AND info3.PARAMVALUE = ?", "name", serviceName).
		Where("info2.PARAMKEY = ?", "settings").
		Where("info1.PARAMKEY = ?", "DOCKERID").Find(&serviceResults)

	dbRecordID = strconv.Itoa(serviceResults.ID)
	dbServiceID = serviceResults.ParamValue
	logger.SugarLogger.Infof("查询到的服务ID: %s, 服务名称: %s", dbServiceID, serviceName)

	logger.SugarLogger.Infof("连接服务中心数据库")
	dbServiceCenter, err := ConnectionInit(dbConf.ServiceCenter)
	if err == nil {
		logger.SugarLogger.Infof("连接到%s数据库: %s@%s:%s/%s", dbConf.ServiceCenter.DBType, dbConf.ServiceCenter.User, dbConf.ServiceCenter.Host, dbConf.ServiceCenter.Port, dbConf.ServiceCenter.DBName)
	} else {
		logger.SugarLogger.Panicln(err)
	}

	logger.SugarLogger.Infof("连接服务网关数据库")
	dbServiceProxy, err := ConnectionInit(dbConf.ServiceProxy)
	if err == nil {
		logger.SugarLogger.Infof("连接到%s数据库: %s@%s:%s/%s", dbConf.ServiceProxy.DBType, dbConf.ServiceProxy.User, dbConf.ServiceProxy.Host, dbConf.ServiceProxy.Port, dbConf.ServiceProxy.DBName)
	} else {
		logger.SugarLogger.Panicln(err)
	}

	var serviceCenter ScServiceCluster
	var serviceProxy ProxyCluster
	var service ScService
	var proxy Proxy
	for _, svc := range serviceList {
		if svc.Spec.Name == serviceName {
			for _, node := range nodeList {
				if node.Status.Addr == nodeTarget {
					unConstraitService(ctx, dockerClient, svc)
					constraint(ctx, dockerClient, svc, node.ID, nodeTarget)

					logger.SugarLogger.Infoln("数据库记录服务地址: ", url, "将被更新成: ", newUrl)
					if dbConf.ServiceCenter.Schema == "" {
						schema = dbConf.ServiceCenter.Schema
					} else {
						schema = dbConf.ServiceCenter.Schema + "."
					}
					dbServiceCenter.Table(schema+"sc_service").Where("service_name = ?", serviceName).Find(&service)
					dbServiceCenter.Table(schema+"sc_service_cluster").Where("service_id = ?", service.ID).Find(&serviceCenter)
					serviceCenter.ServiceAddress = strings.Replace(serviceCenter.ServiceAddress, serviceCenter.ServiceAddress, newUrl+"/", -1)
					tx := dbServiceCenter.Table(schema+"sc_service_cluster").Where("service_id = ?", service.ID).Save(&serviceCenter)
					logger.SugarLogger.Infof("update sc_service_cluster , %d rows affected. ", tx.RowsAffected)

					if dbConf.ServiceProxy.Schema == "" {
						schema = dbConf.ServiceProxy.Schema
					} else {
						schema = dbConf.ServiceProxy.Schema + "."
					}
					dbServiceProxy.Table(schema+"proxy").Where("servicename", serviceName).Find(&proxy)
					dbServiceProxy.Table(schema+"proxy_cluster").Where("proxy_id", proxy.ID).Find(&serviceProxy)
					serviceProxy.PhysicalAddress = strings.Replace(serviceProxy.PhysicalAddress, serviceProxy.PhysicalAddress, newUrl+"/", -1)
					tx = dbServiceProxy.Table(schema+"proxy_cluster").Where("proxy_id", proxy.ID).Save(&serviceProxy)
					logger.SugarLogger.Infof("update proxy_cluster , %d rows affected. ", tx.RowsAffected)

					if dbConf.Globe.Schema == "" {
						schema = dbConf.Globe.Schema
					} else {
						schema = dbConf.Globe.Schema + "."
					}
					serviceResults = GGSSrServiceinfo{}
					dbGlobe.Table(schema+"ggs_sr_serviceinfo").Where("paramvalue = ?", url).Find(&serviceResults)
					serviceResults.ParamValue = strings.Replace(serviceResults.ParamValue, url, newUrl, -1)
					tx = dbGlobe.Table(schema+"ggs_sr_serviceinfo").Where("paramvalue = ?", url).Save(&serviceResults)
					logger.SugarLogger.Infof("update ggs_sr_serviceinfo , %d rows affected. ", tx.RowsAffected)

					serviceResults = GGSSrServiceinfo{}
					dbGlobe.Table(schema+"ggs_sr_serviceinfo").Where("id = ?", dbRecordID).Find(&serviceResults)
					if serviceResults.ParamValue != svc.ID {
						logger.SugarLogger.Infoln("数据库记录服务ID: ", dbServiceID, "被更新成: ", svc.ID)
						serviceResults.ParamValue = svc.ID
						tx = dbGlobe.Table(schema+"ggs_sr_serviceinfo").Where("id = ?", dbRecordID).Save(&serviceResults)
						logger.SugarLogger.Infof("update ggs_sr_serviceinfo , %d rows affected. ", tx.RowsAffected)
					}
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
