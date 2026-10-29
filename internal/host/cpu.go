package host

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type CPUCheck struct {
	Usage  float64
	Status string
}

func CheckCPU() (*CPUCheck, error) {
	total1, idle1, err := readCPUStat()
	if err != nil {
		return nil, err
	}

	time.Sleep(time.Second)

	total2, idle2, err := readCPUStat()
	if err != nil {
		return nil, err
	}

	totalDelta := total2 - total1
	idleDelta := idle2 - idle1

	if totalDelta == 0 {
		return nil, fmt.Errorf("cannot calculate CPU usage")
	}

	usage := float64(totalDelta-idleDelta) / float64(totalDelta) * 100

	status := "OK"
	if usage >= 90 {
		status = "CRITICAL"
	} else if usage >= 80 {
		status = "WARN"
	}

	return &CPUCheck{
		Usage:  usage,
		Status: status,
	}, nil
}

func readCPUStat() (uint64, uint64, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0, fmt.Errorf("read CPU stat: %w", err)
	}

	fields := strings.Fields(strings.SplitN(string(data), "\n", 2)[0])

	if len(fields) < 8 || fields[0] != "cpu" {
		return 0, 0, fmt.Errorf("invalid CPU stat")
	}

	var total uint64
	var idle uint64

	for i, field := range fields[1:8] {
		value, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("parse CPU stat: %w", err)
		}

		total += value

		if i == 3 || i == 4 {
			idle += value
		}
	}

	return total, idle, nil
}

func (c *CPUCheck) Print() {
	fmt.Println("=========  Host CPU Check  =========")
	fmt.Printf("Usage:         %.1f%%\n", c.Usage)
	fmt.Printf("Status:        %s\n", c.Status)
}