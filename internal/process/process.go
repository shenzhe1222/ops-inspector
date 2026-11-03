package process

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Process struct {
	PID           int
	Name          string
	PPID          int
	State         byte
	Threads       int
	UserTicks     uint64
	SystemTicks   uint64
	VirtualBytes  uint64
	ResidentBytes uint64
	CommandLine   string
}

type Thread struct {
	TID   int
	Name  string
	State byte
}

type TreeNode struct {
	Process  Process
	Children []*TreeNode
}

type Stats struct {
	Total           int
	Running         int
	Sleeping        int
	Uninterruptible int
	Zombie          int
	Stopped         int
	Idle            int
	Other           int
}

type Usage struct {
	Process       Process
	CPUPercent    float64
	MemoryPercent float64
}

type Reader struct {
	root string
}

func NewReader(procRoot string) *Reader {
	if procRoot == "" {
		procRoot = "/proc"
	}
	return &Reader{root: procRoot}
}

func (r *Reader) List() ([]Process, error) {
	entries, err := os.ReadDir(r.root)
	if err != nil {
		return nil, fmt.Errorf("read process directory %s: %w", r.root, err)
	}

	processes := make([]Process, 0, len(entries))
	for _, entry := range entries {
		pid, ok := parsePID(entry.Name())
		if !ok {
			continue
		}

		proc, err := r.readProcess(pid)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read process %d: %w", pid, err)
		}
		processes = append(processes, proc)
	}

	sort.Slice(processes, func(i, j int) bool {
		return processes[i].PID < processes[j].PID
	})
	return processes, nil
}

func (r *Reader) Info(pid int) (Process, error) {
	proc, err := r.readProcess(pid)
	if err != nil {
		return Process{}, fmt.Errorf("read process %d: %w", pid, err)
	}

	data, err := os.ReadFile(filepath.Join(r.root, strconv.Itoa(pid), "cmdline"))
	if err != nil {
		return Process{}, fmt.Errorf("read command line for process %d: %w", pid, err)
	}
	proc.CommandLine = strings.TrimSpace(strings.ReplaceAll(string(data), "\x00", " "))
	if proc.CommandLine == "" {
		proc.CommandLine = proc.Name
	}
	return proc, nil
}

func (r *Reader) Threads(pid int) ([]Thread, error) {
	dir := filepath.Join(r.root, strconv.Itoa(pid), "task")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read threads for process %d: %w", pid, err)
	}

	threads := make([]Thread, 0, len(entries))
	for _, entry := range entries {
		tid, ok := parsePID(entry.Name())
		if !ok {
			continue
		}
		proc, err := readStat(filepath.Join(dir, entry.Name(), "stat"))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read thread %d: %w", tid, err)
		}
		threads = append(threads, Thread{TID: tid, Name: proc.Name, State: proc.State})
	}

	sort.Slice(threads, func(i, j int) bool {
		return threads[i].TID < threads[j].TID
	})
	return threads, nil
}

func (r *Reader) FileDescriptorCount(pid int) (int, error) {
	dir := filepath.Join(r.root, strconv.Itoa(pid), "fd")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("count file descriptors for process %d: %w", pid, err)
	}
	return len(entries), nil
}

func (r *Reader) Tree(pid int) (*TreeNode, error) {
	processes, err := r.List()
	if err != nil {
		return nil, err
	}

	nodes := make(map[int]*TreeNode, len(processes))
	children := make(map[int][]int, len(processes))
	for _, proc := range processes {
		nodes[proc.PID] = &TreeNode{Process: proc}
		children[proc.PPID] = append(children[proc.PPID], proc.PID)
	}
	root, ok := nodes[pid]
	if !ok {
		return nil, fmt.Errorf("process %d not found", pid)
	}

	var attachChildren func(*TreeNode, map[int]bool)
	attachChildren = func(node *TreeNode, visited map[int]bool) {
		visited[node.Process.PID] = true
		for _, childPID := range children[node.Process.PID] {
			child := nodes[childPID]
			if visited[childPID] {
				continue
			}
			node.Children = append(node.Children, child)
			attachChildren(child, visited)
		}
	}
	attachChildren(root, make(map[int]bool))
	return root, nil
}

func (r *Reader) Stats() (Stats, error) {
	processes, err := r.List()
	if err != nil {
		return Stats{}, err
	}

	stats := Stats{Total: len(processes)}
	for _, proc := range processes {
		switch proc.State {
		case 'R':
			stats.Running++
		case 'S':
			stats.Sleeping++
		case 'D':
			stats.Uninterruptible++
		case 'Z':
			stats.Zombie++
		case 'T', 't':
			stats.Stopped++
		case 'I':
			stats.Idle++
		default:
			stats.Other++
		}
	}
	return stats, nil
}

func (r *Reader) Top(interval time.Duration) ([]Usage, []Usage, error) {
	if interval <= 0 {
		return nil, nil, fmt.Errorf("sampling interval must be greater than zero")
	}

	firstCPU, _, err := r.readCPUCounters()
	if err != nil {
		return nil, nil, err
	}
	first, err := r.List()
	if err != nil {
		return nil, nil, err
	}

	time.Sleep(interval)

	secondCPU, cpuCount, err := r.readCPUCounters()
	if err != nil {
		return nil, nil, err
	}
	second, err := r.List()
	if err != nil {
		return nil, nil, err
	}
	totalMemory, err := r.readTotalMemory()
	if err != nil {
		return nil, nil, err
	}
	if totalMemory == 0 {
		return nil, nil, fmt.Errorf("total memory must be greater than zero")
	}
	if secondCPU <= firstCPU {
		return nil, nil, fmt.Errorf("system CPU counters did not advance during sampling")
	}

	usage := calculateUsage(first, second, secondCPU-firstCPU, cpuCount, totalMemory)
	cpuRanking := append([]Usage(nil), usage...)
	memoryRanking := append([]Usage(nil), usage...)
	sort.Slice(cpuRanking, func(i, j int) bool {
		return cpuRanking[i].CPUPercent > cpuRanking[j].CPUPercent
	})
	sort.Slice(memoryRanking, func(i, j int) bool {
		return memoryRanking[i].Process.ResidentBytes > memoryRanking[j].Process.ResidentBytes
	})
	return cpuRanking, memoryRanking, nil
}

func calculateUsage(first, second []Process, totalCPUTicks uint64, cpuCount int, totalMemory uint64) []Usage {
	firstByPID := make(map[int]Process, len(first))
	for _, proc := range first {
		firstByPID[proc.PID] = proc
	}

	usage := make([]Usage, 0, len(second))
	for _, current := range second {
		item := Usage{Process: current}
		if previous, ok := firstByPID[current.PID]; ok && totalCPUTicks > 0 && cpuCount > 0 {
			previousTicks := previous.UserTicks + previous.SystemTicks
			currentTicks := current.UserTicks + current.SystemTicks
			if currentTicks >= previousTicks {
				item.CPUPercent = float64(currentTicks-previousTicks) *
					float64(cpuCount) * 100 / float64(totalCPUTicks)
			}
		}
		if totalMemory > 0 {
			item.MemoryPercent = float64(current.ResidentBytes) * 100 / float64(totalMemory)
		}
		usage = append(usage, item)
	}
	return usage
}

func (r *Reader) readProcess(pid int) (Process, error) {
	proc, err := readStat(filepath.Join(r.root, strconv.Itoa(pid), "stat"))
	if err != nil {
		return Process{}, err
	}
	if proc.PID != pid {
		return Process{}, fmt.Errorf("process stat PID mismatch: got %d", proc.PID)
	}
	return proc, nil
}

func (r *Reader) readCPUCounters() (uint64, int, error) {
	file, err := os.Open(filepath.Join(r.root, "stat"))
	if err != nil {
		return 0, 0, fmt.Errorf("read system CPU counters: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var total uint64
	cpuCount := 0
	foundAggregate := false
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 {
			continue
		}
		if fields[0] == "cpu" {
			if len(fields) < 5 {
				return 0, 0, fmt.Errorf("invalid system CPU counters")
			}
			foundAggregate = true
			limit := len(fields) - 1
			if limit > 8 {
				limit = 8
			}
			for _, field := range fields[1 : limit+1] {
				value, err := strconv.ParseUint(field, 10, 64)
				if err != nil {
					return 0, 0, fmt.Errorf("parse system CPU counters: %w", err)
				}
				total += value
			}
			continue
		}
		if strings.HasPrefix(fields[0], "cpu") {
			_, err := strconv.Atoi(strings.TrimPrefix(fields[0], "cpu"))
			if err == nil {
				cpuCount++
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, 0, fmt.Errorf("read system CPU counters: %w", err)
	}
	if !foundAggregate {
		return 0, 0, fmt.Errorf("system CPU counters are missing")
	}
	if cpuCount == 0 {
		cpuCount = 1
	}
	return total, cpuCount, nil
}

func (r *Reader) readTotalMemory() (uint64, error) {
	file, err := os.Open(filepath.Join(r.root, "meminfo"))
	if err != nil {
		return 0, fmt.Errorf("read total memory: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[0] == "MemTotal:" {
			kilobytes, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				return 0, fmt.Errorf("parse total memory: %w", err)
			}
			return kilobytes * 1024, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("read total memory: %w", err)
	}
	return 0, fmt.Errorf("total memory is missing")
}

func readStat(path string) (Process, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Process{}, err
	}
	return parseStat(string(data))
}

func parseStat(data string) (Process, error) {
	openParen := strings.IndexByte(data, '(')
	closeParen := strings.LastIndexByte(data, ')')
	if openParen <= 0 || closeParen <= openParen {
		return Process{}, fmt.Errorf("invalid process stat")
	}

	pid, err := strconv.Atoi(strings.TrimSpace(data[:openParen]))
	if err != nil {
		return Process{}, fmt.Errorf("parse process PID: %w", err)
	}
	fields := strings.Fields(data[closeParen+1:])
	if len(fields) < 22 || len(fields[0]) != 1 {
		return Process{}, fmt.Errorf("invalid process stat fields")
	}

	parseInt := func(field string) (int64, error) {
		return strconv.ParseInt(field, 10, 64)
	}
	parseUint := func(field string) (uint64, error) {
		return strconv.ParseUint(field, 10, 64)
	}

	ppid, err := parseInt(fields[1])
	if err != nil {
		return Process{}, fmt.Errorf("parse parent PID: %w", err)
	}
	userTicks, err := parseUint(fields[11])
	if err != nil {
		return Process{}, fmt.Errorf("parse user CPU time: %w", err)
	}
	systemTicks, err := parseUint(fields[12])
	if err != nil {
		return Process{}, fmt.Errorf("parse system CPU time: %w", err)
	}
	threadCount, err := parseInt(fields[17])
	if err != nil {
		return Process{}, fmt.Errorf("parse thread count: %w", err)
	}
	virtualBytes, err := parseUint(fields[20])
	if err != nil {
		return Process{}, fmt.Errorf("parse virtual memory: %w", err)
	}
	residentPages, err := parseInt(fields[21])
	if err != nil {
		return Process{}, fmt.Errorf("parse resident memory: %w", err)
	}

	var residentBytes uint64
	if residentPages > 0 {
		residentBytes = uint64(residentPages) * uint64(os.Getpagesize())
	}
	return Process{
		PID:           pid,
		Name:          data[openParen+1 : closeParen],
		PPID:          int(ppid),
		State:         fields[0][0],
		Threads:       int(threadCount),
		UserTicks:     userTicks,
		SystemTicks:   systemTicks,
		VirtualBytes:  virtualBytes,
		ResidentBytes: residentBytes,
	}, nil
}

func parsePID(value string) (int, bool) {
	pid, err := strconv.Atoi(value)
	return pid, err == nil && pid > 0
}
