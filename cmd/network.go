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
		fmt.Printf("Remote Address: %s\n", socket.RemoteAddress)
		fmt.Printf("Remote Port:    %d\n", socket.RemotePort)
		fmt.Printf("State:          %s\n", socket.State)
		fmt.Printf("Address Type:   %s\n", socket.AddressType)
		fmt.Printf("Socket Inode:   %d\n", socket.Inode)
		fmt.Println()
		fmt.Println("Process:")
		if len(socket.Processes) == 0 {
			if socket.ProcessWarning != "" {
				fmt.Println("  Process association may be incomplete")
			} else {
				fmt.Println("  No associated process found")
			}
		}
		for i, proc := range socket.Processes {
			if i > 0 {
				fmt.Println()
			}
			fmt.Printf("PID:            %d\n", proc.PID)
			fmt.Printf("Name:           %s\n", proc.Name)
			fmt.Printf("User:           %s\n", proc.User)
			fmt.Printf("Command:        %s\n", proc.Command)
		}
		if socket.ProcessWarning != "" {
			fmt.Printf("Warning:        %s\n", socket.ProcessWarning)
		}
	}
}
