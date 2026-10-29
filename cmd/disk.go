package cmd

import (
	"fmt"
	"ops-inspector/internal/disk"
)

func runDisk(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing disk command: use usage or analyze")
	}

	switch args[0] {
	case "usage":
		if len(args) != 1 {
			return fmt.Errorf("usage: ops-inspector disk usage")
		}

		results, err := disk.ListUsage()
		if err != nil {
			return err
		}

		disk.PrintUsage(results)
		return nil

	default:
		return fmt.Errorf("unknown disk command: %s", args[0])
	}
}