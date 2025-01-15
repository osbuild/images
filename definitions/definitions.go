package definitions

import "embed"

//go:embed centos
//go:embed rhel
var Data embed.FS
