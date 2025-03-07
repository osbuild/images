package rhel10

import (
	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/rhel"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/rpmmd"
)

func mkWSLImgType() *rhel.ImageType {
	it := rhel.NewImageType(
		"wsl",
		"disk.tar.gz",
		"application/x-tar",
		map[string]rhel.PackageSetFunc{
			rhel.OSPkgsKey: wslPackageSet,
		},
		rhel.TarImage,
		[]string{"build"},
		[]string{"os", "archive"},
		[]string{"archive"},
	)

	it.DefaultImageConfig = &distro.ImageConfig{
		CloudInit: []*osbuild.CloudInitStageOptions{
			{
				Filename: "99_wsl.cfg",
				Config: osbuild.CloudInitConfigFile{
					DatasourceList: []string{
						"WSL",
						"None",
					},
					Network: &osbuild.CloudInitConfigNetwork{
						Config: "disabled",
					},
				},
			},
		},
		NoSElinux: common.ToPtr(true),
		WSLConfig: &osbuild.WSLConfStageOptions{
			Boot: osbuild.WSLConfBootOptions{
				Systemd: true,
			},
		},
	}

	return it
}

func ubiCommonPackageSet(t *rhel.ImageType) rpmmd.PackageSet {
	ps := rpmmd.PackageSet{
		Include: []string{
			"alternatives",
			"audit-libs",
			"basesystem",
			"bash",
			"ca-certificates",
			"coreutils-single",
			"crypto-policies-scripts",
			"curl-minimal",
			"dejavu-sans-fonts",
			"dnf",
			"filesystem",
			"findutils",
			"gdb-gdbserver",
			// Differs from official UBI, as we don't include CRB repos
			// "gdbm",
			"glibc-minimal-langpack",
			"gmp",
			"gnupg2",
			"gobject-introspection",
			"hostname",
			"langpacks-en",
			"libcurl-minimal",
			"openssl",
			"pam",
			"passwd",
			"procps-ng",
			"python3",
			"python3-inotify",
			"redhat-release",
			"rootfiles",
			"rpm",
			"sed",
			"setup",
			"shadow-utils",
			"subscription-manager",
			"systemd",
			"tar",
			"tpm2-tss",
			"tzdata",
			"util-linux",
			"vim-minimal",
			"yum",
		},
		Exclude: []string{
			"gawk-all-langpacks",
			"glibc-gconv-extra",
			"glibc-langpack-en",
			"openssl-pkcs11",
			"python-unversioned-command",
			"redhat-release-eula",
			"rpm-plugin-systemd-inhibit",
		},
	}

	return ps
}

func wslPackageSet(t *rhel.ImageType) rpmmd.PackageSet {
	pkgset := ubiCommonPackageSet(t)
	pkgset = pkgset.Append(rpmmd.PackageSet{
		Include: []string{
			"cloud-init",
		},
	})
	return pkgset
}
