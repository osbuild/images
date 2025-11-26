package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/osbuild/images/cmd/check-host-config/cos"
)

// runningWait emulates 'systemctl is-system-running --wait'
// It blocks until the system reaches "running" or fails on other states.
// It is requried for older versions of systemd that don't support the option (EL8)
func runningWait(ctx context.Context, logger cos.Logger) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		if ctx.Err() != nil {
			return fmt.Errorf("timeout before waiting for running state: %w", ctx.Err())
		}

		out, err := cos.ExecContext(ctx, logger, "systemctl", "is-system-running")
		if err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("timeout during waiting for running state: %w", ctx.Err())
			}

			// Non-zero exit code, continue polling
			continue
		}

		state := strings.TrimSpace(string(out))
		switch state {
		case "initializing", "starting":
			select {
			case <-ctx.Done():
				return fmt.Errorf("timeout during waiting for running state: %w", ctx.Err())
			case <-ticker.C:
				continue
			}
		case "running":
			return nil
		case "":
			return fmt.Errorf("systemctl returned empty output")
		default:
			return fmt.Errorf("system is at non-running state: %q", state)
		}
	}
}

// getActivatingUnits returns a space-separated string of systemd units that are
// still in the activating state. It calls systemctl list-units to get the list.
func getActivatingUnits(ctx context.Context) string {
	out, err := cos.ExecContext(ctx, nil, "systemctl", "list-units", "--state=activating", "--plain", "--no-legend", "--no-pager")
	if err != nil {
		return ""
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var units []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// First column is the unit name (everything before the first space)
		fields := strings.Fields(line)
		if len(fields) > 0 {
			units = append(units, fields[0])
		}
	}

	return strings.Join(units, " ")
}
