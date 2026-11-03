package cmd

import (
	"fmt"
	"strconv"

	"ops-inspector/internal/process"
)

func runProcess(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ops-inspector -p list|info|stats")
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
		return fmt.Errorf("unknown process command %q; available commands: list, info, stats", args[0])
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

func printProcesses(processes []process.ProcessInfo) {
	fmt.Printf("%-8s %-8s %-10s %-8s %-12s %-9s %s\n",
		"PID", "PPID", "USER", "STATE", "RSS", "THREADS", "COMMAND")
	for _, proc := range processes {
		fmt.Printf("%-8d %-8d %-10s %-8s %-12s %-9d %s\n",
			proc.PID, proc.PPID, proc.User, proc.State,
			humanBytes(proc.MemoryRSS), proc.Threads, proc.Command)
	}
}

func printProcessInfo(proc process.ProcessInfo) {
	fmt.Println("Process Information")
	fmt.Println()
	fmt.Printf("PID:          %d\n", proc.PID)
	fmt.Printf("PPID:         %d\n", proc.PPID)
	fmt.Printf("Name:         %s\n", proc.Name)
	fmt.Printf("State:        %s\n", process.StateName(proc.State))
	fmt.Printf("User:         %s\n", proc.User)
	fmt.Printf("Memory RSS:   %s\n", humanBytes(proc.MemoryRSS))
	fmt.Printf("Memory:       %.1f%%\n", proc.MemoryPct)
	fmt.Printf("Threads:      %d\n", proc.Threads)
	fmt.Printf("Start Time:   %s\n", proc.StartTime.Local().Format("2006-01-02 15:04:05"))
	fmt.Printf("Command:      %s\n", proc.Command)
}

func printProcessStats(stats process.Stats) {
	fmt.Println("Process State Statistics")
	fmt.Printf("Total:              %d\n", stats.Total)
	fmt.Printf("Running (R):        %d\n", stats.Running)
	fmt.Printf("Sleeping (S):       %d\n", stats.Sleeping)
	fmt.Printf("Uninterruptible (D): %d\n", stats.Uninterruptible)
	fmt.Printf("Zombie (Z):         %d\n", stats.Zombie)
	fmt.Printf("Stopped (T/t):      %d\n", stats.Stopped)
	fmt.Printf("Idle (I):           %d\n", stats.Idle)
	fmt.Printf("Other:              %d\n", stats.Other)
}

func humanBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	value := float64(bytes)
	units := []string{"KiB", "MiB", "GiB", "TiB", "PiB"}
	for _, name := range units {
		value /= unit
		if value < unit || name == units[len(units)-1] {
			return fmt.Sprintf("%.1f %s", value, name)
		}
	}
	return fmt.Sprintf("%.1f PiB", value)
}
