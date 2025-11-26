package cos

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"strings"
)

// Logger is an interface for logging operations.
// It matches check.Logger to allow sharing the same interface.
type Logger interface {
	Print(v ...any)
	Printf(format string, v ...any)
	Println(v ...any)
}

// ExecContext calls os/exec.Command and returns combined output,
// except when overridden in the context.
// If logger is provided and not nil, it will log debug information about the command execution.
func ExecContext(ctx context.Context, logger Logger, name string, arg ...string) ([]byte, error) {
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
	out, err := cmd.CombinedOutput()

	if logger != nil {
		if err != nil {
			logger.Printf("Command failed: %s (exit code: %v)\n", name, err)
		} else {
			logger.Printf("Command succeeded: %s\n", name)
		}
	}

	return out, err
}

// ExistsContext checks if a file or directory exists.
// If logger is provided and not nil, it will log debug information about the check.
func ExistsContext(ctx context.Context, logger Logger, name string) bool {
	if f := ExistsFunc(ctx); f != nil {
		return f(name)
	}

	if logger != nil {
		logger.Printf("Checking if file exists: %s\n", name)
	}

	_, err := os.Stat(name)
	exists := !os.IsNotExist(err)

	if logger != nil {
		if exists {
			logger.Printf("File exists: %s\n", name)
		} else {
			logger.Printf("File does not exist: %s\n", name)
		}
	}

	return exists
}

// GrepContext searches for a pattern in a file and returns true if found.
// This is a native Go implementation that replaces shelling out to grep.
// If logger is provided and not nil, it will log debug information about the search.
func GrepContext(ctx context.Context, logger Logger, pattern, filename string) (bool, error) {
	if f := GrepFunc(ctx); f != nil {
		return f(pattern, filename)
	}

	if logger != nil {
		logger.Printf("Searching for pattern '%s' in file: %s\n", pattern, filename)
	}

	file, err := os.Open(filename)
	if err != nil {
		if logger != nil {
			logger.Printf("Failed to open file for grep: %s (error: %v)\n", filename, err)
		}
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}

		lineNum++
		line := scanner.Text()
		if strings.Contains(line, pattern) {
			if logger != nil {
				logger.Printf("Pattern found at line %d in %s\n", lineNum, filename)
			}
			return true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		if logger != nil {
			logger.Printf("Error scanning file: %s (error: %v)\n", filename, err)
		}
		return false, err
	}

	if logger != nil {
		logger.Printf("Pattern not found in file: %s (searched %d lines)\n", filename, lineNum)
	}

	return false, nil
}
