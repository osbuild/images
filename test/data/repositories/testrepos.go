package testrepos

import (
	"embed"
	"io/fs"

	"github.com/osbuild/images/pkg/reporegistry"
)

//go:embed *.json
var FS embed.FS

func New() (*reporegistry.RepoRegistry, error) {
	repositories, err := reporegistry.LoadAllRepositoriesFromFS([]fs.FS{FS})
	if err != nil {
		return nil, err
	}
	return reporegistry.NewFromDistrosRepoConfigs(repositories), nil
}
