// nolint: deadcode,unused // Helper functions for future implementations of pipelines
package rhel85

// This file defines package sets that are used by more than one image type.

import "github.com/osbuild/osbuild-composer/internal/rpmmd"

// BUILD PACKAGE SETS

// distro-wide build package set
func buildPackageSet() rpmmd.PackageSet {
	return rpmmd.PackageSet{
		Include: []string{
			"dnf", "dosfstools", "e2fsprogs", "glibc", "lorax-templates-generic",
			"lorax-templates-rhel", "policycoreutils", "python36",
			"python3-iniparse", "qemu-img", "selinux-policy-targeted", "systemd",
			"tar", "xfsprogs", "xz",
		},
	}
}

func x8664BuildPackageSet() rpmmd.PackageSet {
	return rpmmd.PackageSet{
		Include: []string{"grub2-pc"},
	}
}

func ppc64lePackageSet() rpmmd.PackageSet {
	return rpmmd.PackageSet{
		Include: []string{"grub2-ppc64le", "grub2-ppc64le-modules"},
	}
}

func edgeBuildPackageSet() rpmmd.PackageSet {
	return rpmmd.PackageSet{
		Include: []string{
			"dnf", "dosfstools", "e2fsprogs", "grub2-pc", "policycoreutils",
			"python36", "python3-iniparse", "qemu-img", "rpm-ostree", "systemd",
			"tar", "xfsprogs", "xz", "selinux-policy-targeted", "genisoimage",
			"isomd5sum", "xorriso", "syslinux", "lorax-templates-generic",
			"lorax-templates-rhel", "syslinux-nonlinux", "squashfs-tools",
			"grub2-pc-modules", "grub2-tools", "grub2-efi-x64", "shim-x64",
			"efibootmgr", "grub2-tools-minimal", "grub2-tools-extra",
			"grub2-tools-efi", "grub2-efi-x64", "grub2-efi-x64-cdboot",
			"shim-ia32", "grub2-efi-ia32-cdboot",
		},
	}
}

func installerBuildPackageSet() rpmmd.PackageSet {
	return rpmmd.PackageSet{
		Include: []string{
			"efibootmgr", "genisoimage", "grub2-efi-ia32-cdboot",
			"grub2-efi-x64", "grub2-efi-x64-cdboot", "grub2-pc",
			"grub2-pc-modules", "grub2-tools", "grub2-tools-efi",
			"grub2-tools-extra", "grub2-tools-minimal", "isomd5sum",
			"lorax-templates-generic", "lorax-templates-rhel", "rpm-ostree",
			"shim-ia32", "shim-x64", "squashfs-tools", "syslinux",
			"syslinux-nonlinux", "xorriso",
		},
	}
}

// BOOT PACKAGE SETS

func x8664BootPackageSet() rpmmd.PackageSet {
	return rpmmd.PackageSet{
		Include: []string{"dracut-config-generic", "grub2-pc", "grub2-efi-x64", "shim-x64"},
	}
}

func aarch64BootPackageSet() rpmmd.PackageSet {
	return rpmmd.PackageSet{
		Include: []string{
			"dracut-config-generic", "efibootmgr", "grub2-efi-aa64",
			"grub2-tools", "shim-aa64",
		},
	}
}

func ppc64leBootPackageSet() rpmmd.PackageSet {
	return rpmmd.PackageSet{
		Include: []string{
			"dracut-config-generic", "powerpc-utils", "grub2-ppc64le",
			"grub2-ppc64le-modules",
		},
	}
}

func s390xBootPackageSet() rpmmd.PackageSet {
	return rpmmd.PackageSet{
		Include: []string{"dracut-config-generic", "s390utils-base"},
	}
}

func x8664BasePackageSet() rpmmd.PackageSet {
	return rpmmd.PackageSet{
		Include: []string{"grub2-pc"},
	}
}

// INSTALLER PACKAGE SET
func installerPackageSet() rpmmd.PackageSet {
	// TODO: simplify
	return rpmmd.PackageSet{
		Include: []string{
			"aajohan-comfortaa-fonts", "abattis-cantarell-fonts",
			"alsa-firmware", "alsa-tools-firmware", "anaconda",
			"anaconda-dracut", "anaconda-install-env-deps", "anaconda-widgets",
			"audit", "bind-utils", "biosdevname", "bitmap-fangsongti-fonts",
			"bzip2", "cryptsetup", "curl", "dbus-x11", "dejavu-sans-fonts",
			"dejavu-sans-mono-fonts", "device-mapper-persistent-data",
			"dmidecode", "dnf", "dracut-config-generic", "dracut-network",
			"dump", "efibootmgr", "ethtool", "ftp", "gdb-gdbserver", "gdisk",
			"gfs2-utils", "glibc-all-langpacks",
			"google-noto-sans-cjk-ttc-fonts", "grub2-efi-ia32-cdboot",
			"grub2-efi-x64-cdboot", "grub2-tools", "grub2-tools-efi",
			"grub2-tools-extra", "grub2-tools-minimal", "grubby",
			"gsettings-desktop-schemas", "hdparm", "hexedit", "hostname",
			"initscripts", "ipmitool", "iwl1000-firmware", "iwl100-firmware",
			"iwl105-firmware", "iwl135-firmware", "iwl2000-firmware",
			"iwl2030-firmware", "iwl3160-firmware", "iwl3945-firmware",
			"iwl4965-firmware", "iwl5000-firmware", "iwl5150-firmware",
			"iwl6000-firmware", "iwl6000g2a-firmware", "iwl6000g2b-firmware",
			"iwl6050-firmware", "iwl7260-firmware", "jomolhari-fonts",
			"kacst-farsi-fonts", "kacst-qurn-fonts", "kbd", "kbd-misc",
			"kdump-anaconda-addon", "kernel", "khmeros-base-fonts", "less",
			"libblockdev-lvm-dbus", "libertas-sd8686-firmware",
			"libertas-sd8787-firmware", "libertas-usb8388-firmware",
			"libertas-usb8388-olpc-firmware", "libibverbs",
			"libreport-plugin-bugzilla", "libreport-plugin-reportuploader",
			"libreport-rhel-anaconda-bugzilla", "librsvg2", "linux-firmware",
			"lklug-fonts", "lohit-assamese-fonts", "lohit-bengali-fonts",
			"lohit-devanagari-fonts", "lohit-gujarati-fonts",
			"lohit-gurmukhi-fonts", "lohit-kannada-fonts", "lohit-odia-fonts",
			"lohit-tamil-fonts", "lohit-telugu-fonts", "lsof", "madan-fonts",
			"memtest86+", "metacity", "mtr", "mt-st", "net-tools", "nfs-utils",
			"nmap-ncat", "nm-connection-editor", "nss-tools",
			"openssh-clients", "openssh-server", "oscap-anaconda-addon",
			"ostree", "pciutils", "perl-interpreter", "pigz", "plymouth",
			"prefixdevname", "python3-pyatspi", "rdma-core",
			"redhat-release-eula", "rng-tools", "rpcbind", "rpm-ostree",
			"rsync", "rsyslog", "selinux-policy-targeted", "sg3_utils",
			"shim-ia32", "shim-x64", "sil-abyssinica-fonts",
			"sil-padauk-fonts", "sil-scheherazade-fonts", "smartmontools",
			"smc-meera-fonts", "spice-vdagent", "strace", "syslinux",
			"systemd", "system-storage-manager", "tar",
			"thai-scalable-waree-fonts", "tigervnc-server-minimal",
			"tigervnc-server-module", "udisks2", "udisks2-iscsi", "usbutils",
			"vim-minimal", "volume_key", "wget", "xfsdump", "xfsprogs",
			"xorg-x11-drivers", "xorg-x11-fonts-misc", "xorg-x11-server-utils",
			"xorg-x11-server-Xorg", "xorg-x11-xauth", "xz",
		},
	}
}
