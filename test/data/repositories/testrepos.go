package testrepos

import (
	"embed"
	"io/fs"

	"github.com/osbuild/images/pkg/reporegistry"
)

//go:embed *.json
var FS embed.FS

func New() (*reporegistry.RepoRegistry, error) {
	return reporegistry.New(nil, []fs.FS{FS})
}
