package fedora

import (
	"github.com/osbuild/images/pkg/distro/defs"
	"github.com/osbuild/images/pkg/rpmmd"
)

func packageSetLoader(t *imageType) (map[string]rpmmd.PackageSet, error) {
	return defs.PackageSets(t, VersionReplacements())
}
