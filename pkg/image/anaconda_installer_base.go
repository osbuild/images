package image

import (
	"github.com/osbuild/images/pkg/customizations/kickstart"
	"github.com/osbuild/images/pkg/manifest"
)

// common struct that all anaconda installers share
type AnacondaInstallerBase struct {
	InstallerCustomizations manifest.InstallerCustomizations
	RootfsCompression       string
	Kickstart               *kickstart.Options
}
