package main

import (
	"io"

	"github.com/osbuild/images/pkg/distrofactory"
)

func getFilteredImages(filterExprs []string) ([]FilterResult, error) {
	repos, err := newRepoRegistry()
	if err != nil {
		return nil, err
	}
	filters, err := NewFilters(filterExprs)
	if err != nil {
		return nil, err
	}
	fac := distrofactory.NewDefault()
	return FilterDistros(fac, repos.ListDistros(), filters)
}

func listImages(out io.Writer, format string, filterExprs []string) error {
	filteredResult, err := getFilteredImages(filterExprs)
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
