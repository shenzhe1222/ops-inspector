package network

import (
	"fmt"
	"sort"
)

func (r *Reader) Listen() ([]Socket, error) {
	sockets, err := r.readListeners()
	if err != nil {
		return nil, err
	}
	return sockets, nil
}

func (r *Reader) Port(port uint16) ([]Socket, error) {
	if port == 0 {
		return nil, fmt.Errorf("port must be between 1 and 65535")
	}
	sockets, err := r.readListeners()
	if err != nil {
		return nil, err
	}

	matches := make([]Socket, 0)
	for _, socket := range sockets {
		if socket.LocalPort == port {
			matches = append(matches, socket)
		}
	}
	if err := r.associateProcesses(matches); err != nil {
		return nil, err
	}
	return matches, nil
}

func (r *Reader) readListeners() ([]Socket, error) {
	ipv4, err := r.readTCPFile("tcp")
	if err != nil {
		return nil, err
	}
	ipv6, err := r.readTCPFile("tcp6")
	if err != nil {
		return nil, err
	}

	listeners := make([]Socket, 0, len(ipv4)+len(ipv6))
	for _, socket := range append(ipv4, ipv6...) {
		if socket.State == "LISTEN" {
			listeners = append(listeners, socket)
		}
	}
	sort.Slice(listeners, func(i, j int) bool {
		if listeners[i].LocalPort != listeners[j].LocalPort {
			return listeners[i].LocalPort < listeners[j].LocalPort
		}
		return listeners[i].LocalAddress < listeners[j].LocalAddress
	})
	return listeners, nil
}
