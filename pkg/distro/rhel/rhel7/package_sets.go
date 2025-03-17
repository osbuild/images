package rhel7

import (
	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/distro/packagesets"
	"github.com/osbuild/images/pkg/distro/rhel"
	"github.com/osbuild/images/pkg/rpmmd"
)

func packageSetLoader(t *rhel.ImageType) rpmmd.PackageSet {
	return common.Must(packagesets.Load(t, "", nil))
}
