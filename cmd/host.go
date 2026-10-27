package cmd

import (
	"fmt"

	"ops-inspector/internal/host"
)

func runHost(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing host command: use info or check")
	}

	switch args[0] {
	case "info":
		if len(args) != 1 {
			return fmt.Errorf("usage: ops-inspector host info")
		}

		var info host.HostInfo

		if err := info.CheckInfo(); err != nil {
			return err
		}

		info.Print()
		return nil

	case "check":
		if len(args) != 1 {
			return fmt.Errorf("usage: ops-inspector host check")
		}

		disk, err := host.CheckDisk("/")
		if err != nil {
			return err
		}

		disk.Print()
		return nil

	default:
		return fmt.Errorf("unknown host command: %s", args[0])
	}
}