package mockos

import (
	"context"
	"os"
)

// ReadFileContext reads the contents of a file.
// If logger is provided and not nil, it will log debug information about the read operation.
func ReadFileContext(ctx context.Context, logger Logger, filename string) ([]byte, error) {
	if f := ReadFileFunc(ctx); f != nil {
		return f(filename)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		if logger != nil {
			logger.Printf("Failed to read file: %s (error: %v)\n", filename, err)
		}
		return nil, err
	}

	if logger != nil {
		logger.Printf("Read file: %s (%d bytes)\n", filename, len(data))
	}

	return data, nil
}
