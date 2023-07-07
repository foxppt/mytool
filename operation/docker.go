package operation

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"myTool/logger"
	"os"
	"path/filepath"

	"github.com/shirou/gopsutil/v3/disk"
)

func ChangeDockerBaseDir(srcDir string, destDir string) error {
	// 获取原始路径的文件夹大小
	var sourceSize uint64
	var distSize uint64
	err := filepath.WalkDir(srcDir,
		func(path string, d fs.DirEntry, err error) error {
			if !d.IsDir() {
				info, err := d.Info()
				sourceSize += uint64(info.Size())
				return err
			}
			return nil
		})
	if err != nil {
		return err
	}
	logger.SugarLogger.Infof("目录%s的大小为%d", srcDir, sourceSize)

	// 获取目标路径的可用大小
	diskStatus, err := disk.Usage(destDir)
	if err != nil {
		return err
	}
	distSize = diskStatus.Free

	// 如果目标路径空间不足，直接error返回
	if sourceSize >= distSize {
		return fmt.Errorf("目标路径%s空间不足", destDir)
	} else {
		// 如果目标路径空间充足
		// 停止dockerd
		resp, err := execCMD(nil, "systemctl stop docker")
		if err != nil {
			logger.SugarLogger.Panicln(resp)
		}
		// 修改docker配置文件指向目标目录
		editDataroot(srcDir, destDir)
		// 判断是不是一个挂载盘，如果是同一个，剪切原始路径的文件夹到目标路径
		sameMountpoint := isSameMountpoint(srcDir, destDir)
		if sameMountpoint {
			err = os.Rename(srcDir, destDir)
			if err != nil {
				return err
			}
		} else {
			// 如果不是同一个挂载盘，复制原始路径的文件夹到目标路径, 等复制完成以后再删除原始目录(可以保证文件不损坏)
			errCopy := copyDir(srcDir, destDir)
			if errCopy != nil {
				// 如果拷贝文件失败了得回滚(就是源和目标反过来修改)，不然docker会出问题
				editDataroot(destDir, srcDir)
				resp, err := execCMD(nil, "systemctl start docker")
				if err != nil {
					logger.SugarLogger.Panicln(resp)
				}
				return errCopy
			}
			err = os.RemoveAll(srcDir)
			if err != nil {
				return err
			}
		}

		// 重启docker
		resp, err = execCMD(nil, "systemctl start docker")
		if err != nil {
			logger.SugarLogger.Panicln(resp)
		}
	}
	return nil
}

// 复制文件夹及其下面的文件
func copyDir(srcDir, destDir string) error {
	// 创建目标文件夹
	err := os.MkdirAll(destDir, 0755)
	if err != nil {
		return fmt.Errorf("创建目标文件夹失败：%v", err)
	}
	// 遍历源文件夹
	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 拼接目标文件路径
		destPath := filepath.Join(destDir, path[len(srcDir):])
		if info.IsDir() {
			err = os.MkdirAll(destPath, 0755)
			if err != nil {
				return fmt.Errorf("创建目标文件夹失败：%v", err)
			}
		} else {
			// 复制文件
			err = copyFile(path, destPath)
			if err != nil {
				return fmt.Errorf("复制文件失败：%v", err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("遍历源文件夹失败：%v", err)
	}
	logger.SugarLogger.Infoln("文件拷贝完成")
	return nil
}

// 复制文件
func copyFile(src, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("打开源文件失败：%v", err)
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("创建目标文件失败：%v", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Errorf("复制文件失败：%v", err)
	}

	return nil
}

// 判断是不是同一个挂载点
func isSameMountpoint(srcDir, destDir string) bool {
	return filepath.VolumeName(srcDir) == filepath.VolumeName(destDir)
}

func editDataroot(srcDir, destDir string) {
	// 读取/etc/docker/daemon.json(当然得判断有没有哈)
	if _, err := os.Stat("/etc/docker/daemon.json"); os.IsNotExist(err) {
		if _, err := os.Stat("/etc/docker"); os.IsNotExist(err) {
			os.Mkdir("/etc/docker", 0775)
		}
		// 增加"data-root": "/path/to/user/defined/docker" 到json中
		jsonStr := `{"data-root": "` + destDir + `"}`
		f, err := os.Create("/etc/docker/daemon.json")
		if err != nil {
			panic(err)
		}
		defer f.Close()
		_, err = f.Write([]byte(jsonStr))
		if err != nil {
			panic(err)
		}
	} else {
		content, err := os.ReadFile("/etc/docker/daemon.json")
		if err != nil {
			panic(err)
		}
		var config map[string]interface{}
		err = json.Unmarshal(content, &config)
		if err != nil {
			logger.SugarLogger.Errorln("解析/etc/docker/daemon.json配置文件失败! ")
		}
		if _, ok := config["data-root"]; ok {
			// 修改json中key为"data-root"的value为destDir
			config["data-root"] = destDir
		} else if _, ok := config["graph"]; ok {
			// 修改json中key为"graph"的value为destDir
			config["graph"] = destDir
		} else {
			// 增加一个kv，"data-root": destDir
			config["data-root"] = destDir
		}
		// 回写配置文件
		updatedJsonStr, err := json.MarshalIndent(config, "", "    ")
		if err != nil {
			panic(err)
		}
		err = os.WriteFile("/etc/docker/daemon.json", updatedJsonStr, 0644)
		if err != nil {
			panic(err)
		}
	}
}
