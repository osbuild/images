package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/osbuild/images/cmd/check-host-config/mockos"
)

// runningWait emulates 'systemctl is-system-running --wait'
// It blocks until the system reaches "running" or fails on other states.
// It is requried for older versions of systemd that don't support the option (EL8)
func runningWait(ctx context.Context, logger mockos.Logger) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	lastLogTime := time.Now()

	for {
		var logSometimes mockos.Logger // logger that logs only once per every minute
		if time.Since(lastLogTime) > 1*time.Minute {
			logSometimes = logger
			lastLogTime = time.Now()
		}

		if ctx.Err() != nil {
			return fmt.Errorf("timeout before waiting for running state: %w", ctx.Err())
		}

		out, _, err := mockos.ExecContext(ctx, logSometimes, "systemctl", "is-system-running")
		state := strings.TrimSpace(string(out))
		if err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("%w: last known state of systemd is-system-running: %q", ctx.Err(), state)
			}

			// systemctl typically returns non-zero exit code for non-running states but on
			// older RHEL systems it returns zero exit code and outputs the state to stdout.
			continue
		}

		switch state {
		case "initializing", "starting":
			select {
			case <-ctx.Done():
				return fmt.Errorf("%w: last known state of systemd is-system-running: %q", ctx.Err(), state)
			case <-ticker.C:
				continue
			}
		case "running":
			logger.Printf("System is running\n")
			return nil
		case "degraded":
			logger.Printf("System is degraded\n")
			return fmt.Errorf("systemctl returned degraded output")
		case "":
			logger.Printf("System is in unknown state\n")
			return fmt.Errorf("systemctl returned empty output")
		default:
			return fmt.Errorf("system is at non-running state: %q", state)
		}
	}
}

// listBadUnits returns a space-separated string of systemd units that are
// still in the activating state. It calls systemctl list-units to get the list.
// This is only used in case of timeout to help with debugging.
func listBadUnits(ctx context.Context) string {
	out, _, err := mockos.ExecContext(ctx, nil, "systemctl", "list-units", "--state=activating,failed", "--plain", "--no-legend", "--no-pager")
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
