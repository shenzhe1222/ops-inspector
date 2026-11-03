package process

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReaderProcessCommands(t *testing.T) {
	root := t.TempDir()
	writeProcess(t, root, 1, "init", 'S', 0, 1, 1, 1, 1024, 1)
	writeProcess(t, root, 100, "worker ) main", 'R', 1, 10, 4, 2, 4096, 2)
	writeProcess(t, root, 101, "worker thread", 'S', 100, 1, 1, 1, 1024, 1)
	writeProcess(t, root, 102, "defunct", 'Z', 1, 0, 0, 1, 0, 0)

	if err := os.WriteFile(filepath.Join(root, "100", "cmdline"), []byte("worker\x00--mode\x00fast\x00"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, thread := range []struct {
		tid   int
		name  string
		state byte
	}{
		{100, "worker ) main", 'R'},
		{103, "worker thread", 'S'},
	} {
		dir := filepath.Join(root, "100", "task", fmt.Sprint(thread.tid))
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		writeStatFile(t, filepath.Join(dir, "stat"),
			makeStat(thread.tid, thread.name, thread.state, 100, 0, 0, 1, 0, 0))
	}
	for _, fd := range []string{"0", "1", "2"} {
		path := filepath.Join(root, "100", "fd", fd)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	reader := NewReader(root)
	processes, err := reader.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(processes) != 4 || processes[0].PID != 1 || processes[1].PID != 100 {
		t.Fatalf("List() PIDs are not sorted: %#v", processes)
	}

	info, err := reader.Info(100)
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	if info.Name != "worker ) main" || info.CommandLine != "worker --mode fast" {
		t.Fatalf("Info() returned unexpected command details: %#v", info)
	}
	if info.ResidentBytes != 2*uint64(os.Getpagesize()) || info.Threads != 2 {
		t.Fatalf("Info() returned unexpected resource details: %#v", info)
	}

	threads, err := reader.Threads(100)
	if err != nil {
		t.Fatalf("Threads() error = %v", err)
	}
	if len(threads) != 2 || threads[0].TID != 100 || threads[1].TID != 103 {
		t.Fatalf("Threads() returned unexpected results: %#v", threads)
	}

	fdCount, err := reader.FileDescriptorCount(100)
	if err != nil {
		t.Fatalf("FileDescriptorCount() error = %v", err)
	}
	if fdCount != 3 {
		t.Fatalf("FileDescriptorCount() = %d, want 3", fdCount)
	}

	tree, err := reader.Tree(100)
	if err != nil {
		t.Fatalf("Tree() error = %v", err)
	}
	if len(tree.Children) != 1 || tree.Children[0].Process.PID != 101 {
		t.Fatalf("Tree() returned unexpected children: %#v", tree.Children)
	}

	stats, err := reader.Stats()
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if stats.Total != 4 || stats.Running != 1 || stats.Sleeping != 2 || stats.Zombie != 1 {
		t.Fatalf("Stats() returned unexpected counts: %#v", stats)
	}
}

func TestCalculateUsage(t *testing.T) {
	first := []Process{{PID: 1, UserTicks: 7, SystemTicks: 3}}
	second := []Process{
		{PID: 1, UserTicks: 12, SystemTicks: 3, ResidentBytes: 1000},
		{PID: 2, UserTicks: 5, ResidentBytes: 500},
	}
	usage := calculateUsage(first, second, 100, 4, 10000)
	if len(usage) != 2 {
		t.Fatalf("calculateUsage() returned %d entries, want 2", len(usage))
	}
	if usage[0].CPUPercent != 20 || usage[0].MemoryPercent != 10 {
		t.Fatalf("calculateUsage() returned unexpected usage: %#v", usage[0])
	}
}

func TestReadCPUCountersExcludesGuestCounters(t *testing.T) {
	root := t.TempDir()
	data := "cpu 1 2 3 4 5 6 7 8 9 10\ncpu0 1 2 3 4 5 6 7 8 9 10\ncpu1 1 2 3 4 5 6 7 8 9 10\n"
	if err := os.WriteFile(filepath.Join(root, "stat"), []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	total, cpuCount, err := NewReader(root).readCPUCounters()
	if err != nil {
		t.Fatalf("readCPUCounters() error = %v", err)
	}
	if total != 36 || cpuCount != 2 {
		t.Fatalf("readCPUCounters() = (%d, %d), want (36, 2)", total, cpuCount)
	}
}

func writeProcess(t *testing.T, root string, pid int, name string, state byte, ppid, userTicks, systemTicks, threads int, virtualBytes uint64, residentPages int64) {
	t.Helper()
	dir := filepath.Join(root, fmt.Sprint(pid))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	writeStatFile(t, filepath.Join(dir, "stat"),
		makeStat(pid, name, state, ppid, userTicks, systemTicks, threads, virtualBytes, residentPages))
}

func writeStatFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

func makeStat(pid int, name string, state byte, ppid, userTicks, systemTicks, threads int, virtualBytes uint64, residentPages int64) string {
	fields := make([]string, 22)
	for i := range fields {
		fields[i] = "0"
	}
	fields[0] = string(state)
	fields[1] = fmt.Sprint(ppid)
	fields[11] = fmt.Sprint(userTicks)
	fields[12] = fmt.Sprint(systemTicks)
	fields[17] = fmt.Sprint(threads)
	fields[20] = fmt.Sprint(virtualBytes)
	fields[21] = fmt.Sprint(residentPages)
	return fmt.Sprintf("%d (%s) %s", pid, name, strings.Join(fields, " "))
}
