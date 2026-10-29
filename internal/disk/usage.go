package disk

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
)

type Usage struct {
	Filesystem string
	MountPoint string
	Total      uint64
	Used       uint64
	Available  uint64
	Percent    float64
	Status     string
}

func ListUsage() ([]Usage, error) {
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, fmt.Errorf("read mounts: %w", err)
	}
	defer file.Close()

	var results []Usage
	seen := make(map[string]bool)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}

		filesystem := fields[0]
		mountPoint := fields[1]
		fsType := fields[2]

		// 跳过虚拟文件系统
		switch fsType {
		case "proc", "sysfs", "devtmpfs", "devpts",
			"tmpfs", "cgroup", "cgroup2", "securityfs",
			"pstore", "debugfs", "tracefs", "configfs",
			"mqueue", "hugetlbfs", "rpc_pipefs",
			"autofs", "fusectl":
			continue
		}

		if seen[mountPoint] {
			continue
		}
		seen[mountPoint] = true

		var stat syscall.Statfs_t
		if err := syscall.Statfs(mountPoint, &stat); err != nil {
			continue
		}

		total := stat.Blocks * uint64(stat.Bsize)
		free := stat.Bfree * uint64(stat.Bsize)
		available := stat.Bavail * uint64(stat.Bsize)

		var used uint64
		if total >= free {
			used = total - free
		}

		var percent float64
		if total > 0 {
			percent = float64(used) / float64(total) * 100
		}

		status := "OK"
		if percent >= 90 {
			status = "CRITICAL"
		} else if percent >= 80 {
			status = "WARN"
		}

		results = append(results, Usage{
			Filesystem: filesystem,
			MountPoint: mountPoint,
			Total:      total,
			Used:       used,
			Available:  available,
			Percent:    percent,
			Status:     status,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan mounts: %w", err)
	}

	return results, nil
}

func PrintUsage(results []Usage) {
	const gib = 1024 * 1024 * 1024

	fmt.Printf("%-25s %-20s %10s %10s %10s %8s %10s\n",
		"Filesystem", "Mount", "Total", "Used", "Available", "Usage", "Status")

	for _, item := range results {
		fmt.Printf("%-25s %-20s %9.2fG %9.2fG %9.2fG %7.1f%% %10s\n",
			item.Filesystem,
			item.MountPoint,
			float64(item.Total)/float64(gib),
			float64(item.Used)/float64(gib),
			float64(item.Available)/float64(gib),
			item.Percent,
			item.Status,
		)
	}
}