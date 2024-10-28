package main

import (
	"io"
	"os"
	"strings"

	"github.com/osbuild/images/pkg/distrofactory"
	"github.com/osbuild/images/pkg/reporegistry"
)

// XXX: copied from "composer", should be exported there so
// that we keep this in sync
// XXX2: means we need to depend on osbuild-composer-common or something
var repositoryConfigs = []string{
	"/etc/osbuild-composer",
	"/usr/share/osbuild-composer",
}

func listImages(out io.Writer, format string, filterExprs []string) error {
	// useful for development/debugging, e.g. run:
	// go build && IMAGE_BUILDER_EXTRA_REPOS_PATH=../../test/data ./image-builder
	if extraReposPath := os.Getenv("IMAGE_BUILDER_EXTRA_REPOS_PATH"); extraReposPath != "" {
		repositoryConfigs = append(repositoryConfigs, strings.Split(extraReposPath, ":")...)
	}

	repos, err := reporegistry.New(repositoryConfigs)
	if err != nil {
		return err
	}
	filters, err := NewFilters(filterExprs)
	if err != nil {
		return err
	}
	fac := distrofactory.NewDefault()
	filteredResult, err := FilterDistros(fac, repos.ListDistros(), filters)
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
