package process

import "fmt"

func (r *Reader) Stats() (Stats, error) {
	processes, err := r.readStats()
	if err != nil {
		return Stats{}, err
	}

	stats := Stats{Total: len(processes)}
	for _, proc := range processes {
		switch proc.State {
		case "R":
			stats.Running++
		case "S":
			stats.Sleeping++
		case "D":
			stats.Uninterruptible++
		case "Z":
			stats.Zombie++
		case "T", "t":
			stats.Stopped++
		case "I":
			stats.Idle++
		default:
			stats.Other++
		}
	}
	return stats, nil
}

func StateName(state string) string {
	switch state {
	case "R":
		return "Running"
	case "S":
		return "Sleeping"
	case "D":
		return "Uninterruptible"
	case "Z":
		return "Zombie"
	case "T", "t":
		return "Stopped"
	case "I":
		return "Idle"
	default:
		return fmt.Sprintf("Unknown (%s)", state)
	}
}
