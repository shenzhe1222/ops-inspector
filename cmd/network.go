package cmd

import (
	"fmt"
	"strconv"

	"ops-inspector/internal/network"
)

func runNetwork(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ops-inspector -n listen|port <port>")
	}

	reader := network.NewReader("/proc/net")
	switch args[0] {
	case "listen":
		if len(args) != 1 {
			return fmt.Errorf("usage: ops-inspector -n listen")
		}
		sockets, err := reader.Listen()
		if err != nil {
			return err
		}
		printListeners(sockets)
	case "port":
		if len(args) != 2 {
			return fmt.Errorf("usage: ops-inspector -n port <port>")
		}
		port, err := strconv.Atoi(args[1])
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("invalid port %q: must be an integer between 1 and 65535", args[1])
		}
		sockets, err := reader.Port(uint16(port))
		if err != nil {
			return err
		}
		if len(sockets) == 0 {
			return fmt.Errorf("no TCP listener found on port %d", port)
		}
		printPortInformation(sockets)
	default:
		return fmt.Errorf("unknown network command %q; available commands: listen, port", args[0])
	}
	return nil
}

func printListeners(sockets []network.Socket) {
	fmt.Printf("%-18s %-8s %-10s %s\n", "LOCAL ADDRESS", "PORT", "PROTOCOL", "STATE")
	for _, socket := range sockets {
		fmt.Printf("%-18s %-8d %-10s %s\n",
			socket.LocalAddress, socket.LocalPort, socket.Protocol, socket.State)
	}
}

func printPortInformation(sockets []network.Socket) {
	for i, socket := range sockets {
		if i > 0 {
			fmt.Println()
		}
		fmt.Println("Port Information")
		fmt.Println()
		fmt.Printf("Protocol:       %s\n", socket.Protocol)
		fmt.Printf("Local Address:  %s\n", socket.LocalAddress)
		fmt.Printf("Local Port:     %d\n", socket.LocalPort)
		fmt.Printf("State:          %s\n", socket.State)
	}
}
