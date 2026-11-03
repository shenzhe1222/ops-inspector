package process

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (r *Reader) List() ([]ProcessInfo, error) {
	processes, err := r.readStats()
	if err != nil {
		return nil, err
	}
	totalMemory, bootTime, err := r.readSystemInfo()
	if err != nil {
		return nil, err
	}
	users, err := readUsers()
	if err != nil {
		return nil, err
	}

	result := make([]ProcessInfo, 0, len(processes))
	for _, proc := range processes {
		info, err := r.enrich(proc, totalMemory, bootTime, users)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		result = append(result, info)
	}
	return result, nil
}

func (r *Reader) readStats() ([]statInfo, error) {
	entries, err := os.ReadDir(r.root)
	if err != nil {
		return nil, fmt.Errorf("read process directory %s: %w", r.root, err)
	}

	processes := make([]statInfo, 0, len(entries))
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 {
			continue
		}
		proc, err := r.readStat(pid)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		processes = append(processes, proc)
	}
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].PID < processes[j].PID
	})
	return processes, nil
}

func (r *Reader) enrich(proc statInfo, totalMemory, bootTime uint64, users map[string]string) (ProcessInfo, error) {
	dir := filepath.Join(r.root, strconv.Itoa(proc.PID))
	status, err := os.ReadFile(filepath.Join(dir, "status"))
	if err != nil {
		return ProcessInfo{}, processReadError(proc.PID, err)
	}
	uid, err := parseUID(string(status))
	if err != nil {
		return ProcessInfo{}, fmt.Errorf("invalid status data for process %d: %w", proc.PID, err)
	}
	user := strconv.FormatUint(uid, 10)
	if name, ok := users[user]; ok {
		user = name
	}

	commandLine, err := os.ReadFile(filepath.Join(dir, "cmdline"))
	if err != nil {
		return ProcessInfo{}, processReadError(proc.PID, err)
	}
	command := strings.TrimSpace(strings.ReplaceAll(string(commandLine), "\x00", " "))
	if command == "" {
		command = "[" + proc.Name + "]"
	}

	proc.User = user
	proc.Command = command
	proc.MemoryPct = float64(proc.MemoryRSS) * 100 / float64(totalMemory)
	startSeconds := proc.startTicks / clockTicksPerSecond
	startNanos := proc.startTicks % clockTicksPerSecond * uint64(time.Second) / clockTicksPerSecond
	proc.StartTime = time.Unix(
		int64(bootTime+startSeconds),
		int64(startNanos),
	)
	return proc.ProcessInfo, nil
}

func parseUID(status string) (uint64, error) {
	for _, line := range strings.Split(status, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == "Uid:" {
			uid, err := strconv.ParseUint(fields[2], 10, 64)
			if err != nil {
				return 0, fmt.Errorf("parse UID: %w", err)
			}
			return uid, nil
		}
	}
	return 0, errors.New("UID field is missing")
}

func readUsers() (map[string]string, error) {
	data, err := os.ReadFile("/etc/passwd")
	if err != nil {
		return nil, fmt.Errorf("read user names: %w", err)
	}
	users := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Split(line, ":")
		if len(fields) >= 3 && fields[0] != "" {
			users[fields[2]] = fields[0]
		}
	}
	return users, nil
}

func (r *Reader) readSystemInfo() (uint64, uint64, error) {
	meminfo, err := os.ReadFile(filepath.Join(r.root, "meminfo"))
	if err != nil {
		return 0, 0, fmt.Errorf("read total memory: %w", err)
	}
	var totalMemory uint64
	for _, line := range strings.Split(string(meminfo), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "MemTotal:" {
			kilobytes, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				return 0, 0, fmt.Errorf("invalid total memory: %w", err)
			}
			totalMemory = kilobytes * 1024
			break
		}
	}
	if totalMemory == 0 {
		return 0, 0, errors.New("total memory is missing or zero")
	}

	stat, err := os.ReadFile(filepath.Join(r.root, "stat"))
	if err != nil {
		return 0, 0, fmt.Errorf("read system boot time: %w", err)
	}
	for _, line := range strings.Split(string(stat), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "btime" {
			bootTime, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				return 0, 0, fmt.Errorf("invalid system boot time: %w", err)
			}
			return totalMemory, bootTime, nil
		}
	}
	return 0, 0, errors.New("system boot time is missing")
}
