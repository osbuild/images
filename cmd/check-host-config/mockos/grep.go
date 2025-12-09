package mockos

import (
	"bufio"
	"context"
	"os"
	"strings"
)

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
