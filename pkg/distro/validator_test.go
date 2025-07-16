package distro_test

import (
	"testing"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/platform"
	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/stretchr/testify/assert"
)

type TestImageType struct {
	name                    string
	supportedCustomizations []string
	requiredCustomizations  []string
}

func (t *TestImageType) Name() string {
	return t.name
}

func (t *TestImageType) Arch() distro.Arch {
	return nil
}

func (t *TestImageType) Filename() string {
	return ""
}

func (t *TestImageType) MIMEType() string {
	return ""
}

func (t *TestImageType) OSTreeRef() string {
	return ""
}

func (t *TestImageType) Size(size uint64) uint64 {
	return 0
}

func (t *TestImageType) PartitionType() disk.PartitionTableType {
	return 0
}

func (t *TestImageType) BootMode() platform.BootMode {
	return platform.BOOT_HYBRID
}

func (t *TestImageType) BuildPipelines() []string {
	return distro.BuildPipelinesFallback()
}

func (t *TestImageType) PayloadPipelines() []string {
	return distro.PayloadPipelinesFallback()
}

func (t *TestImageType) PayloadPackageSets() []string {
	return nil
}

func (t *TestImageType) PackageSetsChains() map[string][]string {
	return nil
}

func (t *TestImageType) Exports() []string {
	return distro.ExportsFallback()
}

func (t *TestImageType) BasePartitionTable() (*disk.PartitionTable, error) {
	return nil, nil
}

func (t *TestImageType) ISOLabel() (string, error) {
	return "", nil
}

func (t *TestImageType) Manifest(b *blueprint.Blueprint, options distro.ImageOptions, repos []rpmmd.RepoConfig, seed *int64) (*manifest.Manifest, []string, error) {
	return nil, nil, nil
}

func (t *TestImageType) SupportedCustomizations() []string {
	return t.supportedCustomizations
}

func (t *TestImageType) RequiredCustomizations() []string {
	return t.requiredCustomizations
}

func fullBlueprint() blueprint.Blueprint {
	return blueprint.Blueprint{
		Packages: []blueprint.Package{
			{
				Name: "package",
			},
		},
		Modules: []blueprint.Package{
			{
				Name: "module",
			},
		},
		Groups: []blueprint.Group{
			{
				Name: "group",
			},
		},
		Containers: []blueprint.Container{
			{
				Source: "example.com/containers/test",
			},
		},
		Customizations: &blueprint.Customizations{
			Hostname: common.ToPtr("myhost"),
			Kernel: &blueprint.KernelCustomization{
				Name:   "mykernel",
				Append: "option=value",
			},
			User: []blueprint.UserCustomization{
				{
					Name:        "petris",
					Description: common.ToPtr("I am Petris"),
					Password:    common.ToPtr("terrible password"),
					Key:         common.ToPtr("ssh-key"),
					Home:        common.ToPtr("/home/petros"),
					Shell:       common.ToPtr("/bin/ksh"),
					Groups:      []string{"wheelie"},
					UID:         common.ToPtr(1042),
					GID:         common.ToPtr(1013),
				},
			},
			Group: []blueprint.GroupCustomization{
				{
					Name: "wheelie",
					GID:  common.ToPtr(9901),
				},
			},
			Timezone: &blueprint.TimezoneCustomization{
				Timezone:   common.ToPtr("Australia/Adelaide"),
				NTPServers: []string{"ntp.example.com"},
			},
			Locale: &blueprint.LocaleCustomization{
				Languages: []string{"en_GB.UTF-8", "el_CY.UTF-8"},
				Keyboard:  common.ToPtr("uk"),
			},
			Firewall: &blueprint.FirewallCustomization{
				Ports: []string{"1337:tcp", "1337:udp"},
				Services: &blueprint.FirewallServicesCustomization{
					Enabled:  []string{"leet.service"},
					Disabled: []string{"noob.service"},
				},
				Zones: []blueprint.FirewallZoneCustomization{
					{
						Name:    common.ToPtr("new-zone"),
						Sources: []string{"192.0.42.0/8"},
					},
				},
			},
			Services: &blueprint.ServicesCustomization{
				Enabled:  []string{"leet.service"},
				Disabled: []string{"noob.service", "bad.service"},
				Masked:   []string{"never.service"},
			},
			Filesystem: []blueprint.FilesystemCustomization{
				{
					Mountpoint: "/mnt/stuff",
					MinSize:    100,
				},
			},
			InstallationDevice: "/dev/full",
			FDO: &blueprint.FDOCustomization{
				ManufacturingServerURL:  "fdo.example.com",
				DiunPubKeyInsecure:      "insecure",
				DiunPubKeyHash:          "ffffaaaa123",
				DiunPubKeyRootCerts:     "root-cert-key",
				DiMfgStringTypeMacIface: "--",
			},
			OpenSCAP: &blueprint.OpenSCAPCustomization{
				DataStream: "/usr/share/xml/scap/ssg/content/ssg-fedora-ds.xml",
				ProfileID:  "pci-dss",
				Tailoring: &blueprint.OpenSCAPTailoringCustomizations{
					Selected:   []string{"bind_crypto_policy"},
					Unselected: []string{"rpm_verify_permissions"},
				},
			},
			Ignition: &blueprint.IgnitionCustomization{
				Embedded: &blueprint.EmbeddedIgnitionCustomization{
					Config: "c29tZSBraW5kIG9mIGNvbmZpZwo=",
				},
				FirstBoot: &blueprint.FirstBootIgnitionCustomization{
					ProvisioningURL: "ignition.example.org",
				},
			},
			Directories: []blueprint.DirectoryCustomization{
				{
					Path:          "/etc/path/to/mydir",
					User:          1000,
					Group:         1001,
					Mode:          "700",
					EnsureParents: true,
				},
			},
			Files: []blueprint.FileCustomization{
				{
					Path:  "/etc/path/to/mydir",
					User:  1000,
					Group: 1001,
					Mode:  "700",
					Data:  "SEVMUCEgIEknbSB0cmFwcGVkIGluIGEgdGVzdCEhCg==",
				},
			},
			Repositories: []blueprint.RepositoryCustomization{
				{
					Id:             "baseappstream",
					BaseURLs:       []string{"https://base.repo.example.org"},
					GPGKeys:        []string{"KEY!!!"},
					Metalink:       "https://meta.repo.example.org",
					Mirrorlist:     "https://mirrors.repo.example.org",
					Name:           "baseappstream",
					Priority:       common.ToPtr(3),
					Enabled:        common.ToPtr(true),
					GPGCheck:       common.ToPtr(true),
					RepoGPGCheck:   common.ToPtr(true),
					SSLVerify:      common.ToPtr(true),
					ModuleHotfixes: common.ToPtr(false),
					Filename:       "baseappstream.repo",
				},
			},
			FIPS: common.ToPtr(false),
			ContainersStorage: &blueprint.ContainerStorageCustomization{
				StoragePath: common.ToPtr("/usr/share/my-containers"),
			},
			Installer: &blueprint.InstallerCustomization{
				Unattended:   true,
				SudoNopasswd: []string{"%wheelie"},
			},
		},
		Distro:  "fedora-99",
		Minimal: true,
	}
}

func allOptionStrings() []string {
	return []string{
		"Containers",
		"Customizations.ContainersStorage.StoragePath",
		"Customizations.Directories.EnsureParents",
		"Customizations.Directories.Group",
		"Customizations.Directories.Mode",
		"Customizations.Directories.Path",
		"Customizations.Directories.User",
		"Customizations.FDO.DiMfgStringTypeMacIface",
		"Customizations.FDO.DiunPubKeyHash",
		"Customizations.FDO.DiunPubKeyInsecure",
		"Customizations.FDO.DiunPubKeyRootCerts",
		"Customizations.FDO.ManufacturingServerURL",
		"Customizations.Files.Data",
		"Customizations.Files.Group",
		"Customizations.Files.Mode",
		"Customizations.Files.Path",
		"Customizations.Files.User",
		"Customizations.Filesystem.MinSize",
		"Customizations.Filesystem.Mountpoint",
		"Customizations.FIPS",
		"Customizations.Firewall.Ports",
		"Customizations.Firewall.Services",
		"Customizations.Firewall.Zones",
		"Customizations.Group.GID",
		"Customizations.Group.Name",
		"Customizations.Hostname",
		"Customizations.Ignition.Embedded.Config",
		"Customizations.Ignition.FirstBoot.ProvisioningURL",
		"Customizations.InstallationDevice",
		"Customizations.Installer.Unattended",
		"Customizations.Installer.SudoNopasswd",
		"Customizations.Kernel.Append",
		"Customizations.Kernel.Name",
		"Customizations.Locale.Keyboard",
		"Customizations.Locale.Languages",
		"Customizations.OpenSCAP.DataStream",
		"Customizations.OpenSCAP.ProfileID",
		"Customizations.OpenSCAP.ProfileID",
		"Customizations.OpenSCAP.Tailoring.Selected",
		"Customizations.OpenSCAP.Tailoring.Unselected",
		"Customizations.Repositories",
		"Customizations.Repositories.BaseURLs",
		"Customizations.Repositories.Enabled",
		"Customizations.Repositories.Filename",
		"Customizations.Repositories.GPGCheck",
		"Customizations.Repositories.GPGKeys",
		"Customizations.Repositories.Id",
		"Customizations.Repositories.Metalink",
		"Customizations.Repositories.Mirrorlist",
		"Customizations.Repositories.ModuleHotfixes",
		"Customizations.Repositories.Name",
		"Customizations.Repositories.Priority",
		"Customizations.Repositories.RepoGPGCheck",
		"Customizations.Repositories.SSLVerify",
		"Customizations.Services.Disabled",
		"Customizations.Services.Enabled",
		"Customizations.Services.Masked",
		"Customizations.SSHKey",
		"Customizations.SSHKey.Key",
		"Customizations.SSHKey.User",
		"Customizations.Timezone",
		"Customizations.User.Description",
		"Customizations.User.GID",
		"Customizations.User.Groups",
		"Customizations.User.Home",
		"Customizations.User.Key",
		"Customizations.User.Name",
		"Customizations.User.Password",
		"Customizations.User.Shell",
		"Customizations.User.UID",
		"Distro",
		"Groups",
		"Minimal",
		"Modules",
		"Packages",
	}
}

func TestValidateConfig(t *testing.T) {
	type testCase struct {
		supported []string
		required  []string
		bp        blueprint.Blueprint
		err       string
	}

	testCases := map[string]testCase{
		"simple": {
			// Support some options and set them
			supported: []string{
				"packages",
				"customizations",
				"customizations.kernel",
				"customizations.timezone",
				"customizations.openscap.profile_id",
				"customizations.locale.keyboard",
			},
			required: []string{"packages"},
			bp: blueprint.Blueprint{
				Packages: []blueprint.Package{
					{Name: "vim"},
				},
				Customizations: &blueprint.Customizations{
					Kernel: &blueprint.KernelCustomization{
						Name: "kernol",
					},
					Locale: &blueprint.LocaleCustomization{
						Keyboard: common.ToPtr("us"),
					},
				},
			},
		},
		"full-array-supported": {
			supported: []string{
				"customizations.user",
			},
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					User: []blueprint.UserCustomization{
						{
							Name: "mario",
							Key:  common.ToPtr("ssh-key"),
						},
						{
							Name: "green-mario",
							Key:  common.ToPtr("ssh-key"),
						},
					},
				},
			},
		},
		"nothing-supported": {
			// Don't support anything and add Packages
			bp: blueprint.Blueprint{
				Packages: []blueprint.Package{
					{Name: "vim"},
				},
			},
			err: `Packages: not supported by image type`,
		},
		"category-not-supported": {
			// Support just the Locale under customizations and select Kernel
			supported: []string{
				"customizations.locale",
			},
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Kernel: &blueprint.KernelCustomization{
						Name: "linux",
					},
				},
			},
			err: `Customizations.Kernel: not supported by image type`,
		},
		"leaf-not-supported": {
			// Support only Enabled under Services and select Disabled as well
			supported: []string{
				"customizations.services.enabled",
			},
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Services: &blueprint.ServicesCustomization{
						Enabled:  []string{"good.service"},
						Disabled: []string{"bad.service"},
					},
				},
			},
			err: `Customizations.Services.Disabled: not supported by image type`,
		},
		"leaf-array-not-supported": {
			// Support only Mountpoint under Filesystem (an array) and select MinSize as well
			supported: []string{
				"customizations.filesystem.mountpoint",
			},
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Filesystem: []blueprint.FilesystemCustomization{
						{
							Mountpoint: "/mnt/stuff",
							MinSize:    1,
						},
					},
				},
			},
			err: `Customizations.Filesystem[0].MinSize: not supported by image type`,
		},
		"everything-toplevel": {
			// Support all options and customizations at the top level.
			supported: []string{
				"containers",
				"customizations",
				"distro",
				"groups",
				"minimal",
				"modules",
				"packages",
			},
			required: []string{},
			bp:       fullBlueprint(),
		},
		"everything-supported": {
			// Explicitly support all customizations down to each individual value.
			// Normally these can be enabled by simply enabling all the top
			// level categories, but testing the whole thing is good to make
			// sure all elements are visited in the validator.
			supported: allOptionStrings(),
			bp:        fullBlueprint(),
		},
		"everything-required": {
			// Explicitly require all customizations down to each individual value.
			// Required customizations should also be supported.
			supported: allOptionStrings(),
			required:  allOptionStrings(),
			bp:        fullBlueprint(),
		},
		"missing-customizations-required": {
			supported: []string{"customizations.user"},
			// Require User and don't set anything.
			required: []string{"customizations.user"},
			err:      `Customizations: required by image type`,
		},
		"missing-users-required": {
			// Require User and set a Customization but not User.
			supported: []string{"customizations"},
			required:  []string{"customizations.user"},
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Hostname: common.ToPtr("fail"),
				},
			},
			err: `Customizations.User: required by image type`,
		},
		"required-slice-leaf": {
			// Require the Name under User and set it only for one of the two.
			supported: []string{"customizations.user"},
			required:  []string{"customizations.user.name"},
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					User: []blueprint.UserCustomization{
						{
							Name: "user-with-name",
							Key:  common.ToPtr("ssh-key"),
						},
						{
							Key: common.ToPtr("ssh-key"),
						},
					},
				},
			},
			err: `Customizations.User[1].Name: required by image type`,
		},
	}

	for name := range testCases {
		tc := testCases[name]
		t.Run(name, func(t *testing.T) {
			testImage := &TestImageType{
				name:                    name,
				supportedCustomizations: tc.supported,
				requiredCustomizations:  tc.required,
			}

			err := distro.ValidateConfig(testImage, tc.bp, distro.ImageOptions{})
			if tc.err == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.err)
			}
		},
		)
	}
}
