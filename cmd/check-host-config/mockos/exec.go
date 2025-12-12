package mockos

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
)

// ExecContext calls os/exec.Command and returns stdout and stderr separately,
// except when overridden in the context.
// If logger is provided and not nil, it will log debug information about the command execution.
// If stderr contains output, it will be logged as a string.
func ExecContext(ctx context.Context, logger Logger, name string, arg ...string) ([]byte, []byte, error) {
	if f := ExecFunc(ctx); f != nil {
		return f(name, arg...)
	}

	if logger != nil {
		cmdStr := name
		if len(arg) > 0 {
			cmdStr += " " + strings.Join(arg, " ")
		}
		logger.Printf("Executing: %s\n", cmdStr)
	}

	cmd := exec.Command(name, arg...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err := cmd.Run()
	stdout := stdoutBuf.Bytes()
	stderr := stderrBuf.Bytes()

	if logger != nil {
		if err != nil {
			logger.Printf("Command failed: %s with exit code: %v\n", name, err)
		}

		if len(stdout) > 0 {
			logger.Printf("Command stdout: %s\n", strings.TrimSpace(string(stdout)))
		}

		if len(stderr) > 0 {
			logger.Printf("Command stderr: %s\n", strings.TrimSpace(string(stderr)))
		}
	}

	return stdout, stderr, err
}
