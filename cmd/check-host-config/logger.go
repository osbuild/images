package main

import (
	"fmt"
	"io"
	"log"
)

func NewLogger(writer io.Writer, prefix string, quiet bool) *log.Logger {
	for len(prefix) < MaxShortCheckName {
		prefix += " "
	}
	logger := log.New(writer, fmt.Sprintf("[%s] ", prefix), 0)

	if quiet {
		logger.SetOutput(io.Discard)
	}

	return logger
}
