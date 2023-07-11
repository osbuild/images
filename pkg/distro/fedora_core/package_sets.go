package fedora_core

// This file defines package sets that are used by more than one image type.

import (
	"github.com/osbuild/images/pkg/rpmmd"
)

func corePackageSet(t *imageType) rpmmd.PackageSet {
	return rpmmd.PackageSet{
		Include: []string{
			"@core",
		},
	}
}
