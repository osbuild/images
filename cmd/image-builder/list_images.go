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

// XXX: move to pkg/reporegistry
func newRepoRegistry() (*reporegistry.RepoRegistry, error) {
	// useful for development/debugging, e.g. run:
	// go build && IMAGE_BUILDER_EXTRA_REPOS_PATH=../../test/data ./image-builder
	if extraReposPath := os.Getenv("IMAGE_BUILDER_EXTRA_REPOS_PATH"); extraReposPath != "" {
		repositoryConfigs = append(repositoryConfigs, strings.Split(extraReposPath, ":")...)
	}

	return reporegistry.New(repositoryConfigs)
}

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
