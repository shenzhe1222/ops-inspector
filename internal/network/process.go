package network

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func (r *Reader) associateProcesses(sockets []Socket) error {
	inodes := make(map[uint64]bool)
	for _, socket := range sockets {
		if socket.Inode != 0 {
			inodes[socket.Inode] = true
		}
	}
	if len(inodes) == 0 {
		return nil
	}

	users, err := readUsers()
	if err != nil {
		return err
	}
	processesByInode := make(map[uint64][]ProcessInfo, len(inodes))
	seen := make(map[uint64]map[int]bool, len(inodes))
	for inode := range inodes {
		seen[inode] = make(map[int]bool)
	}

	entries, err := os.ReadDir(r.procRoot)
	if err != nil {
		return fmt.Errorf("read process directory %s: %w", r.procRoot, err)
	}
	inaccessible := 0
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 || !entry.IsDir() {
			continue
		}
		fdDir := filepath.Join(r.procRoot, entry.Name(), "fd")
		fds, err := os.ReadDir(fdDir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if errors.Is(err, os.ErrPermission) {
				inaccessible++
				continue
			}
			return fmt.Errorf("read file descriptors for process %d: %w", pid, err)
		}

		matched := make(map[uint64]bool)
		for _, fd := range fds {
			target, err := os.Readlink(filepath.Join(fdDir, fd.Name()))
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
				if errors.Is(err, os.ErrPermission) {
					inaccessible++
					continue
				}
				return fmt.Errorf("read file descriptor %s for process %d: %w", fd.Name(), pid, err)
			}
			inode, ok := socketInode(target)
			if ok && inodes[inode] {
				matched[inode] = true
			}
		}
		for inode := range matched {
			if seen[inode][pid] {
				continue
			}
			process, warning, alive, err := r.readProcessInfo(pid, users)
			if err != nil {
				return err
			}
			if !alive {
				continue
			}
			processesByInode[inode] = append(processesByInode[inode], process)
			seen[inode][pid] = true
			if warning {
				inaccessible++
			}
		}
	}

	warning := ""
	if inaccessible > 0 {
		warning = fmt.Sprintf(
			"process association may be incomplete: permission denied while inspecting %d /proc entries",
			inaccessible,
		)
	}
	for i := range sockets {
		sockets[i].Processes = processesByInode[sockets[i].Inode]
		sort.Slice(sockets[i].Processes, func(a, b int) bool {
			return sockets[i].Processes[a].PID < sockets[i].Processes[b].PID
		})
		sockets[i].ProcessWarning = warning
	}
	return nil
}

func (r *Reader) readProcessInfo(pid int, users map[string]string) (ProcessInfo, bool, bool, error) {
	dir := filepath.Join(r.procRoot, strconv.Itoa(pid))
	read := func(name string) ([]byte, bool, error) {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		if errors.Is(err, os.ErrPermission) {
			return nil, true, nil
		}
		if err != nil {
			return nil, false, fmt.Errorf("read %s for process %d: %w", name, pid, err)
		}
		return data, false, nil
	}

	comm, commDenied, err := read("comm")
	if err != nil {
		return ProcessInfo{}, false, false, err
	}
	status, statusDenied, err := read("status")
	if err != nil {
		return ProcessInfo{}, false, false, err
	}
	command, commandDenied, err := read("cmdline")
	if err != nil {
		return ProcessInfo{}, false, false, err
	}
	if (!commDenied && comm == nil) || (!statusDenied && status == nil) ||
		(!commandDenied && command == nil) {
		return ProcessInfo{}, false, false, nil
	}

	info := ProcessInfo{PID: pid}
	if commDenied || len(comm) == 0 {
		info.Name = "[unavailable]"
	} else {
		info.Name = strings.TrimSpace(string(comm))
	}
	if statusDenied {
		info.User = "[unavailable]"
	} else if len(status) > 0 {
		uid, err := processUID(string(status))
		if err != nil {
			return ProcessInfo{}, false, false, fmt.Errorf("parse UID for process %d: %w", pid, err)
		}
		info.User = users[uid]
		if info.User == "" {
			info.User = uid
		}
	}
	if commandDenied {
		info.Command = "[unavailable]"
	} else if len(command) == 0 {
		info.Command = info.Name
	} else {
		info.Command = strings.TrimSpace(strings.ReplaceAll(string(command), "\x00", " "))
	}
	return info, commDenied || statusDenied || commandDenied, true, nil
}

func socketInode(target string) (uint64, bool) {
	if !strings.HasPrefix(target, "socket:[") || !strings.HasSuffix(target, "]") {
		return 0, false
	}
	inode, err := strconv.ParseUint(strings.TrimSuffix(strings.TrimPrefix(target, "socket:["), "]"), 10, 64)
	return inode, err == nil
}

func processUID(status string) (string, error) {
	for _, line := range strings.Split(status, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "Uid:" {
			return fields[1], nil
		}
	}
	return "", errors.New("Uid field is missing")
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
