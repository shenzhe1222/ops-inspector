package process

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Linux reports process start time in USER_HZ ticks.
const clockTicksPerSecond = 100

type ProcessInfo struct {
	PID        int
	PPID       int
	Name       string
	State      string
	User       string
	MemoryRSS  uint64
	MemoryPct  float64
	Threads    int
	StartTime  time.Time
	Command    string
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

type Reader struct {
	root string
}

type statInfo struct {
	ProcessInfo
	startTicks uint64
}

func NewReader(procRoot string) *Reader {
	if procRoot == "" {
		procRoot = "/proc"
	}
	return &Reader{root: procRoot}
}

func (r *Reader) readStat(pid int) (statInfo, error) {
	path := filepath.Join(r.root, strconv.Itoa(pid), "stat")
	data, err := os.ReadFile(path)
	if err != nil {
		return statInfo{}, processReadError(pid, err)
	}
	info, err := parseStat(string(data))
	if err != nil {
		return statInfo{}, fmt.Errorf("invalid stat data for process %d: %w", pid, err)
	}
	if info.PID != pid {
		return statInfo{}, fmt.Errorf("invalid stat data for process %d: reported PID %d", pid, info.PID)
	}
	return info, nil
}

func parseStat(data string) (statInfo, error) {
	openParen := strings.IndexByte(data, '(')
	closeParen := strings.LastIndexByte(data, ')')
	if openParen <= 0 || closeParen <= openParen {
		return statInfo{}, errors.New("missing command name delimiters")
	}

	pid, err := strconv.Atoi(strings.TrimSpace(data[:openParen]))
	if err != nil || pid <= 0 {
		return statInfo{}, errors.New("invalid PID")
	}
	fields := strings.Fields(data[closeParen+1:])
	if len(fields) < 22 || len(fields[0]) != 1 {
		return statInfo{}, errors.New("incomplete stat fields")
	}

	ppid, err := strconv.Atoi(fields[1])
	if err != nil || ppid < 0 {
		return statInfo{}, errors.New("invalid parent PID")
	}
	threadCount, err := strconv.Atoi(fields[17])
	if err != nil || threadCount < 0 {
		return statInfo{}, errors.New("invalid thread count")
	}
	residentPages, err := strconv.ParseInt(fields[21], 10, 64)
	if err != nil {
		return statInfo{}, errors.New("invalid resident memory size")
	}
	startTicks, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil {
		return statInfo{}, errors.New("invalid process start time")
	}

	var residentBytes uint64
	if residentPages > 0 {
		residentBytes = uint64(residentPages) * uint64(os.Getpagesize())
	}
	return statInfo{
		ProcessInfo: ProcessInfo{
			PID:       pid,
			PPID:      ppid,
			Name:      data[openParen+1 : closeParen],
			State:     fields[0],
			Threads:   threadCount,
			MemoryRSS: residentBytes,
			Command:   data[openParen+1 : closeParen],
		},
		startTicks: startTicks,
	}, nil
}

func processReadError(pid int, err error) error {
	switch {
	case errors.Is(err, os.ErrNotExist):
		return fmt.Errorf("process %d not found: %w", pid, err)
	case errors.Is(err, os.ErrPermission):
		return fmt.Errorf("permission denied reading process %d: %w", pid, err)
	default:
		return fmt.Errorf("read process %d: %w", pid, err)
	}
}
