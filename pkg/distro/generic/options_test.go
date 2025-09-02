package generic_test

import (
	"testing"

	"github.com/osbuild/blueprint/pkg/blueprint"
	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/generic"
	"github.com/osbuild/images/pkg/ostree"
	"github.com/stretchr/testify/assert"
)

func TestCheckOptions(t *testing.T) {
	type testCase struct {
		distro  string
		it      string
		bp      blueprint.Blueprint
		options distro.ImageOptions
		expErr  string
	}

	testCases := map[string]testCase{
		"f42/ami-ok": {
			distro:  "fedora-42",
			it:      "server-ami",
			bp:      blueprint.Blueprint{},
			options: distro.ImageOptions{},
			expErr:  "",
		},
		"f42/ami-installer-error": {
			distro: "fedora-42",
			it:     "server-ami",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Installer: &blueprint.InstallerCustomization{
						Unattended: true,
					},
				},
			},
			expErr: "blueprint validation failed for image type \"server-ami\": customizations.installer: not supported",
		},
		"f42/ami-ostree-error": {
			distro: "fedora-42",
			it:     "server-ami",
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "OSTree is not supported for \"server-ami\"",
		},
		"f42/ostree-installer-requires-ostree-url": {
			distro: "fedora-42",
			it:     "iot-installer",
			expErr: "options validation failed for image type \"iot-installer\": ostree.url: required",
		},
		"f42/ostree-disk-supported": {
			distro: "fedora-42",
			it:     "iot-qcow2",
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					User:  []blueprint.UserCustomization{{Name: "root"}},
					Group: []blueprint.GroupCustomization{{Name: "admins"}},
					Files: []blueprint.FileCustomization{{
						Path: "/etc/osbuild/stamp",
						Data: "Created by osbuild",
					}},
					Directories: []blueprint.DirectoryCustomization{{
						Path: "/etc/osbuild",
					}},
					Services: &blueprint.ServicesCustomization{
						Disabled: []string{"sshd.service"},
					},
					FIPS: common.ToPtr(true),
				},
			},
		},
		"f42/ostree-disk-not-supported": {
			distro: "fedora-42",
			it:     "iot-qcow2",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					User:  []blueprint.UserCustomization{{Name: "root"}},
					Group: []blueprint.GroupCustomization{{Name: "admins"}},
					Files: []blueprint.FileCustomization{{
						Path: "/etc/osbuild/stamp",
						Data: "Created by osbuild",
					}},
					Directories: []blueprint.DirectoryCustomization{{
						Path: "/etc/osbuild",
					}},
					Services: &blueprint.ServicesCustomization{
						Disabled: []string{"sshd.service"},
					},
					FIPS: common.ToPtr(true),
					Kernel: &blueprint.KernelCustomization{
						Name: "kernel-rt",
					},
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "blueprint validation failed for image type \"iot-qcow2\": customizations.kernel.name: not supported",
		},
		"f42/iot-simplified-requires-install-device": {
			distro: "fedora-42",
			it:     "iot-simplified-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					InstallationDevice: "/dev/null",
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
		},
		"f42/iot-simplified-requires-install-device-error": {
			distro: "fedora-42",
			it:     "iot-simplified-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "blueprint validation failed for image type \"iot-simplified-installer\": customizations.installation_device: required",
		},
		"f42/iot-simplified-supported-customizations": {
			distro: "fedora-42",
			it:     "iot-simplified-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					InstallationDevice: "/dev/null",
					FDO: &blueprint.FDOCustomization{
						DiunPubKeyInsecure:     "true",
						ManufacturingServerURL: "https://example.com/fdo",
					},
					Ignition: &blueprint.IgnitionCustomization{
						FirstBoot: &blueprint.FirstBootIgnitionCustomization{
							ProvisioningURL: "https://example.com/provision",
						},
					},
					Kernel: &blueprint.KernelCustomization{
						Name: "kernel-debug",
					},
					User:  []blueprint.UserCustomization{{Name: "root"}},
					Group: []blueprint.GroupCustomization{{Name: "admins"}},
					FIPS:  common.ToPtr(true),
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
		},
		"f42/iot-simplified-unsupported-customizations": {
			distro: "fedora-42",
			it:     "iot-simplified-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					InstallationDevice: "/dev/null",
					FDO: &blueprint.FDOCustomization{
						DiunPubKeyInsecure:     "true",
						ManufacturingServerURL: "https://example.com/fdo",
					},
					Ignition: &blueprint.IgnitionCustomization{
						FirstBoot: &blueprint.FirstBootIgnitionCustomization{
							ProvisioningURL: "https://example.com/provision",
						},
					},
					Kernel: &blueprint.KernelCustomization{
						Name: "kernel-debug",
					},
					User:  []blueprint.UserCustomization{{Name: "root"}},
					Group: []blueprint.GroupCustomization{{Name: "admins"}},
					FIPS:  common.ToPtr(true),
					Services: &blueprint.ServicesCustomization{
						Disabled: []string{"sshd.service"},
					},
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "blueprint validation failed for image type \"iot-simplified-installer\": customizations.services: not supported",
		},
		"f42/iot-simplified-fdo-requires-manufacturing-url": {
			distro: "fedora-42",
			it:     "iot-simplified-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					InstallationDevice: "/dev/null",
					FDO: &blueprint.FDOCustomization{
						DiunPubKeyInsecure: "true",
					},
					Ignition: &blueprint.IgnitionCustomization{
						FirstBoot: &blueprint.FirstBootIgnitionCustomization{
							ProvisioningURL: "https://example.com/provision",
						},
					},
					Kernel: &blueprint.KernelCustomization{
						Name: "kernel-debug",
					},
					User:  []blueprint.UserCustomization{{Name: "root"}},
					Group: []blueprint.GroupCustomization{{Name: "admins"}},
					FIPS:  common.ToPtr(true),
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "blueprint validation failed for image type \"iot-simplified-installer\": customizations.fdo.manufacturing_server_url: required when using fdo",
		},
		"f42/iot-simplified-fdo-requires-a-diun-option": {
			distro: "fedora-42",
			it:     "iot-simplified-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					InstallationDevice: "/dev/null",
					FDO: &blueprint.FDOCustomization{
						ManufacturingServerURL: "https://example.com/fdo",
					},
					Ignition: &blueprint.IgnitionCustomization{
						FirstBoot: &blueprint.FirstBootIgnitionCustomization{
							ProvisioningURL: "https://example.com/provision",
						},
					},
					Kernel: &blueprint.KernelCustomization{
						Name: "kernel-debug",
					},
					User:  []blueprint.UserCustomization{{Name: "root"}},
					Group: []blueprint.GroupCustomization{{Name: "admins"}},
					FIPS:  common.ToPtr(true),
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "blueprint validation failed for image type \"iot-simplified-installer\": one of customizations.fdo.diun_pub_key_hash, customizations.fdo.diun_pub_key_insecure, customizations.fdo.diun_pub_key_root_certs: required when using fdo",
		},
		"f42/iot-simplified-fdo-requires-exactly-one-diun-option": {
			distro: "fedora-42",
			it:     "iot-simplified-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					InstallationDevice: "/dev/null",
					FDO: &blueprint.FDOCustomization{
						ManufacturingServerURL: "https://example.com/fdo",
						DiunPubKeyInsecure:     "true",
						DiunPubKeyHash:         "ffff",
					},
					Ignition: &blueprint.IgnitionCustomization{
						FirstBoot: &blueprint.FirstBootIgnitionCustomization{
							ProvisioningURL: "https://example.com/provision",
						},
					},
					Kernel: &blueprint.KernelCustomization{
						Name: "kernel-debug",
					},
					User:  []blueprint.UserCustomization{{Name: "root"}},
					Group: []blueprint.GroupCustomization{{Name: "admins"}},
					FIPS:  common.ToPtr(true),
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "blueprint validation failed for image type \"iot-simplified-installer\": one of customizations.fdo.diun_pub_key_hash, customizations.fdo.diun_pub_key_insecure, customizations.fdo.diun_pub_key_root_certs: required when using fdo",
		},
		"f42/iot-simplified-ignition": {
			distro: "fedora-42",
			it:     "iot-simplified-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					InstallationDevice: "/dev/null",
					Ignition: &blueprint.IgnitionCustomization{
						FirstBoot: &blueprint.FirstBootIgnitionCustomization{
							ProvisioningURL: "https://example.com/provision",
						},
					},
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
		},
		"f42/iot-simplified-ignition-no-provisioning-url": {
			distro: "fedora-42",
			it:     "iot-simplified-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					InstallationDevice: "/dev/null",
					Ignition: &blueprint.IgnitionCustomization{
						FirstBoot: &blueprint.FirstBootIgnitionCustomization{},
					},
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "blueprint validation failed for image type \"iot-simplified-installer\": customizations.ignition.firstboot requires customizations.ignition.firstboot.provisioning_url",
		},
		"f42/iot-simplified-ignition-option-conflict": {
			distro: "fedora-42",
			it:     "iot-simplified-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					InstallationDevice: "/dev/null",
					Ignition: &blueprint.IgnitionCustomization{
						Embedded: &blueprint.EmbeddedIgnitionCustomization{
							Config: "/ignition.cfg",
						},
						FirstBoot: &blueprint.FirstBootIgnitionCustomization{
							ProvisioningURL: "https://example.com/provision",
						},
					},
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "blueprint validation failed for image type \"iot-simplified-installer\": customizations.ignition.embedded cannot be used with customizations.ignition.firstboot",
		},

		"f42/iot-installer-supported-customizations": {
			distro: "fedora-42",
			it:     "iot-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					User:  []blueprint.UserCustomization{{Name: "root"}},
					Group: []blueprint.GroupCustomization{{Name: "admins"}},
					FIPS:  common.ToPtr(true),
					Timezone: &blueprint.TimezoneCustomization{
						Timezone: common.ToPtr("UTC"),
					},
					Locale: &blueprint.LocaleCustomization{
						Languages: []string{"en_GB.UTF-8"},
					},
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
		},
		"f42/iot-installer-unsupported-customizations": {
			distro: "fedora-42",
			it:     "iot-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					User:  []blueprint.UserCustomization{{Name: "root"}},
					Group: []blueprint.GroupCustomization{{Name: "admins"}},
					FIPS:  common.ToPtr(true),
					Timezone: &blueprint.TimezoneCustomization{
						Timezone: common.ToPtr("UTC"),
					},
					Locale: &blueprint.LocaleCustomization{
						Languages: []string{"en_GB.UTF-8"},
					},
					Kernel: &blueprint.KernelCustomization{
						Name: "kernel-rt",
					},
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "blueprint validation failed for image type \"iot-installer\": customizations.kernel: not supported",
		},

		"f42/live-installer-no-installer-customizations": {
			distro: "fedora-42",
			it:     "workstation-live-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Installer: &blueprint.InstallerCustomization{
						Unattended: true,
					},
				},
			},
		},
		"f42/live-installer-unsupported-customizations": {
			distro: "fedora-42",
			it:     "workstation-live-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					User: []blueprint.UserCustomization{{Name: "root"}},
					Installer: &blueprint.InstallerCustomization{
						Unattended: true,
					},
				},
			},
			expErr: "blueprint validation failed for image type \"workstation-live-installer\": customizations.user: not supported",
		},

		"f42/ostree-types-no-oscap": {
			distro: "fedora-42",
			it:     "iot-container",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					OpenSCAP: &blueprint.OpenSCAPCustomization{
						ProfileID: "xccdf_org.ssgproject.content_profile_ospp",
					},
				},
			},
			expErr: "blueprint validation failed for image type \"iot-container\": customizations.openscap: not supported",
		},

		"f42/iot-installer-installer-customizations": {
			distro: "fedora-42",
			it:     "iot-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Installer: &blueprint.InstallerCustomization{
						Unattended: true,
					},
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
		},
		"f42/iot-installer-bad-combinations": {
			distro: "fedora-42",
			it:     "iot-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					User: []blueprint.UserCustomization{{Name: "root"}},
					Installer: &blueprint.InstallerCustomization{
						Kickstart: &blueprint.Kickstart{
							Contents: "echo 'Testing'",
						},
					},
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "blueprint validation failed for image type \"iot-installer\": customizations.installer.kickstart.contents cannot be used with customizations.user or customizations.group",
		},

		"f42/ostree-disk-unsupported-containers": {
			distro: "fedora-42",
			it:     "iot-qcow2",
			bp: blueprint.Blueprint{
				Containers: []blueprint.Container{
					{
						Source: "example.org/containers/test:42",
					},
				},
			},
			expErr: "blueprint validation failed for image type \"iot-qcow2\": containers: not supported",
		},

		"f42/ostree-commit-unsupported-kernel-append": {
			distro: "fedora-42",
			it:     "iot-commit",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Kernel: &blueprint.KernelCustomization{
						Append: "debug",
					},
				},
			},
			expErr: "blueprint validation failed for image type \"iot-commit\": customizations.kernel.append: not supported",
		},

		"f42/oscap-empty-profile": {
			distro: "fedora-42",
			it:     "vhd",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					OpenSCAP: &blueprint.OpenSCAPCustomization{
						ProfileID: "",
					},
				},
			},
			expErr: "blueprint validation failed for image type \"server-vhd\": customizations.oscap.profile_id: required when using customizations.oscap",
		},

		"f42/ostree-disk-requires-ostree-url": {
			distro: "fedora-42",
			it:     "iot-qcow2",
			expErr: "options validation failed for image type \"iot-qcow2\": ostree.url: required",
		},
		"f42/ostree-disk2-requires-ostree-url": {
			distro: "fedora-42",
			it:     "iot-raw-xz",
			expErr: "options validation failed for image type \"iot-raw-xz\": ostree.url: required",
		},

		"r8/ami-ok": {
			distro:  "rhel-8.10",
			it:      "ami",
			bp:      blueprint.Blueprint{},
			options: distro.ImageOptions{},
			expErr:  "",
		},
		"r8/ami-installer-error": {
			distro: "rhel-8.10",
			it:     "ami",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Installer: &blueprint.InstallerCustomization{
						Unattended: true,
					},
				},
			},
			expErr: "installer customizations are not supported for \"ami\"",
		},
		"r8/ami-ostree-error": {
			distro: "rhel-8.10",
			it:     "ami",
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "OSTree is not supported for \"ami\"",
		},
		"r8/ostree-installer-requires-ostree-url": {
			distro: "rhel-8.10",
			it:     "edge-installer",
			expErr: "boot ISO image type \"edge-installer\" requires specifying a URL from which to retrieve the OSTree commit",
		},
		"r8/ostree-disk-supported": {
			distro: "rhel-8.10",
			it:     "edge-raw-image",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					User:  []blueprint.UserCustomization{{Name: "root"}},
					Group: []blueprint.GroupCustomization{{Name: "admins"}},
					FIPS:  common.ToPtr(true),
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
		},
		"r8/ostree-disk-not-supported": {
			distro: "rhel-8.10",
			it:     "edge-raw-image",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					User:  []blueprint.UserCustomization{{Name: "root"}},
					Group: []blueprint.GroupCustomization{{Name: "admins"}},
					Files: []blueprint.FileCustomization{{
						Path: "/etc/osbuild/stamp",
						Data: "Created by osbuild",
					}},
					Directories: []blueprint.DirectoryCustomization{{
						Path: "/etc/osbuild",
					}},
					Services: &blueprint.ServicesCustomization{
						Disabled: []string{"sshd.service"},
					},
					FIPS: common.ToPtr(true),
					Kernel: &blueprint.KernelCustomization{
						Name: "kernel-rt",
					},
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "unsupported blueprint customizations found for image type \"edge-raw-image\": (allowed: User, Group, FIPS)",
		},
		"r8/edge-simplified-requires-install-device": {
			distro: "rhel-8.10",
			it:     "edge-simplified-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					InstallationDevice: "/dev/null",
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
		},
		"r8/edge-simplified-requires-install-device-error": {
			distro: "rhel-8.10",
			it:     "edge-simplified-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "boot ISO image type \"edge-simplified-installer\" requires specifying an installation device to install to",
		},
		"r8/edge-simplified-supported-customizations": {
			distro: "rhel-8.10",
			it:     "edge-simplified-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					InstallationDevice: "/dev/null",
					FDO: &blueprint.FDOCustomization{
						DiunPubKeyInsecure:     "true",
						ManufacturingServerURL: "https://example.com/fdo",
					},
					User:  []blueprint.UserCustomization{{Name: "root"}},
					Group: []blueprint.GroupCustomization{{Name: "admins"}},
					FIPS:  common.ToPtr(true),
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
		},
		"r8/edge-simplified-unsupported-customizations": {
			distro: "rhel-8.10",
			it:     "edge-simplified-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					InstallationDevice: "/dev/null",
					FDO: &blueprint.FDOCustomization{
						DiunPubKeyInsecure:     "true",
						ManufacturingServerURL: "https://example.com/fdo",
					},
					Ignition: &blueprint.IgnitionCustomization{
						FirstBoot: &blueprint.FirstBootIgnitionCustomization{
							ProvisioningURL: "https://example.com/provision",
						},
					},
					Kernel: &blueprint.KernelCustomization{
						Name: "kernel-debug",
					},
					User:  []blueprint.UserCustomization{{Name: "root"}},
					Group: []blueprint.GroupCustomization{{Name: "admins"}},
					FIPS:  common.ToPtr(true),
					Services: &blueprint.ServicesCustomization{
						Disabled: []string{"sshd.service"},
					},
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "unsupported blueprint customizations found for image type \"edge-simplified-installer\": (allowed: InstallationDevice, FDO, User, Group, FIPS)",
		},
		"r8/edge-simplified-fdo-requires-manufacturing-url": {
			distro: "rhel-8.10",
			it:     "edge-simplified-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					InstallationDevice: "/dev/null",
					FDO: &blueprint.FDOCustomization{
						DiunPubKeyInsecure: "true",
					},
					User:  []blueprint.UserCustomization{{Name: "root"}},
					Group: []blueprint.GroupCustomization{{Name: "admins"}},
					FIPS:  common.ToPtr(true),
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "boot ISO image type \"edge-simplified-installer\" requires specifying FDO.ManufacturingServerURL configuration to install to",
		},
		"r8/edge-simplified-fdo-requires-a-diun-option": {
			distro: "rhel-8.10",
			it:     "edge-simplified-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					InstallationDevice: "/dev/null",
					FDO: &blueprint.FDOCustomization{
						ManufacturingServerURL: "https://example.com/fdo",
					},
					User:  []blueprint.UserCustomization{{Name: "root"}},
					Group: []blueprint.GroupCustomization{{Name: "admins"}},
					FIPS:  common.ToPtr(true),
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "boot ISO image type \"edge-simplified-installer\" requires specifying one of [FDO.DiunPubKeyHash,FDO.DiunPubKeyInsecure,FDO.DiunPubKeyRootCerts] configuration to install to",
		},
		"r8/edge-simplified-fdo-requires-exactly-one-diun-option": {
			distro: "rhel-8.10",
			it:     "edge-simplified-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					InstallationDevice: "/dev/null",
					FDO: &blueprint.FDOCustomization{
						ManufacturingServerURL: "https://example.com/fdo",
						DiunPubKeyInsecure:     "true",
						DiunPubKeyHash:         "ffff",
					},
					User:  []blueprint.UserCustomization{{Name: "root"}},
					Group: []blueprint.GroupCustomization{{Name: "admins"}},
					FIPS:  common.ToPtr(true),
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "boot ISO image type \"edge-simplified-installer\" requires specifying one of [FDO.DiunPubKeyHash,FDO.DiunPubKeyInsecure,FDO.DiunPubKeyRootCerts] configuration to install to",
		},

		"r8/edge-installer-supported-customizations": {
			distro: "rhel-8.10",
			it:     "edge-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					User:  []blueprint.UserCustomization{{Name: "root"}},
					Group: []blueprint.GroupCustomization{{Name: "admins"}},
					FIPS:  common.ToPtr(true),
					Timezone: &blueprint.TimezoneCustomization{
						Timezone: common.ToPtr("UTC"),
					},
					Locale: &blueprint.LocaleCustomization{
						Languages: []string{"en_GB.UTF-8"},
					},
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
		},
		"r8/edge-installer-unsupported-customizations": {
			distro: "rhel-8.10",
			it:     "edge-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					User:  []blueprint.UserCustomization{{Name: "root"}},
					Group: []blueprint.GroupCustomization{{Name: "admins"}},
					FIPS:  common.ToPtr(true),
					Timezone: &blueprint.TimezoneCustomization{
						Timezone: common.ToPtr("UTC"),
					},
					Locale: &blueprint.LocaleCustomization{
						Languages: []string{"en_GB.UTF-8"},
					},
					Kernel: &blueprint.KernelCustomization{
						Name: "kernel-rt",
					},
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "unsupported blueprint customizations found for image type \"edge-installer\": (allowed: User, Group, FIPS, Installer, Timezone, Locale)",
		},

		"r8/ostree-types-no-oscap": {
			distro: "rhel-8.10",
			it:     "edge-container",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					OpenSCAP: &blueprint.OpenSCAPCustomization{
						ProfileID: "xccdf_org.ssgproject.content_profile_ospp",
					},
				},
			},
			expErr: "OpenSCAP customizations are not supported for ostree types",
		},

		"r8/edge-installer-installer-customizations": {
			distro: "rhel-8.10",
			it:     "edge-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Installer: &blueprint.InstallerCustomization{
						Unattended: true,
					},
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
		},
		"r8/edge-installer-bad-combinations": {
			distro: "rhel-8.10",
			it:     "edge-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					User: []blueprint.UserCustomization{{Name: "root"}},
					Installer: &blueprint.InstallerCustomization{
						Kickstart: &blueprint.Kickstart{
							Contents: "echo 'Testing'",
						},
					},
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "edge-installer installer.kickstart.contents are not supported in combination with users or groups",
		},

		"r8/ostree-disk-requires-ostree-url": {
			distro: "rhel-8.10",
			it:     "edge-raw-image",
			expErr: "\"edge-raw-image\" images require specifying a URL from which to retrieve the OSTree commit",
		},

		"r8/ostree-no-containers": {
			distro: "rhel-8.10",
			it:     "edge-raw-image",
			bp: blueprint.Blueprint{
				Containers: []blueprint.Container{
					{
						Source: "example.org/containers/test:42",
					},
				},
			},
			expErr: "embedding containers is not supported for edge-raw-image on rhel-8.10",
		},

		"r8/ostree-commit-unsupported-kernel-append": {
			distro: "rhel-8.10",
			it:     "edge-commit",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Kernel: &blueprint.KernelCustomization{
						Append: "debug",
					},
				},
			},
			expErr: "kernel boot parameter customizations are not supported for ostree types",
		},

		"r8/ostree-mountpoints-not-supported": {
			distro: "rhel-8.10",
			it:     "edge-commit",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Filesystem: []blueprint.FilesystemCustomization{
						{
							Mountpoint: "/data",
						},
					},
				},
			},
			expErr: "Custom mountpoints and partitioning are not supported for ostree types",
		},

		"r8/ostree-partitioning-not-supported": {
			distro: "rhel-8.10",
			it:     "edge-commit",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Disk: &blueprint.DiskCustomization{
						Partitions: []blueprint.PartitionCustomization{
							{
								Type: "plain",
								FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
									Mountpoint: "/data",
									FSType:     "ext4",
								},
							},
						},
					},
				},
			},
			// TODO: this should be an error
		},

		"r8/oscap-empty-profile": {
			distro: "rhel-8.10",
			it:     "vhd",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					OpenSCAP: &blueprint.OpenSCAPCustomization{
						ProfileID: "",
					},
				},
			},
			expErr: "OpenSCAP profile cannot be empty",
		},

		"r9/ami-ok": {
			distro:  "rhel-9.7",
			it:      "ami",
			bp:      blueprint.Blueprint{},
			options: distro.ImageOptions{},
			expErr:  "",
		},
		"r9/ami-installer-error": {
			distro: "rhel-9.7",
			it:     "ami",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Installer: &blueprint.InstallerCustomization{
						Unattended: true,
					},
				},
			},
			expErr: "installer customizations are not supported for \"ami\"",
		},
		"r9/ami-ostree-error": {
			distro: "rhel-9.7",
			it:     "ami",
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "OSTree is not supported for \"ami\"",
		},
		"r9/ostree-installer-requires-ostree-url": {
			distro: "rhel-9.7",
			it:     "edge-installer",
			expErr: "boot ISO image type \"edge-installer\" requires specifying a URL from which to retrieve the OSTree commit",
		},
		"r9/ostree-disk-supported": {
			distro: "rhel-9.7",
			it:     "edge-raw-image",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					User:  []blueprint.UserCustomization{{Name: "root"}},
					Group: []blueprint.GroupCustomization{{Name: "admins"}},
					FIPS:  common.ToPtr(true),
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
		},
		"r9/ostree-disk-not-supported": {
			distro: "rhel-9.7",
			it:     "edge-raw-image",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					User:  []blueprint.UserCustomization{{Name: "root"}},
					Group: []blueprint.GroupCustomization{{Name: "admins"}},
					Files: []blueprint.FileCustomization{{
						Path: "/etc/osbuild/stamp",
						Data: "Created by osbuild",
					}},
					Directories: []blueprint.DirectoryCustomization{{
						Path: "/etc/osbuild",
					}},
					Services: &blueprint.ServicesCustomization{
						Disabled: []string{"sshd.service"},
					},
					FIPS: common.ToPtr(true),
					Kernel: &blueprint.KernelCustomization{
						Name: "kernel-rt",
					},
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "unsupported blueprint customizations found for image type \"edge-raw-image\": (allowed: Ignition, Kernel, User, Group, FIPS, Filesystem)",
		},
		"r9/edge-simplified-requires-install-device": {
			distro: "rhel-9.7",
			it:     "edge-simplified-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					InstallationDevice: "/dev/null",
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
		},
		"r9/edge-simplified-requires-install-device-error": {
			distro: "rhel-9.7",
			it:     "edge-simplified-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "boot ISO image type \"edge-simplified-installer\" requires specifying an installation device to install to",
		},
		"r9/edge-simplified-supported-customizations": {
			distro: "rhel-9.7",
			it:     "edge-simplified-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					InstallationDevice: "/dev/null",
					FDO: &blueprint.FDOCustomization{
						DiunPubKeyInsecure:     "true",
						ManufacturingServerURL: "https://example.com/fdo",
					},
					Ignition: &blueprint.IgnitionCustomization{
						FirstBoot: &blueprint.FirstBootIgnitionCustomization{
							ProvisioningURL: "https://example.com/provision",
						},
					},
					Kernel: &blueprint.KernelCustomization{
						Name: "kernel-debug",
					},
					User:  []blueprint.UserCustomization{{Name: "root"}},
					Group: []blueprint.GroupCustomization{{Name: "admins"}},
					FIPS:  common.ToPtr(true),
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
		},
		"r9/edge-simplified-unsupported-customizations": {
			distro: "rhel-9.7",
			it:     "edge-simplified-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					InstallationDevice: "/dev/null",
					FDO: &blueprint.FDOCustomization{
						DiunPubKeyInsecure:     "true",
						ManufacturingServerURL: "https://example.com/fdo",
					},
					Ignition: &blueprint.IgnitionCustomization{
						FirstBoot: &blueprint.FirstBootIgnitionCustomization{
							ProvisioningURL: "https://example.com/provision",
						},
					},
					Kernel: &blueprint.KernelCustomization{
						Name: "kernel-debug",
					},
					User:  []blueprint.UserCustomization{{Name: "root"}},
					Group: []blueprint.GroupCustomization{{Name: "admins"}},
					FIPS:  common.ToPtr(true),
					Services: &blueprint.ServicesCustomization{
						Disabled: []string{"sshd.service"},
					},
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "unsupported blueprint customizations found for image type \"edge-simplified-installer\": (allowed: InstallationDevice, FDO, Ignition, Kernel, User, Group, FIPS, Filesystem)",
		},
		"r9/edge-simplified-fdo-does-not-require-manufacturing-url": {
			distro: "rhel-9.7",
			it:     "edge-simplified-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					InstallationDevice: "/dev/null",
					FDO: &blueprint.FDOCustomization{
						DiunPubKeyInsecure: "true",
					},
					Ignition: &blueprint.IgnitionCustomization{
						FirstBoot: &blueprint.FirstBootIgnitionCustomization{
							ProvisioningURL: "https://example.com/provision",
						},
					},
					Kernel: &blueprint.KernelCustomization{
						Name: "kernel-debug",
					},
					User:  []blueprint.UserCustomization{{Name: "root"}},
					Group: []blueprint.GroupCustomization{{Name: "admins"}},
					FIPS:  common.ToPtr(true),
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "boot ISO image type \"edge-simplified-installer\" requires specifying FDO.ManufacturingServerURL configuration to install to when using FDO",
		},
		"r9/edge-simplified-fdo-requires-a-diun-option": {
			distro: "rhel-9.7",
			it:     "edge-simplified-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					InstallationDevice: "/dev/null",
					FDO: &blueprint.FDOCustomization{
						ManufacturingServerURL: "https://example.com/fdo",
					},
					Ignition: &blueprint.IgnitionCustomization{
						FirstBoot: &blueprint.FirstBootIgnitionCustomization{
							ProvisioningURL: "https://example.com/provision",
						},
					},
					Kernel: &blueprint.KernelCustomization{
						Name: "kernel-debug",
					},
					User:  []blueprint.UserCustomization{{Name: "root"}},
					Group: []blueprint.GroupCustomization{{Name: "admins"}},
					FIPS:  common.ToPtr(true),
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "boot ISO image type \"edge-simplified-installer\" requires specifying one of [FDO.DiunPubKeyHash,FDO.DiunPubKeyInsecure,FDO.DiunPubKeyRootCerts] configuration to install to when using FDO",
		},
		"r9/edge-simplified-fdo-requires-exactly-one-diun-option": {
			distro: "rhel-9.7",
			it:     "edge-simplified-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					InstallationDevice: "/dev/null",
					FDO: &blueprint.FDOCustomization{
						ManufacturingServerURL: "https://example.com/fdo",
						DiunPubKeyInsecure:     "true",
						DiunPubKeyHash:         "ffff",
					},
					Ignition: &blueprint.IgnitionCustomization{
						FirstBoot: &blueprint.FirstBootIgnitionCustomization{
							ProvisioningURL: "https://example.com/provision",
						},
					},
					Kernel: &blueprint.KernelCustomization{
						Name: "kernel-debug",
					},
					User:  []blueprint.UserCustomization{{Name: "root"}},
					Group: []blueprint.GroupCustomization{{Name: "admins"}},
					FIPS:  common.ToPtr(true),
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "boot ISO image type \"edge-simplified-installer\" requires specifying one of [FDO.DiunPubKeyHash,FDO.DiunPubKeyInsecure,FDO.DiunPubKeyRootCerts] configuration to install to when using FDO",
		},
		"r9/edge-simplified-ignition": {
			distro: "rhel-9.7",
			it:     "edge-simplified-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					InstallationDevice: "/dev/null",
					Ignition: &blueprint.IgnitionCustomization{
						FirstBoot: &blueprint.FirstBootIgnitionCustomization{
							ProvisioningURL: "https://example.com/provision",
						},
					},
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
		},
		"r9/edge-simplified-ignition-no-provisioning-url": {
			distro: "rhel-9.7",
			it:     "edge-simplified-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					InstallationDevice: "/dev/null",
					Ignition: &blueprint.IgnitionCustomization{
						FirstBoot: &blueprint.FirstBootIgnitionCustomization{},
					},
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "ignition.firstboot requires a provisioning url",
		},
		"r9/edge-simplified-ignition-option-conflict": {
			distro: "rhel-9.7",
			it:     "edge-simplified-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					InstallationDevice: "/dev/null",
					Ignition: &blueprint.IgnitionCustomization{
						Embedded: &blueprint.EmbeddedIgnitionCustomization{
							Config: "/ignition.cfg",
						},
						FirstBoot: &blueprint.FirstBootIgnitionCustomization{
							ProvisioningURL: "https://example.com/provision",
						},
					},
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "both ignition embedded and firstboot configurations found",
		},

		"r9/edge-installer-supported-customizations": {
			distro: "rhel-9.7",
			it:     "edge-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					User:  []blueprint.UserCustomization{{Name: "root"}},
					Group: []blueprint.GroupCustomization{{Name: "admins"}},
					FIPS:  common.ToPtr(true),
					Timezone: &blueprint.TimezoneCustomization{
						Timezone: common.ToPtr("UTC"),
					},
					Locale: &blueprint.LocaleCustomization{
						Languages: []string{"en_GB.UTF-8"},
					},
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
		},
		"r9/edge-installer-unsupported-customizations": {
			distro: "rhel-9.7",
			it:     "edge-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					User:  []blueprint.UserCustomization{{Name: "root"}},
					Group: []blueprint.GroupCustomization{{Name: "admins"}},
					FIPS:  common.ToPtr(true),
					Timezone: &blueprint.TimezoneCustomization{
						Timezone: common.ToPtr("UTC"),
					},
					Locale: &blueprint.LocaleCustomization{
						Languages: []string{"en_GB.UTF-8"},
					},
					Kernel: &blueprint.KernelCustomization{
						Name: "kernel-rt",
					},
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "unsupported blueprint customizations found for image type \"edge-installer\": (allowed: User, Group, FIPS, Installer, Timezone, Locale)",
		},

		"r9/ostree-types-no-oscap": {
			distro: "rhel-9.7",
			it:     "edge-container",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					OpenSCAP: &blueprint.OpenSCAPCustomization{
						ProfileID: "xccdf_org.ssgproject.content_profile_ospp",
					},
				},
			},
			expErr: "OpenSCAP customizations are not supported for ostree types",
		},

		"r9/oscap-empty-profile": {
			distro: "rhel-9.7",
			it:     "vhd",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					OpenSCAP: &blueprint.OpenSCAPCustomization{
						ProfileID: "",
					},
				},
			},
			expErr: "OpenSCAP profile cannot be empty",
		},

		"r9/edge-installer-installer-customizations": {
			distro: "rhel-9.7",
			it:     "edge-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Installer: &blueprint.InstallerCustomization{
						Unattended: true,
					},
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
		},
		"r9/edge-installer-bad-combinations": {
			distro: "rhel-9.7",
			it:     "edge-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					User: []blueprint.UserCustomization{{Name: "root"}},
					Installer: &blueprint.InstallerCustomization{
						Kickstart: &blueprint.Kickstart{
							Contents: "echo 'Testing'",
						},
					},
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "edge-installer installer.kickstart.contents are not supported in combination with users or groups",
		},

		"r9/ostree-disk-unsupported-containers": {
			distro: "rhel-9.7",
			it:     "edge-ami",
			bp: blueprint.Blueprint{
				Containers: []blueprint.Container{
					{
						Source: "example.org/containers/test:42",
					},
				},
			},
			expErr: "embedding containers is not supported for edge-ami on rhel-9.7",
		},

		"r9/ostree-commit-unsupported-kernel-append": {
			distro: "rhel-9.7",
			it:     "edge-commit",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Kernel: &blueprint.KernelCustomization{
						Append: "debug",
					},
				},
			},
			expErr: "kernel boot parameter customizations are not supported for ostree types",
		},

		"r9/ostree-disk-requires-ostree-url": {
			distro: "rhel-9.7",
			it:     "edge-vsphere",
			expErr: "\"edge-vsphere\" images require specifying a URL from which to retrieve the OSTree commit",
		},
		"r9/ostree-disk2-requires-ostree-url": {
			distro: "rhel-9.7",
			it:     "edge-ami",
			expErr: "\"edge-ami\" images require specifying a URL from which to retrieve the OSTree commit",
		},

		"r9/ostree-mountpoints-not-supported": {
			distro: "rhel-9.7",
			it:     "edge-commit",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Filesystem: []blueprint.FilesystemCustomization{
						{
							Mountpoint: "/data",
						},
					},
				},
			},
			expErr: "custom mountpoints and partitioning are not supported for ostree types",
		},

		"r9/ostree-partitioning-not-supported": {
			distro: "rhel-9.7",
			it:     "edge-commit",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Disk: &blueprint.DiskCustomization{
						Partitions: []blueprint.PartitionCustomization{
							{
								Type: "plain",
								FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
									Mountpoint: "/data",
									FSType:     "ext4",
								},
							},
						},
					},
				},
			},
			expErr: "custom mountpoints and partitioning are not supported for ostree types",
		},

		"r9/ostree-disk-mountpoints-supported": {
			distro: "rhel-9.7",
			it:     "edge-vsphere",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Filesystem: []blueprint.FilesystemCustomization{
						{
							Mountpoint: "/data",
						},
					},
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
		},

		"r9/ostree-disk-partitioning-unsupported": {
			distro: "rhel-9.7",
			it:     "edge-vsphere",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Disk: &blueprint.DiskCustomization{
						Partitions: []blueprint.PartitionCustomization{
							{
								Type: "plain",
								FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
									Mountpoint: "/data",
									FSType:     "ext4",
								},
							},
						},
					},
				},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			// NOTE: this should be supported
			expErr: "unsupported blueprint customizations found for image type \"edge-vsphere\": (allowed: Ignition, Kernel, User, Group, FIPS, Filesystem)",
		},

		"r9/cvm-kernel-unsupported": {
			distro: "rhel-9.7",
			it:     "azure-cvm",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Kernel: &blueprint.KernelCustomization{
						Name: "kernel-rt",
					},
				},
			},
			expErr: "kernel customizations are not supported for \"azure-cvm\"",
		},

		"r10/ami-ok": {
			distro:  "rhel-10.0",
			it:      "ami",
			bp:      blueprint.Blueprint{},
			options: distro.ImageOptions{},
			expErr:  "",
		},
		"r10/ami-installer-error": {
			distro: "rhel-10.0",
			it:     "ami",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Installer: &blueprint.InstallerCustomization{
						Unattended: true,
					},
				},
			},
			expErr: "installer customizations are not supported for \"ami\"",
		},
		"r10/ami-ostree-error": {
			distro: "rhel-10.0",
			it:     "ami",
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "OSTree is not supported for \"ami\"",
		},

		"r10/oscap-empty-profile": {
			distro: "rhel-10.0",
			it:     "vhd",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					OpenSCAP: &blueprint.OpenSCAPCustomization{
						ProfileID: "",
					},
				},
			},
			expErr: "OpenSCAP profile cannot be empty",
		},

		"r10/cvm-kernel-unsupported": {
			distro: "rhel-10.0",
			it:     "azure-cvm",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Kernel: &blueprint.KernelCustomization{
						Name: "kernel-rt",
					},
				},
			},
			// NOTE: this should be an error
		},

		"r7/ok": {
			distro:  "rhel-7.9",
			it:      "qcow2",
			bp:      blueprint.Blueprint{},
			options: distro.ImageOptions{},
			expErr:  "",
		},

		"r7/no-containers": {
			distro: "rhel-7.9",
			it:     "azure-rhui",
			bp: blueprint.Blueprint{
				Containers: []blueprint.Container{
					{
						Name: "example.org/containers/some-kind-of-image:100",
					},
				},
			},
			expErr: "embedding containers is not supported for azure-rhui on rhel-7.9",
		},

		"r7/oscap-empty-profile": {
			distro: "rhel-7.9",
			it:     "qcow2",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					OpenSCAP: &blueprint.OpenSCAPCustomization{
						ProfileID: "xccdf_org.ssgproject.content_profile_ospp",
					},
				},
			},
			expErr: "OpenSCAP unsupported os version: 7.9",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			d := generic.DistroFactory(tc.distro)
			// NOTE: The architecture is only relevant in one small case: swap
			// partitions on RHEL 8. Let's ignore that for now and return to it
			// when we redo the validation.
			imageTypes, err := d.GetArch("x86_64")
			assert.NoError(err)
			it, err := imageTypes.GetImageType(tc.it)
			assert.NoError(err)

			genit, ok := it.(*generic.ImageType) // checkOptions() function is defined on generic.ImageType
			assert.True(ok, "image type %q for distro %q does not appear to be valid", tc.it, d.Name())
			_, err = generic.ImageTypeCheckOptions(genit, &tc.bp, tc.options)
			if tc.expErr == "" {
				assert.NoError(err)
			} else {
				assert.EqualError(err, tc.expErr)
			}
		})
	}
}
