package process

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReaderListInfoAndStats(t *testing.T) {
	root := t.TempDir()
	writeSystemFiles(t, root)
	writeProcess(t, root, 1, "init", "S", 0, 1, 120, 1)
	writeProcess(t, root, 100, "worker ) main", "R", 1, 4, 180, 25)
	writeProcess(t, root, 101, "worker child", "S", 100, 2, 200, 15)
	writeProcess(t, root, 102, "defunct", "Z", 1, 1, 250, 0)

	writeCommand(t, root, 1, "/sbin/init")
	writeCommand(t, root, 100, "/usr/bin/worker\x00--mode\x00fast")
	writeCommand(t, root, 101, "/usr/bin/worker-child")
	writeCommand(t, root, 102, "")

	reader := NewReader(root)
	processes, err := reader.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(processes) != 4 || processes[0].PID != 1 || processes[1].PID != 100 {
		t.Fatalf("List() did not return sorted processes: %#v", processes)
	}
	if processes[1].Command != "/usr/bin/worker --mode fast" {
		t.Fatalf("List() command = %q", processes[1].Command)
	}

	info, err := reader.Info(100)
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	if info.Name != "worker ) main" || info.User == "" || info.State != "R" {
		t.Fatalf("Info() returned unexpected process details: %#v", info)
	}
	if info.PPID != 1 || info.Threads != 4 || info.MemoryRSS != 25*uint64(os.Getpagesize()) {
		t.Fatalf("Info() returned unexpected process resources: %#v", info)
	}
	if info.MemoryPct <= 0 || info.StartTime.IsZero() {
		t.Fatalf("Info() returned invalid memory percentage or start time: %#v", info)
	}
	if info.Command != "/usr/bin/worker --mode fast" {
		t.Fatalf("Info() command = %q", info.Command)
	}

	stats, err := reader.Stats()
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if stats.Total != 4 || stats.Running != 1 || stats.Sleeping != 2 || stats.Zombie != 1 {
		t.Fatalf("Stats() returned unexpected counts: %#v", stats)
	}
}

func TestInfoReportsMissingProcess(t *testing.T) {
	root := t.TempDir()
	_, err := NewReader(root).Info(999)
	if err == nil {
		t.Fatal("Info() error = nil, want missing-process error")
	}
	if !errors.Is(err, os.ErrNotExist) || !strings.Contains(err.Error(), "process 999 not found") {
		t.Fatalf("Info() error = %v, want a not-found error", err)
	}
}

func TestProcessReadErrorPreservesErrorKind(t *testing.T) {
	tests := []struct {
		name        string
		cause       error
		messagePart string
	}{
		{name: "permission", cause: os.ErrPermission, messagePart: "permission denied"},
		{name: "read failure", cause: os.ErrClosed, messagePart: "read process 123"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := processReadError(123, test.cause)
			if !errors.Is(err, test.cause) || !strings.Contains(err.Error(), test.messagePart) {
				t.Fatalf("processReadError() = %v", err)
			}
		})
	}
}

func TestParseStatRejectsMalformedData(t *testing.T) {
	for _, input := range []string{
		"not a proc stat",
		"123 (worker) S 1",
		"0 (worker) S 1 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0",
	} {
		if _, err := parseStat(input); err == nil {
			t.Errorf("parseStat(%q) error = nil, want malformed-data error", input)
		}
	}
}

func writeSystemFiles(t *testing.T, root string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "meminfo"), []byte("MemTotal: 100000 kB\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "stat"), []byte("cpu 10 0 10 80 0 0 0 0\nbtime 1700000000\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeProcess(t *testing.T, root string, pid int, name, state string, ppid, threads int, startTicks uint64, residentPages int64) {
	t.Helper()
	dir := filepath.Join(root, fmt.Sprint(pid))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}

	fields := make([]string, 22)
	for i := range fields {
		fields[i] = "0"
	}
	fields[0] = state
	fields[1] = fmt.Sprint(ppid)
	fields[17] = fmt.Sprint(threads)
	fields[19] = fmt.Sprint(startTicks)
	fields[20] = "4096"
	fields[21] = fmt.Sprint(residentPages)
	stat := fmt.Sprintf("%d (%s) %s", pid, name, strings.Join(fields, " "))
	if err := os.WriteFile(filepath.Join(dir, "stat"), []byte(stat), 0o600); err != nil {
		t.Fatal(err)
	}
	status := fmt.Sprintf("Name:\t%s\nUid:\t0\t0\t0\t0\n", name)
	if err := os.WriteFile(filepath.Join(dir, "status"), []byte(status), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeCommand(t *testing.T, root string, pid int, command string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, fmt.Sprint(pid), "cmdline"), []byte(command), 0o600); err != nil {
		t.Fatal(err)
	}
}
