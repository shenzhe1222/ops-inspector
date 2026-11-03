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
	case "-h", "--host", "host":
		return runHost(args[1:])
	case "-d", "--disk", "disk":
		return runDisk(args[1:])
	case "-p", "--process", "process":
		return runProcess(args[1:])
	case "-n", "--network", "network":
		return fmt.Errorf("network module is not implemented yet")
	case "-P", "--password", "password":
		return runPassword(args[1:])
	case "help", "--help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func printUsage() {
	fmt.Println("Usage: ops-inspector [module] <subcommand> [arguments]")
	fmt.Println()
	fmt.Println("Modules:")
	fmt.Println("  -h, --host       主机（Host）")
	fmt.Println("  -d, --disk       磁盘（Disk）")
	fmt.Println("  -p, --process    进程（Process）")
	fmt.Println("  -n, --network    网络（Network）")
	fmt.Println("  -P, --password   密码（Password）")
	fmt.Println("  help, --help     查看帮助")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ops-inspector -h info")
	fmt.Println("  ops-inspector -d usage")
	fmt.Println("  ops-inspector -p list")
	fmt.Println("  ops-inspector -p info 1234")
	fmt.Println("  ops-inspector -p stats")
	fmt.Println("  ops-inspector -n connections")
	fmt.Println("  ops-inspector -P generate <length>  (8-64 位)")
}
