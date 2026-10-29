package host

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type MemoryCheck struct {
	Total     uint64
	Used      uint64
	Available uint64
	Usage     float64
	Status    string
}

func CheckMemory() (*MemoryCheck, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, fmt.Errorf("read memory: %w", err)
	}
	defer file.Close()

	var totalKB uint64
	var availableKB uint64

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}

		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse memory info: %w", err)
		}

		switch fields[0] {
		case "MemTotal:":
			totalKB = value
		case "MemAvailable:":
			availableKB = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read memory info: %w", err)
	}

	if totalKB == 0 || availableKB > totalKB {
		return nil, fmt.Errorf("invalid memory information")
	}

	total := totalKB * 1024
	available := availableKB * 1024
	used := total - available

	usage := float64(used) / float64(total) * 100

	status := "OK"
	if usage >= 90 {
		status = "CRITICAL"
	} else if usage >= 80 {
		status = "WARN"
	}

	return &MemoryCheck{
		Total:     total,
		Used:      used,
		Available: available,
		Usage:     usage,
		Status:    status,
	}, nil
}

func (m *MemoryCheck) Print() {
	const gib = 1024 * 1024 * 1024

	fmt.Println("========= Host Memory Check =========")
	fmt.Printf("Total:         %.2f GiB\n", float64(m.Total)/float64(gib))
	fmt.Printf("Used:          %.2f GiB\n", float64(m.Used)/float64(gib))
	fmt.Printf("Available:     %.2f GiB\n", float64(m.Available)/float64(gib))
	fmt.Printf("Usage:         %.1f%%\n", m.Usage)
	fmt.Printf("Status:        %s\n", m.Status)
	fmt.Println("=====================================")
}