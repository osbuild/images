package rhel10

// This file defines package sets that are used by more than one image type.

import (
	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/distro/rhel"
	"github.com/osbuild/images/pkg/rpmmd"
)

// BUILD PACKAGE SETS

// distro-wide build package set
func distroBuildPackageSet(t *rhel.ImageType) rpmmd.PackageSet {
	ps := rpmmd.PackageSet{
		Include: []string{
			"dnf",
			"dosfstools",
			"e2fsprogs",
			"glibc",
			"lorax-templates-generic",
			"lorax-templates-rhel",
			"lvm2",
			"policycoreutils",
			"python3-iniparse",
			"qemu-img",
			"selinux-policy-targeted",
			"systemd",
			"tar",
			"xfsprogs",
			"xz",
		},
	}

	switch t.Arch().Name() {

	case arch.ARCH_X86_64.String():
		ps = ps.Append(x8664BuildPackageSet(t))

	case arch.ARCH_PPC64LE.String():
		ps = ps.Append(ppc64leBuildPackageSet(t))
	}

	return ps
}

// x86_64 build package set
func x8664BuildPackageSet(t *rhel.ImageType) rpmmd.PackageSet {
	return rpmmd.PackageSet{
		Include: []string{
			"grub2-pc",
		},
	}
}

// ppc64le build package set
func ppc64leBuildPackageSet(t *rhel.ImageType) rpmmd.PackageSet {
	return rpmmd.PackageSet{
		Include: []string{
			"grub2-ppc64le",
			"grub2-ppc64le-modules",
		},
	}
}

// OS package sets

// packages that are only in some (sub)-distributions
func distroSpecificPackageSet(t *rhel.ImageType) rpmmd.PackageSet {
	if t.IsRHEL() {
		return rpmmd.PackageSet{
			Include: []string{
				"insights-client",
			},
		}
	}
	return rpmmd.PackageSet{}
}
