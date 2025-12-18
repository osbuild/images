// Package repokeys contains the GPG keys for the repositories. It can be used
// in data/repositories/*.json files via the "mem://" URI.
package repokeys

import (
	"embed"
)

//go:embed *
var GPGFS embed.FS
