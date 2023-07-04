/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"myTool/swarmopt"

	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

// unconstaintSvcCmd represents the unconstaintSvc command
var unconstraintSvcCmd = &cobra.Command{
	Use:   "unconstaintSvc",
	Short: "取消约束GeoGlobe Server 服务",
	Long: `unconstaintSvc: 
  取消约束GeoGlobe Server 服务
  可以使得在节点故障时服务漂移到其他节点提供服务
  但是如果跨节点Overlay网络存在问题, 服务可能存在不可访问的情况 `,
	Run: func(cmd *cobra.Command, args []string) {
		var ctx = context.Background()
		dockerClient, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation(), client.WithVersion(""))
		if err != nil {
			panic(err)
		}
		swarmopt.UnConstraitAll(ctx, dockerClient)
	},
}

func init() {
	rootCmd.AddCommand(unconstraintSvcCmd)
}
