package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	
	"ops-inspector/internal/host"
)

var hostCmd = &cobra.Command{
	Use:   "host",
	Short: "主机信息与检查",
}

var hostInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "查看主机基础信息",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		var info host.HostInfo

		if err := info.CheckInfo(); err != nil {
			return err
		}

		info.Print()
		return nil
	},
}

var hostCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "检查主机资源使用情况",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		disk, err := host.CheckDisk("/")
		if err != nil {
			return err
		}

		disk.Print()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(hostCmd)
	hostCmd.AddCommand(hostInfoCmd)
	hostCmd.AddCommand(hostCheckCmd)
}

func printError(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

