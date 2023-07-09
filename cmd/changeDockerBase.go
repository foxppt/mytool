/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"myTool/logger"
	"myTool/operation"
	"os"

	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

var srcDir string
var destDir string

// changeDockerBaseCmd represents the changeDockerBase command
var changeDockerBaseCmd = &cobra.Command{
	Use:   "changeDockerBase",
	Short: "这个命令可以迁移docker的目录",
	Long:  `这个命令可以迁移docker的目录, 指定源目录和目标目录即可.  `,
	Run: func(cmd *cobra.Command, args []string) {
		if destDir == "" {
			logger.SugarLogger.Infoln("目标路径不能为空")
			os.Exit(0)
		}
		if srcDir != destDir {
			var ctx = context.Background()
			dockerClient, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation(), client.WithVersion(""))
			if err != nil {
				panic(err)
			}
			err = operation.ChangeDockerBaseDir(ctx, dockerClient, srcDir, destDir)
			if err != nil {
				logger.SugarLogger.Panicln(err)
			}
		} else {
			logger.SugarLogger.Infof("目标目录%s与源目录%s相同, 程序不做处理. ", destDir, srcDir)
		}
	},
}

func init() {
	rootCmd.AddCommand(changeDockerBaseCmd)
	changeDockerBaseCmd.Flags().StringVarP(&srcDir, "source", "s", "/var/lib/docker", "docker数据目录的源目录")
	changeDockerBaseCmd.Flags().StringVarP(&destDir, "distination", "d", "", "docker数据目录的目标目录")
}
