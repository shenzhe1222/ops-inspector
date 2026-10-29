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

	case "analyze":
		if len(args) < 2 {
			return fmt.Errorf("usage: ops-inspector disk analyze <path> [--top N]")
		}

		path := args[1]
		top := 10

		if len(args) > 2 {
			if len(args) != 4 || args[2] != "--top" {
				return fmt.Errorf("usage: ops-inspector disk analyze <path> [--top N]")
			}

			if _, err := fmt.Sscanf(args[3], "%d", &top); err != nil || top <= 0 {
				return fmt.Errorf("invalid --top value: %s", args[3])
			}
		}

		results, err := disk.Analyze(path, top)
		if err != nil {
			return err
		}

		disk.PrintAnalyze(results)
		return nil

	default:
		return fmt.Errorf("unknown disk command: %s", args[0])
	}
}