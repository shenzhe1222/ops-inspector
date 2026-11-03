package process

import (
	"errors"
)

func (r *Reader) Info(pid int) (ProcessInfo, error) {
	if pid <= 0 {
		return ProcessInfo{}, errors.New("PID must be a positive integer")
	}
	proc, err := r.readStat(pid)
	if err != nil {
		return ProcessInfo{}, err
	}
	totalMemory, bootTime, err := r.readSystemInfo()
	if err != nil {
		return ProcessInfo{}, err
	}
	users, err := readUsers()
	if err != nil {
		return ProcessInfo{}, err
	}
	info, err := r.enrich(proc, totalMemory, bootTime, users)
	if err != nil {
		return ProcessInfo{}, err
	}
	return info, nil
}
