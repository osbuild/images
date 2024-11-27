package main

import (
	"os"
	"strings"

	"github.com/osbuild/images/pkg/reporegistry"
)

// XXX: copied from "composer", should be exported there so
// that we keep this in sync
// XXX2: means we need to depend on osbuild-composer-common or something
var repositoryConfigs = []string{
	"/etc/osbuild-composer",
	"/usr/share/osbuild-composer",
}

// XXX: move this new env into pkg/reporegistry?
func newRepoRegistry() (*reporegistry.RepoRegistry, error) {
	// useful for development/debugging, e.g. run:
	// go build && IMAGE_BUILDER_EXTRA_REPOS_PATH=../../test/data ./image-builder
	if extraReposPath := os.Getenv("IMAGE_BUILDER_EXTRA_REPOS_PATH"); extraReposPath != "" {
		repositoryConfigs = append(repositoryConfigs, strings.Split(extraReposPath, ":")...)
	}

	return reporegistry.New(repositoryConfigs)
}
