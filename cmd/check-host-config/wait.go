package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/osbuild/images/cmd/check-host-config/mockos"
)

// runningWait emulates 'systemctl is-system-running --wait'
// It blocks until the system reaches "running" or fails on other states.
// It is requried for older versions of systemd that don't support the option (EL8)
func runningWait(ctx context.Context, logger mockos.Logger) error {
	// Unset DBUS_VERBOSE so subprocesses don't inherit it (breaks systemd output)
	os.Unsetenv("DBUS_VERBOSE")

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	lastLogTime := time.Now()

	// Set up signal handling for OS termination signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(sigChan)

	for {
		var logSometimes mockos.Logger // logger that logs only once per every minute
		if time.Since(lastLogTime) > 1*time.Minute {
			logSometimes = logger
			lastLogTime = time.Now()
		}

		if ctx.Err() != nil {
			return fmt.Errorf("timeout before waiting for running state: %w", ctx.Err())
		}

		select {
		case sig := <-sigChan:
			if logger != nil {
				logger.Printf("Received termination signal: %v, exiting\n", sig)
			}
			return fmt.Errorf("terminated by signal: %v", sig)
		default:
		}

		out, _, err := mockos.ExecContext(ctx, logSometimes, "systemctl", "is-system-running")
		if err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("timeout during waiting for running state: %w", ctx.Err())
			}

			continue
		}

		state := strings.TrimSpace(string(out))
		switch state {
		case "initializing", "starting":
			select {
			case <-ctx.Done():
				return fmt.Errorf("timeout during waiting for running state: %w", ctx.Err())
			case sig := <-sigChan:
				if logger != nil {
					logger.Printf("Received termination signal: %v, exiting\n", sig)
				}
				return fmt.Errorf("terminated by signal: %v", sig)
			case <-ticker.C:
				continue
			}
		case "running":
			logger.Printf("System is running\n")
			return nil
		case "":
			logger.Printf("System is in unknown state\n")
			return fmt.Errorf("systemctl returned empty output")
		default:
			return fmt.Errorf("system is at non-running state: %q", state)
		}
	}
}

// getActivatingUnits returns a space-separated string of systemd units that are
// still in the activating state. It calls systemctl list-units to get the list.
func getActivatingUnits(ctx context.Context) string {
	out, _, err := mockos.ExecContext(ctx, nil, "systemctl", "list-units", "--state=activating", "--plain", "--no-legend", "--no-pager")
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
