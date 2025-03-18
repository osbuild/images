package rhel9

import (
	"github.com/osbuild/images/pkg/distro/packagesets"
	"github.com/osbuild/images/pkg/distro/rhel"
	"github.com/osbuild/images/pkg/rpmmd"
)

func packageSetLoader(t *rhel.ImageType) (rpmmd.PackageSet, error) {
	return packagesets.Load(t, "", nil)
}
