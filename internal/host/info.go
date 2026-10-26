package host

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"
)

type HostInfo struct {
	Hostname      string
	OS            string
	KernelVersion string
	Architecture  string
	CPUCores      int
	TotalMemory   uint64
	RootDiskTotal uint64
	Uptime        uint64
}

func (h *HostInfo) CheckInfo() error {
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("get hostname: %w", err)
	}
	h.Hostname = hostname

	h.OS = runtime.GOOS

	kernel, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return fmt.Errorf("read kernel version: %w", err)
	}
	h.KernelVersion = strings.TrimSpace(string(kernel))

	h.Architecture = runtime.GOARCH
	h.CPUCores = runtime.NumCPU()

	memory, err := readMemory()
	if err != nil {
		return fmt.Errorf("read memory: %w", err)
	}
	h.TotalMemory = memory

	disk, err := readDisk("/")
	if err != nil {
		return fmt.Errorf("read root filesystem: %w", err)
	}
	h.RootDiskTotal = disk

	uptime, err := readUptime()
	if err != nil {
		return fmt.Errorf("read uptime: %w", err)
	}
	h.Uptime = uptime

	return nil
}

func (h *HostInfo) Print() {
	const gib = 1024 * 1024 * 1024

	days := h.Uptime / 86400
	hours := (h.Uptime % 86400) / 3600
	minutes := (h.Uptime % 3600) / 60
	seconds := h.Uptime % 60

	fmt.Println("========= Host Basic Information =========")
	fmt.Printf("Hostname:              %s\n", h.Hostname)
	fmt.Printf("Operating System:      %s\n", h.OS)
	fmt.Printf("Kernel Version:        %s\n", h.KernelVersion)
	fmt.Printf("CPU Architecture:      %s\n", h.Architecture)
	fmt.Printf("CPU Logical Cores:     %d\n", h.CPUCores)
	fmt.Printf("Total Memory:          %.2f GiB\n",
		float64(h.TotalMemory)/float64(gib))
	fmt.Printf("Root Filesystem Total: %.2f GiB\n",
		float64(h.RootDiskTotal)/float64(gib))
	fmt.Printf("System Uptime:         %d days, %d hours, %d minutes, %d seconds\n",days, hours, minutes, seconds)
	fmt.Println("==========================================")
}

func readMemory() (uint64, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}

	var kb uint64
	line := strings.SplitN(string(data), "\n", 2)[0]

	if _, err := fmt.Sscanf(line, "MemTotal: %d kB", &kb); err != nil {
		return 0, err
	}

	return kb * 1024, nil
}

func readDisk(path string) (uint64, error) {
	var stat syscall.Statfs_t

	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}

	return stat.Blocks * uint64(stat.Bsize), nil
}

func readUptime() (uint64, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}

	var seconds float64
	if _, err := fmt.Sscanf(string(data), "%f", &seconds); err != nil {
		return 0, err
	}

	return uint64(seconds), nil
}