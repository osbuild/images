package main

import (
	"io"
)

func listImages(out io.Writer, format string, filterExprs []string) error {
	imageFilter, err := newImageFilterDefault()
	if err != nil {
		return err
	}

	filteredResult, err := imageFilter.Filter(filterExprs...)
	if err != nil {
		return err
	}

	fmter, err := NewFilteredResultFormatter(format)
	if err != nil {
		return err
	}
	fmter.Output(out, filteredResult)

	return nil
}
