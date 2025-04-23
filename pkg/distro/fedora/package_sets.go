package fedora

import (
	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/defs"
	"github.com/osbuild/images/pkg/rpmmd"
)

func packageSetLoader(t *imageType) (map[string]rpmmd.PackageSet, error) {
	return defs.PackageSets(t, VersionReplacements())
}

func imageConfig(d distribution, imageType string) *distro.ImageConfig {
	// arch is currently not used in fedora
	arch := ""
	return common.Must(defs.ImageConfig(d.name, arch, imageType, VersionReplacements()))
}
