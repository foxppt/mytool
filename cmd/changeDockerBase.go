/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"myTool/logger"
	"myTool/operation"

	"github.com/spf13/cobra"
)

var srcDir string
var distDir string

// changeDockerBaseCmd represents the changeDockerBase command
var changeDockerBaseCmd = &cobra.Command{
	Use:   "changeDockerBase",
	Short: "这个命令可以迁移docker的目录",
	Long:  `本命令支持两个参数-s/--source指定原始目录, -d/--distination指定目标目录, 如果原始目录未指定则为默认的/var/lib/docker. `,
	Run: func(cmd *cobra.Command, args []string) {
		err := operation.ChangeDockerBaseDir(srcDir, distDir)
		if err != nil {
			logger.SugarLogger.Panicln(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(changeDockerBaseCmd)
	changeDockerBaseCmd.Flags().StringVarP(&srcDir, "source", "s", "/var/lib/docker", "docker数据目录的源目录")
	changeDockerBaseCmd.Flags().StringVarP(&distDir, "distination", "d", "", "docker数据目录的目标目录")
}
