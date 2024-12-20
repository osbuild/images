package specs

import "embed"

//go:embed centos-10
//go:embed rhel-10.0
var Data embed.FS
