package cmd

import (
	"fmt"
	"os"
	
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ops-inspector",
	Short: "Linux 主机诊断工具",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}