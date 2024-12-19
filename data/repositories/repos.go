package repos

import (
	"embed"
	"io/fs"

	"github.com/osbuild/images/pkg/reporegistry"
)

//go:embed *.json
var FS embed.FS
