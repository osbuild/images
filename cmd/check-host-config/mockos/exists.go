package mockos

import (
	"context"
	"os"
)

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
