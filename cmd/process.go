package cmd

import (
	"fmt"
	"strconv"
	"time"

	"ops-inspector/internal/process"
)

func runProcess(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ops-inspector -p list|info|top|threads|fd|tree|stats")
	}

	reader := process.NewReader("/proc")
	switch args[0] {
	case "list":
		if len(args) != 1 {
			return fmt.Errorf("usage: ops-inspector -p list")
		}
		processes, err := reader.List()
		if err != nil {
			return err
		}
		printProcesses(processes)
	case "info":
		pid, err := processPID(args, "info")
		if err != nil {
			return err
		}
		proc, err := reader.Info(pid)
		if err != nil {
			return err
		}
		printProcessInfo(proc)
	case "top":
		if len(args) != 1 {
			return fmt.Errorf("usage: ops-inspector -p top")
		}
		cpu, memory, err := reader.Top(time.Second)
		if err != nil {
			return err
		}
		printProcessTop("CPU", cpu, func(item process.Usage) string {
			return fmt.Sprintf("%.1f%%", item.CPUPercent)
		})
		printProcessTop("Memory", memory, func(item process.Usage) string {
			return fmt.Sprintf("%.1f%%", item.MemoryPercent)
		})
	case "threads":
		pid, err := processPID(args, "threads")
		if err != nil {
			return err
		}
		threads, err := reader.Threads(pid)
		if err != nil {
			return err
		}
		fmt.Printf("TID\tSTATE\tNAME\n")
		for _, thread := range threads {
			fmt.Printf("%d\t%c\t%s\n", thread.TID, thread.State, thread.Name)
		}
	case "fd":
		pid, err := processPID(args, "fd")
		if err != nil {
			return err
		}
		count, err := reader.FileDescriptorCount(pid)
		if err != nil {
			return err
		}
		fmt.Printf("PID %d open file descriptors: %d\n", pid, count)
	case "tree":
		pid, err := processPID(args, "tree")
		if err != nil {
			return err
		}
		tree, err := reader.Tree(pid)
		if err != nil {
			return err
		}
		printProcessTree(tree, "", true, true)
	case "stats":
		if len(args) != 1 {
			return fmt.Errorf("usage: ops-inspector -p stats")
		}
		stats, err := reader.Stats()
		if err != nil {
			return err
		}
		printProcessStats(stats)
	default:
		return fmt.Errorf("unknown process command %q; use list, info, top, threads, fd, tree, or stats", args[0])
	}
	return nil
}

func processPID(args []string, command string) (int, error) {
	if len(args) != 2 {
		return 0, fmt.Errorf("usage: ops-inspector -p %s <pid>", command)
	}
	pid, err := strconv.Atoi(args[1])
	if err != nil || pid <= 0 {
		return 0, fmt.Errorf("invalid PID %q: must be a positive integer", args[1])
	}
	return pid, nil
}

func printProcesses(processes []process.Process) {
	fmt.Printf("%-8s %-8s %-7s %-12s %s\n", "PID", "PPID", "STATE", "RSS", "COMMAND")
	for _, proc := range processes {
		command := proc.Name
		if command == "" {
			command = "-"
		}
		fmt.Printf("%-8d %-8d %-7c %-12s %s\n",
			proc.PID, proc.PPID, proc.State, humanBytes(proc.ResidentBytes), command)
	}
}

func printProcessInfo(proc process.Process) {
	fmt.Printf("PID:           %d\n", proc.PID)
	fmt.Printf("Name:          %s\n", proc.Name)
	fmt.Printf("PPID:          %d\n", proc.PPID)
	fmt.Printf("State:         %c (%s)\n", proc.State, processStateName(proc.State))
	fmt.Printf("Threads:       %d\n", proc.Threads)
	fmt.Printf("Virtual Memory: %s\n", humanBytes(proc.VirtualBytes))
	fmt.Printf("Resident Memory: %s\n", humanBytes(proc.ResidentBytes))
	fmt.Printf("User CPU Ticks: %d\n", proc.UserTicks)
	fmt.Printf("System CPU Ticks: %d\n", proc.SystemTicks)
	fmt.Printf("Command Line:  %s\n", proc.CommandLine)
}

func printProcessTop(title string, ranking []process.Usage, metric func(process.Usage) string) {
	fmt.Printf("========= Top Processes by %s =========\n", title)
	fmt.Printf("%-8s %-12s %-12s %s\n", "PID", title, "RSS", "COMMAND")
	limit := len(ranking)
	if limit > 10 {
		limit = 10
	}
	for _, item := range ranking[:limit] {
		fmt.Printf("%-8d %-12s %-12s %s\n", item.Process.PID, metric(item),
			humanBytes(item.Process.ResidentBytes), item.Process.Name)
	}
}

func printProcessTree(node *process.TreeNode, prefix string, last, root bool) {
	if root {
		fmt.Printf("%d %s (state %c, ppid %d)\n",
			node.Process.PID, node.Process.Name, node.Process.State, node.Process.PPID)
	} else {
		branch := "├── "
		if last {
			branch = "└── "
		}
		fmt.Printf("%s%s%d %s (state %c)\n",
			prefix, branch, node.Process.PID, node.Process.Name, node.Process.State)
	}

	childPrefix := prefix
	if !root && last {
		childPrefix += "    "
	} else if !root {
		childPrefix += "│   "
	}
	for i, child := range node.Children {
		printProcessTree(child, childPrefix, i == len(node.Children)-1, false)
	}
}

func printProcessStats(stats process.Stats) {
	fmt.Println("========= Process State Statistics =========")
	fmt.Printf("Total:             %d\n", stats.Total)
	fmt.Printf("Running (R):       %d\n", stats.Running)
	fmt.Printf("Sleeping (S):      %d\n", stats.Sleeping)
	fmt.Printf("Uninterruptible (D): %d\n", stats.Uninterruptible)
	fmt.Printf("Zombie (Z):        %d\n", stats.Zombie)
	fmt.Printf("Stopped (T/t):     %d\n", stats.Stopped)
	fmt.Printf("Idle (I):          %d\n", stats.Idle)
	fmt.Printf("Other:             %d\n", stats.Other)
}

func processStateName(state byte) string {
	switch state {
	case 'R':
		return "running"
	case 'S':
		return "sleeping"
	case 'D':
		return "uninterruptible sleep"
	case 'Z':
		return "zombie"
	case 'T', 't':
		return "stopped"
	case 'I':
		return "idle"
	default:
		return "other"
	}
}

func humanBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	value := float64(bytes)
	units := []string{"KiB", "MiB", "GiB", "TiB"}
	for _, name := range units {
		value /= unit
		if value < unit || name == "TiB" {
			return fmt.Sprintf("%.1f %s", value, name)
		}
	}
	return fmt.Sprintf("%.1f TiB", value)
}
