package fedora_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/distro_test_common"
	"github.com/osbuild/images/pkg/distro/fedora"
)

type fedoraFamilyDistro struct {
	name   string
	distro distro.Distro
}

var fedoraFamilyDistros = []fedoraFamilyDistro{
	{
		name:   "fedora-39",
		distro: fedora.DistroFactory("fedora-39"),
	},
	{
		name:   "fedora-40",
		distro: fedora.DistroFactory("fedora-40"),
	},
}

func TestFilenameFromType(t *testing.T) {
	type args struct {
		outputFormat string
	}
	type wantResult struct {
		filename string
		mimeType string
		wantErr  bool
	}
	type testCfg struct {
		name string
		args args
		want wantResult
	}
	tests := []testCfg{
		{
			name: "ami",
			args: args{"ami"},
			want: wantResult{
				filename: "image.raw",
				mimeType: "application/octet-stream",
			},
		},
		{
			name: "qcow2",
			args: args{"qcow2"},
			want: wantResult{
				filename: "disk.qcow2",
				mimeType: "application/x-qemu-disk",
			},
		},
		{
			name: "openstack",
			args: args{"openstack"},
			want: wantResult{
				filename: "disk.qcow2",
				mimeType: "application/x-qemu-disk",
			},
		},
		{
			name: "vhd",
			args: args{"vhd"},
			want: wantResult{
				filename: "disk.vhd",
				mimeType: "application/x-vhd",
			},
		},
		{
			name: "vmdk",
			args: args{"vmdk"},
			want: wantResult{
				filename: "disk.vmdk",
				mimeType: "application/x-vmdk",
			},
		},
		{
			name: "ova",
			args: args{"ova"},
			want: wantResult{
				filename: "image.ova",
				mimeType: "application/ovf",
			},
		},
		{
			name: "container",
			args: args{"container"},
			want: wantResult{
				filename: "container.tar",
				mimeType: "application/x-tar",
			},
		},
		{
			name: "wsl",
			args: args{"wsl"},
			want: wantResult{
				filename: "wsl.tar",
				mimeType: "application/x-tar",
			},
		},
		{
			name: "iot-commit",
			args: args{"iot-commit"},
			want: wantResult{
				filename: "commit.tar",
				mimeType: "application/x-tar",
			},
		},
		{ // Alias
			name: "fedora-iot-commit",
			args: args{"fedora-iot-commit"},
			want: wantResult{
				filename: "commit.tar",
				mimeType: "application/x-tar",
			},
		},
		{
			name: "iot-container",
			args: args{"iot-container"},
			want: wantResult{
				filename: "container.tar",
				mimeType: "application/x-tar",
			},
		},
		{ // Alias
			name: "fedora-iot-container",
			args: args{"fedora-iot-container"},
			want: wantResult{
				filename: "container.tar",
				mimeType: "application/x-tar",
			},
		},
		{
			name: "iot-installer",
			args: args{"iot-installer"},
			want: wantResult{
				filename: "installer.iso",
				mimeType: "application/x-iso9660-image",
			},
		},
		{ // Alias
			name: "fedora-iot-installer",
			args: args{"fedora-iot-installer"},
			want: wantResult{
				filename: "installer.iso",
				mimeType: "application/x-iso9660-image",
			},
		},
		{
			name: "live-installer",
			args: args{"live-installer"},
			want: wantResult{
				filename: "live-installer.iso",
				mimeType: "application/x-iso9660-image",
			},
		},
		{
			name: "image-installer",
			args: args{"image-installer"},
			want: wantResult{
				filename: "installer.iso",
				mimeType: "application/x-iso9660-image",
			},
		},
		{ // Alias
			name: "fedora-image-installer",
			args: args{"fedora-image-installer"},
			want: wantResult{
				filename: "installer.iso",
				mimeType: "application/x-iso9660-image",
			},
		},
		{
			name: "invalid-output-type",
			args: args{"foobar"},
			want: wantResult{wantErr: true},
		},
		{
			name: "minimal-raw",
			args: args{"minimal-raw"},
			want: wantResult{
				filename: "disk.raw.xz",
				mimeType: "application/xz",
			},
		},
	}
	verTypes := map[string][]testCfg{
		"38": {
			{
				name: "iot-simplified-installer",
				args: args{"iot-simplified-installer"},
				want: wantResult{
					filename: "simplified-installer.iso",
					mimeType: "application/x-iso9660-image",
				},
			},
		},
		"39": {
			{
				name: "iot-bootable-container",
				args: args{"iot-bootable-container"},
				want: wantResult{
					filename: "iot-bootable-container.tar",
					mimeType: "application/x-tar",
				},
			},
			{
				name: "iot-simplified-installer",
				args: args{"iot-simplified-installer"},
				want: wantResult{
					filename: "simplified-installer.iso",
					mimeType: "application/x-iso9660-image",
				},
			},
		},
		"40": {
			{
				name: "iot-bootable-container",
				args: args{"iot-bootable-container"},
				want: wantResult{
					filename: "iot-bootable-container.tar",
					mimeType: "application/x-tar",
				},
			},
			{
				name: "iot-simplified-installer",
				args: args{"iot-simplified-installer"},
				want: wantResult{
					filename: "simplified-installer.iso",
					mimeType: "application/x-iso9660-image",
				},
			},
		},
	}
	for _, dist := range fedoraFamilyDistros {
		t.Run(dist.distro.Name(), func(t *testing.T) {
			allTests := append(tests, verTypes[dist.distro.Releasever()]...)
			for _, tt := range allTests {
				t.Run(tt.name, func(t *testing.T) {
					dist := dist.distro
					arch, _ := dist.GetArch("x86_64")
					imgType, err := arch.GetImageType(tt.args.outputFormat)
					if tt.want.wantErr {
						require.Error(t, err)
					} else {
						require.NoError(t, err)
						require.NotNil(t, imgType)
						gotFilename := imgType.Filename()
						gotMIMEType := imgType.MIMEType()
						if gotFilename != tt.want.filename {
							t.Errorf("ImageType.Filename()  got = %v, want %v", gotFilename, tt.want.filename)
						}
						if gotMIMEType != tt.want.mimeType {
							t.Errorf("ImageType.MIMEType() got1 = %v, want %v", gotMIMEType, tt.want.mimeType)
						}
					}
				})
			}
		})
	}
}

func TestImageType_BuildPackages(t *testing.T) {
	x8664BuildPackages := []string{
		"dnf",
		"dosfstools",
		"e2fsprogs",
		"policycoreutils",
		"qemu-img",
		"selinux-policy-targeted",
		"systemd",
		"tar",
		"xz",
		"grub2-pc",
	}
	aarch64BuildPackages := []string{
		"dnf",
		"dosfstools",
		"e2fsprogs",
		"policycoreutils",
		"qemu-img",
		"selinux-policy-targeted",
		"systemd",
		"tar",
		"xz",
	}
	buildPackages := map[string][]string{
		"x86_64":  x8664BuildPackages,
		"aarch64": aarch64BuildPackages,
	}
	for _, dist := range fedoraFamilyDistros {
		t.Run(dist.distro.Name(), func(t *testing.T) {
			d := dist.distro
			for _, archLabel := range d.ListArches() {
				archStruct, err := d.GetArch(archLabel)
				if assert.NoErrorf(t, err, "d.GetArch(%v) returned err = %v; expected nil", archLabel, err) {
					continue
				}
				for _, itLabel := range archStruct.ListImageTypes() {
					itStruct, err := archStruct.GetImageType(itLabel)
					if assert.NoErrorf(t, err, "d.GetArch(%v) returned err = %v; expected nil", archLabel, err) {
						continue
					}
					manifest, _, err := itStruct.Manifest(&blueprint.Blueprint{}, distro.ImageOptions{}, nil, 0)
					assert.NoError(t, err)
					buildPkgs := manifest.GetPackageSetChains()["build"]
					assert.NotNil(t, buildPkgs)
					assert.Len(t, buildPkgs, 1)
					assert.ElementsMatch(t, buildPackages[archLabel], buildPkgs[0].Include)
				}
			}
		})
	}
}

func TestImageType_Name(t *testing.T) {
	imgMap := []struct {
		arch     string
		imgNames []string
		verTypes map[string][]string
	}{
		{
			arch: "x86_64",
			imgNames: []string{
				"ami",
				"image-installer",
				"iot-commit",
				"iot-container",
				"iot-installer",
				"iot-qcow2-image",
				"iot-raw-image",
				"live-installer",
				"minimal-raw",
				"oci",
				"openstack",
				"ova",
				"qcow2",
				"vhd",
				"vmdk",
				"wsl",
			},
			verTypes: map[string][]string{
				"38": {"iot-simplified-installer"},
				"39": {
					"iot-bootable-container",
					"iot-simplified-installer",
				},
				"40": {
					"iot-bootable-container",
					"iot-simplified-installer",
				},
			},
		},
		{
			arch: "aarch64",
			imgNames: []string{
				"ami",
				"image-installer",
				"iot-commit",
				"iot-container",
				"iot-installer",
				"iot-qcow2-image",
				"iot-raw-image",
				"minimal-raw",
				"oci",
				"openstack",
				"qcow2",
			},
			verTypes: map[string][]string{
				"38": {"iot-simplified-installer"},
				"39": {
					"iot-bootable-container",
					"iot-simplified-installer",
				},
				"40": {
					"iot-bootable-container",
					"iot-simplified-installer",
				},
			},
		},
	}

	for _, dist := range fedoraFamilyDistros {
		t.Run(dist.distro.Name(), func(t *testing.T) {
			for _, mapping := range imgMap {
				arch, err := dist.distro.GetArch(mapping.arch)
				if assert.NoError(t, err) {
					imgTypes := append(mapping.imgNames, mapping.verTypes[dist.distro.Releasever()]...)
					for _, imgName := range imgTypes {
						imgType, err := arch.GetImageType(imgName)
						if assert.NoError(t, err) {
							assert.Equalf(t, imgName, imgType.Name(), "arch: %s", mapping.arch)
						}
					}
				}
			}
		})
	}
}

func TestImageTypeAliases(t *testing.T) {
	type args struct {
		imageTypeAliases []string
	}
	type wantResult struct {
		imageTypeName string
	}
	tests := []struct {
		name string
		args args
		want wantResult
	}{
		{
			name: "iot-commit aliases",
			args: args{
				imageTypeAliases: []string{"fedora-iot-commit"},
			},
			want: wantResult{
				imageTypeName: "iot-commit",
			},
		},
		{
			name: "iot-container aliases",
			args: args{
				imageTypeAliases: []string{"fedora-iot-container"},
			},
			want: wantResult{
				imageTypeName: "iot-container",
			},
		},
		{
			name: "iot-installer aliases",
			args: args{
				imageTypeAliases: []string{"fedora-iot-installer"},
			},
			want: wantResult{
				imageTypeName: "iot-installer",
			},
		},
	}
	for _, dist := range fedoraFamilyDistros {
		t.Run(dist.name, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					dist := dist.distro
					for _, archName := range dist.ListArches() {
						t.Run(archName, func(t *testing.T) {
							arch, err := dist.GetArch(archName)
							require.Nilf(t, err,
								"failed to get architecture '%s', previously listed as supported for the distro '%s'",
								archName, dist.Name())
							// Test image type aliases only if the aliased image type is supported for the arch
							if _, err = arch.GetImageType(tt.want.imageTypeName); err != nil {
								t.Skipf("aliased image type '%s' is not supported for architecture '%s'",
									tt.want.imageTypeName, archName)
							}
							for _, alias := range tt.args.imageTypeAliases {
								t.Run(fmt.Sprintf("'%s' alias for image type '%s'", alias, tt.want.imageTypeName),
									func(t *testing.T) {
										gotImage, err := arch.GetImageType(alias)
										require.Nilf(t, err, "arch.GetImageType() for image type alias '%s' failed: %v",
											alias, err)
										assert.Equalf(t, tt.want.imageTypeName, gotImage.Name(),
											"got unexpected image type name for alias '%s'. got = %s, want = %s",
											alias, tt.want.imageTypeName, gotImage.Name())
									})
							}
						})
					}
				})
			}
		})
	}
}

// Check that Manifest() function returns an error for unsupported
// configurations.
func TestDistro_ManifestError(t *testing.T) {
	// Currently, the only unsupported configuration is OSTree commit types
	// with Kernel boot options
	bp := blueprint.Blueprint{
		Customizations: &blueprint.Customizations{
			Kernel: &blueprint.KernelCustomization{
				Append: "debug",
			},
		},
	}

	for _, dist := range fedoraFamilyDistros {
		fedoraDistro := dist.distro
		for _, archName := range fedoraDistro.ListArches() {
			arch, _ := fedoraDistro.GetArch(archName)
			for _, imgTypeName := range arch.ListImageTypes() {
				t.Run(fmt.Sprintf("%s/%s", archName, imgTypeName), func(t *testing.T) {
					imgType, _ := arch.GetImageType(imgTypeName)
					imgOpts := distro.ImageOptions{
						Size: imgType.Size(0),
					}
					_, _, err := imgType.Manifest(&bp, imgOpts, nil, 0)
					if imgTypeName == "iot-commit" || imgTypeName == "iot-container" || imgTypeName == "iot-bootable-container" {
						assert.EqualError(t, err, "kernel boot parameter customizations are not supported for ostree types")
					} else if imgTypeName == "iot-installer" || imgTypeName == "iot-simplified-installer" {
						assert.EqualError(t, err, fmt.Sprintf("boot ISO image type \"%s\" requires specifying a URL from which to retrieve the OSTree commit", imgTypeName))
					} else if imgTypeName == "image-installer" {
						assert.EqualError(t, err, fmt.Sprintf(distro.UnsupportedCustomizationError, imgTypeName, "User, Group, FIPS, Installer, Timezone, Locale"))
					} else if imgTypeName == "live-installer" {
						assert.EqualError(t, err, fmt.Sprintf(distro.NoCustomizationsAllowedError, imgTypeName))
					} else if imgTypeName == "iot-raw-image" || imgTypeName == "iot-qcow2-image" {
						assert.EqualError(t, err, fmt.Sprintf(distro.UnsupportedCustomizationError, imgTypeName, "User, Group, Directories, Files, Services, FIPS"))
					} else {
						assert.NoError(t, err)
					}
				})
			}
		}
	}
}

func TestArchitecture_ListImageTypes(t *testing.T) {
	imgMap := []struct {
		arch     string
		imgNames []string
		verTypes map[string][]string
	}{
		{
			arch: "x86_64",
			imgNames: []string{
				"ami",
				"container",
				"image-installer",
				"iot-commit",
				"iot-container",
				"iot-installer",
				"iot-qcow2-image",
				"iot-raw-image",
				"live-installer",
				"minimal-raw",
				"oci",
				"openstack",
				"ova",
				"qcow2",
				"vhd",
				"vmdk",
				"wsl",
			},
			verTypes: map[string][]string{
				"38": {"iot-simplified-installer"},
				"39": {
					"iot-bootable-container",
					"iot-simplified-installer",
				},
				"40": {
					"iot-bootable-container",
					"iot-simplified-installer",
				},
			},
		},
		{
			arch: "aarch64",
			imgNames: []string{
				"ami",
				"container",
				"image-installer",
				"iot-commit",
				"iot-container",
				"iot-installer",
				"iot-qcow2-image",
				"iot-raw-image",
				"live-installer",
				"minimal-raw",
				"oci",
				"openstack",
				"qcow2",
			},
			verTypes: map[string][]string{
				"38": {"iot-simplified-installer"},
				"39": {
					"iot-bootable-container",
					"iot-simplified-installer",
				},
				"40": {
					"iot-bootable-container",
					"iot-simplified-installer",
				},
			},
		},
		{
			arch: "ppc64le",
			imgNames: []string{
				"container",
				"qcow2",
			},
			verTypes: map[string][]string{
				"39": {
					"iot-bootable-container",
				},
				"40": {
					"iot-bootable-container",
				},
			},
		},
		{
			arch: "s390x",
			imgNames: []string{
				"container",
				"qcow2",
			},
			verTypes: map[string][]string{
				"39": {
					"iot-bootable-container",
				},
				"40": {
					"iot-bootable-container",
				},
			},
		},
	}

	for _, dist := range fedoraFamilyDistros {
		t.Run(dist.distro.Name(), func(t *testing.T) {
			for _, mapping := range imgMap {
				arch, err := dist.distro.GetArch(mapping.arch)
				require.NoError(t, err)
				imageTypes := arch.ListImageTypes()

				var expectedImageTypes []string
				expectedImageTypes = append(expectedImageTypes, mapping.imgNames...)
				expectedImageTypes = append(expectedImageTypes, mapping.verTypes[dist.distro.Releasever()]...)

				require.ElementsMatch(t, expectedImageTypes, imageTypes)
			}
		})
	}
}

func TestFedora_ListArches(t *testing.T) {
	for _, dist := range fedoraFamilyDistros {
		fedoraDistro := dist.distro
		t.Run(dist.name, func(t *testing.T) {
			arches := fedoraDistro.ListArches()
			assert.Equal(t, []string{"aarch64", "ppc64le", "s390x", "x86_64"}, arches)
		})
	}
}

func TestFedora38_GetArch(t *testing.T) {
	arches := []struct {
		name                  string
		errorExpected         bool
		errorExpectedInCentos bool
	}{
		{
			name: "x86_64",
		},
		{
			name: "aarch64",
		},
		{
			name: "s390x",
		},
		{
			name: "ppc64le",
		},
		{
			name:          "foo-arch",
			errorExpected: true,
		},
	}

	for _, dist := range fedoraFamilyDistros {
		t.Run(dist.distro.Name(), func(t *testing.T) {
			for _, a := range arches {
				actualArch, err := dist.distro.GetArch(a.name)
				if a.errorExpected {
					assert.Nil(t, actualArch)
					assert.Error(t, err)
				} else {
					assert.Equal(t, a.name, actualArch.Name())
					assert.NoError(t, err)
				}
			}
		})
	}
}

func TestFedora_Name(t *testing.T) {
	for _, dist := range fedoraFamilyDistros {
		fedoraDistro := dist.distro
		t.Run(dist.name, func(t *testing.T) {
			assert.Equal(t, dist.name, fedoraDistro.Name())
		})
	}
}

func TestFedora_KernelOption(t *testing.T) {
	for _, dist := range fedoraFamilyDistros {
		fedoraDistro := dist.distro
		t.Run(dist.name, func(t *testing.T) {
			distro_test_common.TestDistro_KernelOption(t, fedoraDistro)
		})
	}
}

func TestFedora_OSTreeOptions(t *testing.T) {
	for _, dist := range fedoraFamilyDistros {
		fedoraDistro := dist.distro
		t.Run(dist.name, func(t *testing.T) {
			distro_test_common.TestDistro_OSTreeOptions(t, fedoraDistro)
		})
	}
}

func TestDistro_CustomFileSystemManifestError(t *testing.T) {
	bp := blueprint.Blueprint{
		Customizations: &blueprint.Customizations{
			Filesystem: []blueprint.FilesystemCustomization{
				{
					MinSize:    1024,
					Mountpoint: "/etc",
				},
			},
		},
	}
	for _, dist := range fedoraFamilyDistros {
		fedoraDistro := dist.distro
		for _, archName := range fedoraDistro.ListArches() {
			arch, _ := fedoraDistro.GetArch(archName)
			for _, imgTypeName := range arch.ListImageTypes() {
				imgType, _ := arch.GetImageType(imgTypeName)
				_, _, err := imgType.Manifest(&bp, distro.ImageOptions{}, nil, 0)
				if imgTypeName == "iot-commit" || imgTypeName == "iot-container" || imgTypeName == "iot-bootable-container" {
					assert.EqualError(t, err, "Custom mountpoints and partitioning are not supported for ostree types")
				} else if imgTypeName == "iot-raw-image" || imgTypeName == "iot-qcow2-image" {
					assert.EqualError(t, err, fmt.Sprintf(distro.UnsupportedCustomizationError, imgTypeName, "User, Group, Directories, Files, Services, FIPS"))
				} else if imgTypeName == "iot-installer" || imgTypeName == "iot-simplified-installer" || imgTypeName == "image-installer" {
					continue
				} else if imgTypeName == "live-installer" {
					assert.EqualError(t, err, fmt.Sprintf(distro.NoCustomizationsAllowedError, imgTypeName))
				} else {
					assert.EqualError(t, err, "The following errors occurred while setting up custom mountpoints:\npath \"/etc\" is not allowed")
				}
			}
		}
	}
}

func TestDistro_TestRootMountPoint(t *testing.T) {
	bp := blueprint.Blueprint{
		Customizations: &blueprint.Customizations{
			Filesystem: []blueprint.FilesystemCustomization{
				{
					MinSize:    1024,
					Mountpoint: "/",
				},
			},
		},
	}
	for _, dist := range fedoraFamilyDistros {
		fedoraDistro := dist.distro
		for _, archName := range fedoraDistro.ListArches() {
			arch, _ := fedoraDistro.GetArch(archName)
			for _, imgTypeName := range arch.ListImageTypes() {
				imgType, _ := arch.GetImageType(imgTypeName)
				_, _, err := imgType.Manifest(&bp, distro.ImageOptions{}, nil, 0)
				if imgTypeName == "iot-commit" || imgTypeName == "iot-container" || imgTypeName == "iot-bootable-container" {
					assert.EqualError(t, err, "Custom mountpoints and partitioning are not supported for ostree types")
				} else if imgTypeName == "iot-raw-image" || imgTypeName == "iot-qcow2-image" {
					assert.EqualError(t, err, fmt.Sprintf(distro.UnsupportedCustomizationError, imgTypeName, "User, Group, Directories, Files, Services, FIPS"))
				} else if imgTypeName == "iot-installer" || imgTypeName == "iot-simplified-installer" || imgTypeName == "image-installer" {
					continue
				} else if imgTypeName == "live-installer" {
					assert.EqualError(t, err, fmt.Sprintf(distro.NoCustomizationsAllowedError, imgTypeName))
				} else {
					assert.NoError(t, err)
				}
			}
		}
	}
}

func TestDistro_CustomFileSystemSubDirectories(t *testing.T) {
	bp := blueprint.Blueprint{
		Customizations: &blueprint.Customizations{
			Filesystem: []blueprint.FilesystemCustomization{
				{
					MinSize:    1024,
					Mountpoint: "/var/log",
				},
				{
					MinSize:    1024,
					Mountpoint: "/var/log/audit",
				},
			},
		},
	}
	for _, dist := range fedoraFamilyDistros {
		fedoraDistro := dist.distro
		for _, archName := range fedoraDistro.ListArches() {
			arch, _ := fedoraDistro.GetArch(archName)
			for _, imgTypeName := range arch.ListImageTypes() {
				imgType, _ := arch.GetImageType(imgTypeName)
				_, _, err := imgType.Manifest(&bp, distro.ImageOptions{}, nil, 0)
				if strings.HasPrefix(imgTypeName, "iot-") || strings.HasPrefix(imgTypeName, "image-") {
					continue
				} else if imgTypeName == "live-installer" {
					assert.EqualError(t, err, fmt.Sprintf(distro.NoCustomizationsAllowedError, imgTypeName))
				} else {
					assert.NoError(t, err)
				}
			}
		}
	}
}

func TestDistro_MountpointsWithArbitraryDepthAllowed(t *testing.T) {
	bp := blueprint.Blueprint{
		Customizations: &blueprint.Customizations{
			Filesystem: []blueprint.FilesystemCustomization{
				{
					MinSize:    1024,
					Mountpoint: "/var/a",
				},
				{
					MinSize:    1024,
					Mountpoint: "/var/a/b",
				},
				{
					MinSize:    1024,
					Mountpoint: "/var/a/b/c",
				},
				{
					MinSize:    1024,
					Mountpoint: "/var/a/b/c/d",
				},
			},
		},
	}
	for _, dist := range fedoraFamilyDistros {
		fedoraDistro := dist.distro
		for _, archName := range fedoraDistro.ListArches() {
			arch, _ := fedoraDistro.GetArch(archName)
			for _, imgTypeName := range arch.ListImageTypes() {
				imgType, _ := arch.GetImageType(imgTypeName)
				_, _, err := imgType.Manifest(&bp, distro.ImageOptions{}, nil, 0)
				if strings.HasPrefix(imgTypeName, "iot-") || strings.HasPrefix(imgTypeName, "image-") {
					continue
				} else if imgTypeName == "live-installer" {
					assert.EqualError(t, err, fmt.Sprintf(distro.NoCustomizationsAllowedError, imgTypeName))
				} else {
					assert.NoError(t, err)
				}
			}
		}
	}
}

func TestDistro_DirtyMountpointsNotAllowed(t *testing.T) {
	bp := blueprint.Blueprint{
		Customizations: &blueprint.Customizations{
			Filesystem: []blueprint.FilesystemCustomization{
				{
					MinSize:    1024,
					Mountpoint: "//",
				},
				{
					MinSize:    1024,
					Mountpoint: "/var//",
				},
				{
					MinSize:    1024,
					Mountpoint: "/var//log/audit/",
				},
			},
		},
	}
	for _, dist := range fedoraFamilyDistros {
		fedoraDistro := dist.distro
		for _, archName := range fedoraDistro.ListArches() {
			arch, _ := fedoraDistro.GetArch(archName)
			for _, imgTypeName := range arch.ListImageTypes() {
				imgType, _ := arch.GetImageType(imgTypeName)
				_, _, err := imgType.Manifest(&bp, distro.ImageOptions{}, nil, 0)
				if strings.HasPrefix(imgTypeName, "iot-") || strings.HasPrefix(imgTypeName, "image-") {
					continue
				} else if imgTypeName == "live-installer" {
					assert.EqualError(t, err, fmt.Sprintf(distro.NoCustomizationsAllowedError, imgTypeName))
				} else {
					assert.EqualError(t, err, "The following errors occurred while setting up custom mountpoints:\npath \"//\" must be canonical\npath \"/var//\" must be canonical\npath \"/var//log/audit/\" must be canonical")
				}
			}
		}
	}
}

func TestDistro_CustomUsrPartitionNotLargeEnough(t *testing.T) {
	bp := blueprint.Blueprint{
		Customizations: &blueprint.Customizations{
			Filesystem: []blueprint.FilesystemCustomization{
				{
					MinSize:    1024,
					Mountpoint: "/usr",
				},
			},
		},
	}
	for _, dist := range fedoraFamilyDistros {
		fedoraDistro := dist.distro
		for _, archName := range fedoraDistro.ListArches() {
			arch, _ := fedoraDistro.GetArch(archName)
			for _, imgTypeName := range arch.ListImageTypes() {
				imgType, _ := arch.GetImageType(imgTypeName)
				_, _, err := imgType.Manifest(&bp, distro.ImageOptions{}, nil, 0)
				if imgTypeName == "iot-commit" || imgTypeName == "iot-container" || imgTypeName == "iot-bootable-container" {
					assert.EqualError(t, err, "Custom mountpoints and partitioning are not supported for ostree types")
				} else if imgTypeName == "iot-raw-image" || imgTypeName == "iot-qcow2-image" {
					assert.EqualError(t, err, fmt.Sprintf(distro.UnsupportedCustomizationError, imgTypeName, "User, Group, Directories, Files, Services, FIPS"))
				} else if imgTypeName == "iot-installer" || imgTypeName == "iot-simplified-installer" || imgTypeName == "image-installer" {
					continue
				} else if imgTypeName == "live-installer" {
					assert.EqualError(t, err, fmt.Sprintf(distro.NoCustomizationsAllowedError, imgTypeName))
				} else {
					assert.NoError(t, err)
				}
			}
		}
	}
}

func TestDistro_PartitioningConflict(t *testing.T) {
	bp := blueprint.Blueprint{
		Customizations: &blueprint.Customizations{
			Filesystem: []blueprint.FilesystemCustomization{
				{
					MinSize:    1024,
					Mountpoint: "/",
				},
			},
			Partitioning: &blueprint.PartitioningCustomization{
				Plain: &blueprint.PlainFilesystemCustomization{
					Filesystems: []blueprint.FilesystemCustomization{
						{
							MinSize:    19,
							Mountpoint: "/home",
						},
					},
				},
			},
		},
	}
	for _, dist := range fedoraFamilyDistros {
		fedoraDistro := dist.distro
		for _, archName := range fedoraDistro.ListArches() {
			arch, _ := fedoraDistro.GetArch(archName)
			for _, imgTypeName := range arch.ListImageTypes() {
				imgType, _ := arch.GetImageType(imgTypeName)
				_, _, err := imgType.Manifest(&bp, distro.ImageOptions{}, nil, 0)
				if imgTypeName == "iot-commit" || imgTypeName == "iot-container" || imgTypeName == "iot-bootable-container" {
					assert.EqualError(t, err, "Custom mountpoints and partitioning are not supported for ostree types")
				} else if imgTypeName == "iot-raw-image" || imgTypeName == "iot-qcow2-image" {
					assert.EqualError(t, err, fmt.Sprintf(distro.UnsupportedCustomizationError, imgTypeName, "User, Group, Directories, Files, Services, FIPS"))
				} else if imgTypeName == "iot-installer" || imgTypeName == "iot-simplified-installer" || imgTypeName == "image-installer" {
					continue
				} else if imgTypeName == "live-installer" {
					assert.EqualError(t, err, fmt.Sprintf(distro.NoCustomizationsAllowedError, imgTypeName))
				} else {
					assert.EqualError(t, err, "partitioning customizations cannot be used with custom filesystems (mountpoints)")
				}
			}
		}
	}

}

func TestDistroFactory(t *testing.T) {
	type testCase struct {
		strID    string
		expected distro.Distro
	}

	testCases := []testCase{
		{
			strID:    "fedora-38",
			expected: fedora.DistroFactory("fedora-38"),
		},
		{
			strID:    "fedora-38.1",
			expected: nil,
		},
		{
			strID:    "fedora",
			expected: nil,
		},
		{
			strID:    "rhel-9",
			expected: nil,
		},
		{
			strID:    "rhel-8.4",
			expected: nil,
		},
		{
			strID:    "rhel-810",
			expected: nil,
		},
		{
			strID:    "rhel-8.4.1",
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.strID, func(t *testing.T) {
			d := fedora.DistroFactory(tc.strID)
			if tc.expected == nil {
				assert.Nil(t, d)
			} else {
				assert.NotNil(t, d)
				assert.Equal(t, tc.expected.Name(), d.Name())
			}
		})
	}
}
