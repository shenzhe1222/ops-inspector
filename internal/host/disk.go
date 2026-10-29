package host

import (
	"fmt"
	"syscall"
)

type DiskCheck struct {
	Path       string
	Total      uint64
	Used       uint64
	Available  uint64
	Usage      float64
	Status     string
}

func CheckDisk(path string) (*DiskCheck, error) {
	var stat syscall.Statfs_t

	if err := syscall.Statfs(path, &stat); err != nil {
		return nil, fmt.Errorf("check disk %s: %w", path, err)
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	available := stat.Bavail * uint64(stat.Bsize)

	var used uint64
	if total >= free {
		used = total - free
	}

	var usage float64
	if total > 0 {
		usage = float64(used) / float64(total) * 100
	}

	status := "OK"
	if usage >= 90 {
		status = "CRITICAL"
	} else if usage >= 80 {
		status = "WARN"
	}

	return &DiskCheck{
		Path:      path,
		Total:     total,
		Used:      used,
		Available: available,
		Usage:     usage,
		Status:    status,
	}, nil
}

func (d *DiskCheck) Print() {
	const gib = 1024 * 1024 * 1024

	fmt.Println("========= Host Resource Check =========")
	fmt.Printf("Filesystem:    %s\n", d.Path)
	fmt.Printf("Total:         %.2f GiB\n", float64(d.Total)/float64(gib))
	fmt.Printf("Used:          %.2f GiB\n", float64(d.Used)/float64(gib))
	fmt.Printf("Available:     %.2f GiB\n", float64(d.Available)/float64(gib))
	fmt.Printf("Usage:         %.1f%%\n", d.Usage)
	fmt.Printf("Status:        %s\n", d.Status)
	fmt.Println("========================================")
}