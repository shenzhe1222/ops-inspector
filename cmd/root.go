package cmd

import (
	"fmt"
	"os"
)

func Execute() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	switch args[0] {
	case "host":
		return runHost(args[1:])
	case "disk":
		return runDisk(args[1:])
	case "password":
		return runPassword(args[1:])
	case "help", "--help", "-h":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func printUsage() {
	fmt.Println("Usage: ops-inspector <command> [subcommand]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  host info    查看主机基础信息")
	fmt.Println("  host check   检查主机资源使用情况")
	fmt.Println("  disk usage   查看磁盘使用情况")
	fmt.Println("  disk analyze 分析指定目录的磁盘使用情况")
	fmt.Println("  password generate <length> 生成指定长度的随机密码（9-63 位）")
	fmt.Println("  help         查看帮助")
}
