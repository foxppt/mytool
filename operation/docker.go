package operation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"myTool/logger"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/client"
	"github.com/shirou/gopsutil/disk"
)

const (
	B = uint64(1)
	K = 1024 * B
	M = 1024 * K
	G = 1024 * M
	T = 1024 * G
	P = 1024 * T
)

func ChangeDockerBaseDir(ctx context.Context, dockerClient *client.Client, srcDir string, destDir string) error {
	// 比对源路径是否为docker使用的目录
	dockerRootDir, err := checkSrcISDockerRootDir(ctx, dockerClient)
	if err != nil {
		return err
	}
	if dockerRootDir != srcDir {
		return fmt.Errorf("源路径存在问题, 目前docker使用的目录为:%s", dockerRootDir)
	}

	// 获取原始路径的文件夹大小
	var sourceSize uint64
	var destSize uint64
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		logger.SugarLogger.Panicf("源路径%s不存在", srcDir)
	} else {
		sourceSize, err = calcDir(srcDir)
		if err != nil {
			return err
		}
		logger.SugarLogger.Infof("docker目录%s的大小为%s", srcDir, formatFileSize(sourceSize))
	}

	// 获取目标路径的可用大小
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		logger.SugarLogger.Infoln(destDir, "不存在, 将被创建")
		os.MkdirAll(destDir, 0775)
	}
	diskStatus, err := disk.Usage(destDir)
	if err != nil {
		return err
	}
	destSize = diskStatus.Free
	logger.SugarLogger.Infof("目录%s的可用大小为%s", destDir, formatFileSize(destSize))
	err = os.RemoveAll(destDir)
	if err != nil {
		return err
	}

	// 如果目标路径空间不足，直接error返回
	if sourceSize >= destSize {
		return fmt.Errorf("目标路径%s空间不足", destDir)
	} else {
		// 如果目标路径空间充足
		logger.SugarLogger.Infof("目标路径%s空间满足要求", destDir)
		// 停止dockerd
		resp, err := execCMD(nil, "systemctl stop docker")
		if err != nil {
			logger.SugarLogger.Infoln(resp, err)
			logger.SugarLogger.Panicln(resp)
		}
		// 修改docker配置文件指向目标目录
		err = editDataroot(destDir)
		if err != nil {
			return errors.Join(fmt.Errorf("修改配置文件出错: "), err)
		}
		// 判断是不是一个挂载盘，如果是同一个，剪切原始路径的文件夹到目标路径
		sameMountpoint := isSameMountpoint(srcDir, destDir)
		if sameMountpoint {
			logger.SugarLogger.Infoln("目标路径与源路径在系统中挂载点相同, 启用剪切逻辑. ")
			err = os.Rename(srcDir, destDir)
			if err != nil {
				logger.SugarLogger.Errorln("剪切数据到目标目录时出错: ", err)
				err = editDataroot(srcDir)
				if err != nil {
					return errors.Join(fmt.Errorf("修改配置文件出错: "), err)
				}
				resp, err := execCMD(nil, "systemctl start docker")
				if err != nil {
					logger.SugarLogger.Panicln(resp)
				}
				return err
			}
		} else {
			// 如果不是同一个挂载盘，复制原始路径的文件夹到目标路径, 等复制完成以后再删除原始目录(可以保证文件不损坏)
			logger.SugarLogger.Infoln("目标路径与源路径在系统中挂载点不同, 启用复制逻辑. ")
			errCopy := copyDir(srcDir, destDir)
			if errCopy != nil {
				// 如果拷贝文件失败了得回滚(就是源和目标反过来修改)，不然docker会出问题
				logger.SugarLogger.Errorln("复制数据到目标目录时出错", errCopy)
				err = editDataroot(srcDir)
				if err != nil {
					return errors.Join(fmt.Errorf("修改配置文件出错: "), err)
				}
				resp, err := execCMD(nil, "systemctl start docker")
				if err != nil {
					logger.SugarLogger.Panicln(resp)
				}
				return errCopy
			}
		}

		// 重启docker
		resp, err = execCMD(nil, "systemctl start docker")
		if err != nil {
			logger.SugarLogger.Panicln(resp)
		}
		logger.SugarLogger.Infoln("docker已经重启")

		if !sameMountpoint {
			logger.SugarLogger.Infof("为节省磁盘空间, 源目录%s数据将被清理", srcDir)
			err = os.RemoveAll(srcDir)
			if err != nil {
				return err
			}
		}
	}
	return err
}

// 判断	DockerRootDir
func checkSrcISDockerRootDir(ctx context.Context, dockerClient *client.Client) (string, error) {
	dockerInfo, err := dockerClient.Info(ctx)
	return dockerInfo.DockerRootDir, err
}

// 计算文件夹大小
func calcDir(pathDir string) (count uint64, err error) {
	err = filepath.WalkDir(pathDir,
		func(path string, d fs.DirEntry, err error) error {
			var info fs.FileInfo
			if !d.IsDir() {
				info, err = d.Info()
				count += uint64(info.Size())
			}
			return err
		})
	if err != nil {
		return 0, err
	}
	return count, err
}

// 文件夹大小人类可读化
func formatFileSize(size uint64) string {
	switch {
	case size < K:
		return fmt.Sprintf("%d B", size)
	case size < M:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(K))
	case size < G:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(M))
	case size < T:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(G))
	case size < P:
		return fmt.Sprintf("%.2f TB", float64(size)/float64(T))
	default:
		return fmt.Sprintf("%.2f PB", float64(size)/float64(P))
	}
}

// 编辑配置文件
func editDataroot(destDir string) error {
	// 读取/etc/docker/daemon.json
	if _, err := os.Stat("/etc/docker/daemon.json"); os.IsNotExist(err) {
		logger.SugarLogger.Infoln("配置文件/etc/docker/daemon.json不存在")
		if _, err := os.Stat("/etc/docker"); os.IsNotExist(err) {
			os.Mkdir("/etc/docker", 0775)
		}
		// 增加"data-root": "/path/to/user/defined/docker" 到json中
		jsonStr := `{"data-root": "` + destDir + `"}`
		f, err := os.Create("/etc/docker/daemon.json")
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.Write([]byte(jsonStr))
		if err != nil {
			return err
		}
		logger.SugarLogger.Infoln("配置文件/etc/docker/daemon.json已经创建")
		return nil
	} else {
		content, err := os.ReadFile("/etc/docker/daemon.json")
		if err != nil {
			return err
		}
		var config map[string]interface{}
		err = json.Unmarshal(content, &config)
		if err != nil {
			logger.SugarLogger.Errorln("解析/etc/docker/daemon.json配置文件失败! ")
			return err
		}
		if dataRootNow, ok := config["data-root"]; ok {
			// 修改json中key为"data-root"的value为destDir
			logger.SugarLogger.Infoln("存在data-root配置, 目前值为:", dataRootNow, "将被修改为:", destDir)
			config["data-root"] = destDir
		} else if graph, ok := config["graph"]; ok {
			// 修改json中key为"graph"的value为destDir
			logger.SugarLogger.Infoln("存在graph配置(老版本), 目前值为:", graph, "将被修改为:", destDir)
			config["graph"] = destDir
		} else {
			// 增加一个kv，"data-root": destDir
			logger.SugarLogger.Infoln("配置文件中data-root配置不存在, 将增加为: ", destDir)
			config["data-root"] = destDir
		}
		// 回写配置文件
		updatedJsonStr, err := json.MarshalIndent(config, "", "    ")
		if err != nil {
			return err
		}
		err = os.WriteFile("/etc/docker/daemon.json", updatedJsonStr, 0644)
		return err
	}
}

// 判断是不是同一个挂载点
func isSameMountpoint(srcDir, destDir string) bool {
	srcMnt := getMountPoint(srcDir)
	desMnt := getMountPoint(destDir)
	logger.SugarLogger.Infof("源目录%s的挂载点为: %s; 目标目录%s的挂载点为: %s", srcDir, srcMnt, destDir, desMnt)
	return srcMnt == desMnt
}

// 根据传入的文件夹路径获取文件夹所在的磁盘(或者说文件系统挂载点)
func getMountPoint(dirPath string) (mountPoint string) {
	disks, err := disk.Partitions(true)
	var containsMnt []string
	if err != nil {
		logger.SugarLogger.Panicln(err)
	}
	for _, disk := range disks {
		if strings.Contains(disk.Device, "/dev/") && disk.Mountpoint != "/boot" {
			if strings.Contains(dirPath, disk.Mountpoint) {
				containsMnt = append(containsMnt, disk.Mountpoint)
			}
		}
	}
	for _, mnt := range containsMnt {
		if len(mnt) > len(mountPoint) {
			mountPoint = mnt
		}
	}

	return mountPoint
}

// 复制文件夹及其下面的文件到destDir
func copyDir(srcDir, destDir string) error {
	err := os.MkdirAll(destDir, 0755)
	if err != nil {
		return fmt.Errorf("创建目标文件夹失败：%v", err)
	}

	// golang的整个文件夹拷贝需要遍历和处理各种类型以及异常实在太蠢了, 不如直接使用bash命令
	files, _ := filepath.Glob(srcDir + "/*")
	cpArgs := append([]string{"cp", "-r"}, files...)
	resp, err := execCMD(nil, strings.Join(cpArgs, " ")+" "+destDir)
	if err != nil {
		logger.SugarLogger.Errorln(resp, err)
	}
	return err
}
