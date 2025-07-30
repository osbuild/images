package generic_test

import (
	"testing"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/generic"
	"github.com/osbuild/images/pkg/ostree"
	"github.com/stretchr/testify/assert"
)

func TestCheckOptionsFedora(t *testing.T) {
	type testCase struct {
		it      string
		bp      blueprint.Blueprint
		options distro.ImageOptions
		expErr  string
	}

	// For this test, we just need ImageType instances with a couple of fields
	// set (name, RPMOSTree). However, it's impossible to create one with a
	// given name, because the name is private inside the ImageTypeYAML and
	// meant to only be set by the loader. So we use the real image types,
	// loaded from the YAML files into ImageTypeYAML and create the ImageType
	// itself directly.
	fedora := generic.DistroFactory("fedora-42")
	imageTypes, err := fedora.GetArch("x86_64")
	assert.NoError(t, err)

	testCases := map[string]testCase{
		"ami-ok": {
			it:      "server-ami",
			bp:      blueprint.Blueprint{},
			options: distro.ImageOptions{},
			expErr:  "",
		},
		"ami-installer-error": {
			it: "server-ami",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Installer: &blueprint.InstallerCustomization{
						Unattended: true,
					},
				},
			},
			expErr: "installer customizations are not supported for \"server-ami\"",
		},
		"ami-ostree-error": {
			it: "server-ami",
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "OSTree is not supported for \"server-ami\"",
		},
		"ostree-installer-requires-ostree-url": {
			it:     "iot-installer",
			expErr: "boot ISO image type \"iot-installer\" requires specifying a URL from which to retrieve the OSTree commit",
		},
		"ostree-disk-supported": {
			it: "iot-qcow2",
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
			// NOTE: this should also require an ostree URL
		},
		"ostree-disk-not-supported": {
			it: "iot-qcow2",
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
			expErr: "unsupported blueprint customizations found for image type \"iot-qcow2\": (allowed: User, Group, Directories, Files, Services, FIPS)",
		},
		"iot-simplified-requires-install-device": {
			it: "iot-simplified-installer",
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
		"iot-simplified-requires-install-device-error": {
			it: "iot-simplified-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{},
			},
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			expErr: "boot ISO image type \"iot-simplified-installer\" requires specifying an installation device to install to",
		},
		"iot-simplified-supported-customizations": {
			it: "iot-simplified-installer",
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
		"iot-simplified-unsupported-customizations": {
			it: "iot-simplified-installer",
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
			expErr: "unsupported blueprint customizations found for image type \"iot-simplified-installer\": (allowed: InstallationDevice, FDO, Ignition, Kernel, User, Group, FIPS)",
		},
		"iot-simplified-fdo-requires-manufacturing-url": {
			it: "iot-simplified-installer",
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
			expErr: "boot ISO image type \"iot-simplified-installer\" requires specifying FDO.ManufacturingServerURL configuration to install to when using FDO",
		},
		"iot-simplified-fdo-requires-a-diun-option": {
			it: "iot-simplified-installer",
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
			expErr: "boot ISO image type \"iot-simplified-installer\" requires specifying one of [FDO.DiunPubKeyHash,FDO.DiunPubKeyInsecure,FDO.DiunPubKeyRootCerts] configuration to install to when using FDO",
		},
		"iot-simplified-fdo-requires-exactly-one-diun-option": {
			it: "iot-simplified-installer",
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
			expErr: "boot ISO image type \"iot-simplified-installer\" requires specifying one of [FDO.DiunPubKeyHash,FDO.DiunPubKeyInsecure,FDO.DiunPubKeyRootCerts] configuration to install to when using FDO",
		},
		"iot-simplified-ignition": {
			it: "iot-simplified-installer",
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
		"iot-simplified-ignition-no-provisioning-url": {
			it: "iot-simplified-installer",
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
		"iot-simplified-ignition-option-conflict": {
			it: "iot-simplified-installer",
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

		"iot-installer-supported-customizations": {
			it: "iot-installer",
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
		"iot-installer-unsupported-customizations": {
			it: "iot-installer",
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
			expErr: "unsupported blueprint customizations found for image type \"iot-installer\": (allowed: User, Group, FIPS, Installer, Timezone, Locale)",
		},

		"live-installer-no-installer-customizations": {
			it: "workstation-live-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Installer: &blueprint.InstallerCustomization{
						Unattended: true,
					},
				},
			},
			// NOTE: this is listed as supported in the checks that are
			// specific to the image type but the image type is not listed as
			// supporting installer customizations later in the function
			expErr: "installer customizations are not supported for \"workstation-live-installer\"",
		},
		"live-installer-unsupported-customizations": {
			it: "workstation-live-installer",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					User: []blueprint.UserCustomization{{Name: "root"}},
					Installer: &blueprint.InstallerCustomization{
						Unattended: true,
					},
				},
			},
			expErr: "image type \"workstation-live-installer\" does not support customizations",
		},

		"ostree-types-no-oscap": {
			it: "iot-container",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					OpenSCAP: &blueprint.OpenSCAPCustomization{
						ProfileID: "xccdf_org.ssgproject.content_profile_ospp",
					},
				},
			},
			expErr: "OpenSCAP customizations are not supported for ostree types",
		},

		"iot-installer-installer-customizations": {
			it: "iot-installer",
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
		"iot-installer-bad-combinations": {
			it: "iot-installer",
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
			expErr: "iot-installer installer.kickstart.contents are not supported in combination with users or groups",
		},

		"ostree-disk-unsupported-containers": {
			it: "iot-qcow2",
			bp: blueprint.Blueprint{
				Containers: []blueprint.Container{
					{
						Source: "example.org/containers/test:42",
					},
				},
			},
			expErr: "embedding containers is not supported for iot-qcow2 on fedora-42",
		},

		"ostree-commit-unsupported-kernel-append": {
			it: "iot-commit",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Kernel: &blueprint.KernelCustomization{
						Append: "debug",
					},
				},
			},
			expErr: "kernel boot parameter customizations are not supported for ostree types",
		},

		"oscap-empty-profile": {
			it: "vhd",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					OpenSCAP: &blueprint.OpenSCAPCustomization{
						ProfileID: "",
					},
				},
			},
			expErr: "OpenSCAP profile cannot be empty",
		},

		// NOTE: the following tests verify the current behaviour of the
		// function, but the behaviour itself is wrong
		"ostree-disk-requires-ostree-url": {
			it:     "iot-qcow2",
			expErr: "", // NOTE: it should require a URL
		},
		"ostree-disk2-requires-ostree-url": {
			it:     "iot-raw-xz",
			expErr: "", // NOTE: it should require a URL
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			it, err := imageTypes.GetImageType(tc.it)
			assert.NoError(err)
			genit, ok := it.(*generic.ImageType) // checkOptions() functions require generic.ImageType
			assert.True(ok, "image type %q for distro %q does not appear to be valid", tc.it, fedora.Name())
			_, err = generic.CheckOptionsFedora(genit, &tc.bp, tc.options)
			if tc.expErr == "" {
				assert.NoError(err)
			} else {
				assert.EqualError(err, tc.expErr)
			}
		})
	}
}

func TestCheckOptionsRhel8(t *testing.T) {
	type testCase struct {
		it      string
		bp      blueprint.Blueprint
		options distro.ImageOptions
		expErr  string
	}

	// For this test, we just need ImageType instances with a couple of fields
	// set (name, RPMOSTree). However, it's impossible to create one with a
	// given name, because the name is private inside the ImageTypeYAML and
	// meant to only be set by the loader. So we use the real image types,
	// loaded from the YAML files into ImageTypeYAML and create the ImageType
	// itself directly.
	rhel8 := generic.DistroFactory("rhel-8.10")
	imageTypes, err := rhel8.GetArch("x86_64")
	assert.NoError(t, err)

	testCases := map[string]testCase{
		"ami-ok": {
			it:      "ami",
			bp:      blueprint.Blueprint{},
			options: distro.ImageOptions{},
			expErr:  "",
		},
		"ami-installer-error": {
			it: "ami",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Installer: &blueprint.InstallerCustomization{
						Unattended: true,
					},
				},
			},
			expErr: "installer customizations are not supported for \"ami\"",
		},
		"ami-ostree-error": {
			it: "ami",
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			// TODO: this should be an error
		},
		"ostree-installer-requires-ostree-url": {
			it:     "edge-installer",
			expErr: "boot ISO image type \"edge-installer\" requires specifying a URL from which to retrieve the OSTree commit",
		},
		"ostree-disk-supported": {
			it: "edge-raw-image",
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
		"ostree-disk-not-supported": {
			it: "edge-raw-image",
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
		"edge-simplified-requires-install-device": {
			it: "edge-simplified-installer",
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
		"edge-simplified-requires-install-device-error": {
			it: "edge-simplified-installer",
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
		"edge-simplified-supported-customizations": {
			it: "edge-simplified-installer",
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
		"edge-simplified-unsupported-customizations": {
			it: "edge-simplified-installer",
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
		"edge-simplified-fdo-requires-manufacturing-url": {
			it: "edge-simplified-installer",
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
		"edge-simplified-fdo-requires-a-diun-option": {
			it: "edge-simplified-installer",
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
		"edge-simplified-fdo-requires-exactly-one-diun-option": {
			it: "edge-simplified-installer",
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

		"edge-installer-supported-customizations": {
			it: "edge-installer",
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
		"edge-installer-unsupported-customizations": {
			it: "edge-installer",
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

		"ostree-types-no-oscap": {
			it: "edge-container",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					OpenSCAP: &blueprint.OpenSCAPCustomization{
						ProfileID: "xccdf_org.ssgproject.content_profile_ospp",
					},
				},
			},
			expErr: "OpenSCAP customizations are not supported for ostree types",
		},

		"edge-installer-installer-customizations": {
			it: "edge-installer",
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
		"edge-installer-bad-combinations": {
			it: "edge-installer",
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

		"ostree-disk-requires-ostree-url": {
			it:     "edge-raw-image",
			expErr: "\"edge-raw-image\" images require specifying a URL from which to retrieve the OSTree commit",
		},

		"ostree-no-containers": {
			it: "edge-raw-image",
			bp: blueprint.Blueprint{
				Containers: []blueprint.Container{
					{
						Source: "example.org/containers/test:42",
					},
				},
			},
			expErr: "embedding containers is not supported for edge-raw-image on rhel-8.10",
		},

		"ostree-commit-unsupported-kernel-append": {
			it: "edge-commit",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Kernel: &blueprint.KernelCustomization{
						Append: "debug",
					},
				},
			},
			expErr: "kernel boot parameter customizations are not supported for ostree types",
		},

		"ostree-mountpoints-not-supported": {
			it: "edge-commit",
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

		"ostree-partitioning-not-supported": {
			it: "edge-commit",
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

		"oscap-empty-profile": {
			it: "vhd",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					OpenSCAP: &blueprint.OpenSCAPCustomization{
						ProfileID: "",
					},
				},
			},
			expErr: "OpenSCAP profile cannot be empty",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			it, err := imageTypes.GetImageType(tc.it)
			assert.NoError(err)
			genit, ok := it.(*generic.ImageType) // checkOptions() functions require generic.ImageType
			assert.True(ok, "image type %q for distro %q does not appear to be valid", tc.it, rhel8.Name())
			_, err = generic.CheckOptionsRhel8(genit, &tc.bp, tc.options)
			if tc.expErr == "" {
				assert.NoError(err)
			} else {
				assert.EqualError(err, tc.expErr)
			}
		})
	}
}

func TestCheckOptionsRhel9(t *testing.T) {
	type testCase struct {
		it      string
		bp      blueprint.Blueprint
		options distro.ImageOptions
		expErr  string
	}

	// For this test, we just need ImageType instances with a couple of fields
	// set (name, RPMOSTree). However, it's impossible to create one with a
	// given name, because the name is private inside the ImageTypeYAML and
	// meant to only be set by the loader. So we use the real image types,
	// loaded from the YAML files into ImageTypeYAML and create the ImageType
	// itself directly.
	rhel9 := generic.DistroFactory("rhel-9.7")
	imageTypes, err := rhel9.GetArch("x86_64")
	assert.NoError(t, err)

	testCases := map[string]testCase{
		"ami-ok": {
			it:      "ami",
			bp:      blueprint.Blueprint{},
			options: distro.ImageOptions{},
			expErr:  "",
		},
		"ami-installer-error": {
			it: "ami",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Installer: &blueprint.InstallerCustomization{
						Unattended: true,
					},
				},
			},
			expErr: "installer customizations are not supported for \"ami\"",
		},
		"ami-ostree-error": {
			it: "ami",
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			// NOTE: this should be an error
		},
		"ostree-installer-requires-ostree-url": {
			it:     "edge-installer",
			expErr: "boot ISO image type \"edge-installer\" requires specifying a URL from which to retrieve the OSTree commit",
		},
		"ostree-disk-supported": {
			it: "edge-raw-image",
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
		"ostree-disk-not-supported": {
			it: "edge-raw-image",
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
		"edge-simplified-requires-install-device": {
			it: "edge-simplified-installer",
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
		"edge-simplified-requires-install-device-error": {
			it: "edge-simplified-installer",
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
		"edge-simplified-supported-customizations": {
			it: "edge-simplified-installer",
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
		"edge-simplified-unsupported-customizations": {
			it: "edge-simplified-installer",
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
		"edge-simplified-fdo-does-not-require-manufacturing-url": {
			it: "edge-simplified-installer",
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
		"edge-simplified-fdo-requires-a-diun-option": {
			it: "edge-simplified-installer",
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
		"edge-simplified-fdo-requires-exactly-one-diun-option": {
			it: "edge-simplified-installer",
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
		"edge-simplified-ignition": {
			it: "edge-simplified-installer",
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
		"edge-simplified-ignition-no-provisioning-url": {
			it: "edge-simplified-installer",
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
		"edge-simplified-ignition-option-conflict": {
			it: "edge-simplified-installer",
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

		"edge-installer-supported-customizations": {
			it: "edge-installer",
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
		"edge-installer-unsupported-customizations": {
			it: "edge-installer",
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

		"ostree-types-no-oscap": {
			it: "edge-container",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					OpenSCAP: &blueprint.OpenSCAPCustomization{
						ProfileID: "xccdf_org.ssgproject.content_profile_ospp",
					},
				},
			},
			expErr: "OpenSCAP customizations are not supported for ostree types",
		},

		"oscap-empty-profile": {
			it: "vhd",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					OpenSCAP: &blueprint.OpenSCAPCustomization{
						ProfileID: "",
					},
				},
			},
			expErr: "OpenSCAP profile cannot be empty",
		},

		"edge-installer-installer-customizations": {
			it: "edge-installer",
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
		"edge-installer-bad-combinations": {
			it: "edge-installer",
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

		"ostree-disk-unsupported-containers": {
			it: "edge-ami",
			bp: blueprint.Blueprint{
				Containers: []blueprint.Container{
					{
						Source: "example.org/containers/test:42",
					},
				},
			},
			expErr: "embedding containers is not supported for edge-ami on rhel-9.7",
		},

		"ostree-commit-unsupported-kernel-append": {
			it: "edge-commit",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Kernel: &blueprint.KernelCustomization{
						Append: "debug",
					},
				},
			},
			expErr: "kernel boot parameter customizations are not supported for ostree types",
		},

		"ostree-disk-requires-ostree-url": {
			it:     "edge-vsphere",
			expErr: "\"edge-vsphere\" images require specifying a URL from which to retrieve the OSTree commit",
		},
		"ostree-disk2-requires-ostree-url": {
			it:     "edge-ami",
			expErr: "\"edge-ami\" images require specifying a URL from which to retrieve the OSTree commit",
		},

		"ostree-mountpoints-not-supported": {
			it: "edge-commit",
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

		"ostree-partitioning-not-supported": {
			it: "edge-commit",
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

		"ostree-disk-mountpoints-supported": {
			it: "edge-vsphere",
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

		"ostree-disk-partitioning-unsupported": {
			it: "edge-vsphere",
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

		"cvm-kernel-unsupported": {
			it: "azure-cvm",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Kernel: &blueprint.KernelCustomization{
						Name: "kernel-rt",
					},
				},
			},
			expErr: "kernel customizations are not supported for \"azure-cvm\"",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			it, err := imageTypes.GetImageType(tc.it)
			assert.NoError(err)
			genit, ok := it.(*generic.ImageType) // checkOptions() functions require generic.ImageType
			assert.True(ok, "image type %q for distro %q does not appear to be valid", tc.it, rhel9.Name())
			_, err = generic.CheckOptionsRhel9(genit, &tc.bp, tc.options)
			if tc.expErr == "" {
				assert.NoError(err)
			} else {
				assert.EqualError(err, tc.expErr)
			}
		})
	}
}

func TestCheckOptionsRhel10(t *testing.T) {
	type testCase struct {
		it      string
		bp      blueprint.Blueprint
		options distro.ImageOptions
		expErr  string
	}

	// For this test, we just need ImageType instances with a couple of fields
	// set (name, RPMOSTree). However, it's impossible to create one with a
	// given name, because the name is private inside the ImageTypeYAML and
	// meant to only be set by the loader. So we use the real image types,
	// loaded from the YAML files into ImageTypeYAML and create the ImageType
	// itself directly.
	rhel10 := generic.DistroFactory("rhel-10.0")
	imageTypes, err := rhel10.GetArch("x86_64")
	assert.NoError(t, err)

	testCases := map[string]testCase{
		"ami-ok": {
			it:      "ami",
			bp:      blueprint.Blueprint{},
			options: distro.ImageOptions{},
			expErr:  "",
		},
		"ami-installer-error": {
			it: "ami",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Installer: &blueprint.InstallerCustomization{
						Unattended: true,
					},
				},
			},
			expErr: "installer customizations are not supported for \"ami\"",
		},
		"ami-ostree-error": {
			it: "ami",
			options: distro.ImageOptions{
				OSTree: &ostree.ImageOptions{
					URL: "https://example.org/repo",
				},
			},
			// NOTE: this should be an error
		},

		"oscap-empty-profile": {
			it: "vhd",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					OpenSCAP: &blueprint.OpenSCAPCustomization{
						ProfileID: "",
					},
				},
			},
			expErr: "OpenSCAP profile cannot be empty",
		},

		"cvm-kernel-unsupported": {
			it: "azure-cvm",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Kernel: &blueprint.KernelCustomization{
						Name: "kernel-rt",
					},
				},
			},
			// NOTE: this should be an error
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			it, err := imageTypes.GetImageType(tc.it)
			assert.NoError(err)
			genit, ok := it.(*generic.ImageType) // checkOptions() functions require generic.ImageType
			assert.True(ok, "image type %q for distro %q does not appear to be valid", tc.it, rhel10.Name())
			_, err = generic.CheckOptionsRhel10(genit, &tc.bp, tc.options)
			if tc.expErr == "" {
				assert.NoError(err)
			} else {
				assert.EqualError(err, tc.expErr)
			}
		})
	}
}

func TestCheckOptionsRhel7(t *testing.T) {
	type testCase struct {
		it      string
		bp      blueprint.Blueprint
		options distro.ImageOptions
		expErr  string
	}

	// For this test, we just need ImageType instances with a couple of fields
	// set (name, RPMOSTree). However, it's impossible to create one with a
	// given name, because the name is private inside the ImageTypeYAML and
	// meant to only be set by the loader. So we use the real image types,
	// loaded from the YAML files into ImageTypeYAML and create the ImageType
	// itself directly.
	rhel7 := generic.DistroFactory("rhel-7.9")
	imageTypes, err := rhel7.GetArch("x86_64")
	assert.NoError(t, err)

	testCases := map[string]testCase{
		"ok": {
			it:      "qcow2",
			bp:      blueprint.Blueprint{},
			options: distro.ImageOptions{},
			expErr:  "",
		},

		"no-containers": {
			it: "azure-rhui",
			bp: blueprint.Blueprint{
				Containers: []blueprint.Container{
					{
						Name: "example.org/containers/some-kind-of-image:100",
					},
				},
			},
			expErr: "embedding containers is not supported for azure-rhui on rhel-7.9",
		},

		"oscap-empty-profile": {
			it: "qcow2",
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
			it, err := imageTypes.GetImageType(tc.it)
			assert.NoError(err)
			genit, ok := it.(*generic.ImageType) // checkOptions() functions require generic.ImageType
			assert.True(ok, "image type %q for distro %q does not appear to be valid", tc.it, rhel7.Name())
			_, err = generic.CheckOptionsRhel7(genit, &tc.bp, tc.options)
			if tc.expErr == "" {
				assert.NoError(err)
			} else {
				assert.EqualError(err, tc.expErr)
			}
		})
	}
}
