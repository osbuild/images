package generic_test

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/blueprint/pkg/blueprint"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/disk/partition"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/distro_test_common"
	"github.com/osbuild/images/pkg/distro/generic"
	"github.com/osbuild/images/pkg/ostree"
)

var fedoraFamilyDistros = []distro.Distro{
	generic.DistroFactory("fedora-40"),
	generic.DistroFactory("fedora-41"),
	generic.DistroFactory("fedora-42"),
}

func TestFedoraFilenameFromType(t *testing.T) {
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
			name: "server-ami",
			args: args{"server-ami"},
			want: wantResult{
				filename: "image.raw",
				mimeType: "application/octet-stream",
			},
		},
		{
			name: "server-qcow2",
			args: args{"server-qcow2"},
			want: wantResult{
				filename: "disk.qcow2",
				mimeType: "application/x-qemu-disk",
			},
		},
		{
			name: "server-vagrant-libvirt",
			args: args{"server-vagrant-libvirt"},
			want: wantResult{
				filename: "vagrant-libvirt.box",
				mimeType: "application/x-tar",
			},
		},
		{
			name: "server-vagrant-virtualbox",
			args: args{"server-vagrant-virtualbox"},
			want: wantResult{
				filename: "vagrant-virtualbox.box",
				mimeType: "application/x-tar",
			},
		},
		{
			name: "server-openstack",
			args: args{"server-openstack"},
			want: wantResult{
				filename: "disk.qcow2",
				mimeType: "application/x-qemu-disk",
			},
		},
		{
			name: "server-vhd",
			args: args{"server-vhd"},
			want: wantResult{
				filename: "disk.vhd",
				mimeType: "application/x-vhd",
			},
		},
		{
			name: "server-vmdk",
			args: args{"server-vmdk"},
			want: wantResult{
				filename: "disk.vmdk",
				mimeType: "application/x-vmdk",
			},
		},
		{
			name: "server-ova",
			args: args{"server-ova"},
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
				filename: "image.wsl",
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
			name: "minimal-raw-xz",
			args: args{"minimal-raw-xz"},
			want: wantResult{
				filename: "disk.raw.xz",
				mimeType: "application/xz",
			},
		},
	}
	verTypes := map[string][]testCfg{
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
		"41": {
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
		t.Run(dist.Name(), func(t *testing.T) {
			allTests := append(tests, verTypes[dist.Releasever()]...)
			for _, tt := range allTests {
				t.Run(tt.name, func(t *testing.T) {
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

func TestFedoraImageType_BuildPackages(t *testing.T) {
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
	for _, d := range fedoraFamilyDistros {
		t.Run(d.Name(), func(t *testing.T) {
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
					manifest, _, err := itStruct.Manifest(&blueprint.Blueprint{}, distro.ImageOptions{}, nil, nil)
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

func TestFedoraImageType_Name(t *testing.T) {
	imgMap := []struct {
		arch     string
		imgNames []string
		verTypes map[string][]string
	}{
		{
			arch: "x86_64",
			imgNames: []string{
				"server-ami",
				"minimal-installer",
				"iot-commit",
				"iot-container",
				"iot-installer",
				"iot-qcow2",
				"iot-raw-xz",
				"workstation-live-installer",
				"minimal-raw-xz",
				"minimal-raw-zst",
				"server-oci",
				"server-openstack",
				"server-ova",
				"server-qcow2",
				"server-vhd",
				"server-vmdk",
				"server-vagrant-libvirt",
				"server-vagrant-virtualbox",
				"wsl",
			},
			verTypes: map[string][]string{
				"40": {
					"iot-bootable-container",
					"iot-simplified-installer",
				},
				"41": {
					"iot-bootable-container",
					"iot-simplified-installer",
				},
			},
		},
		{
			arch: "aarch64",
			imgNames: []string{
				"server-ami",
				"minimal-installer",
				"iot-commit",
				"iot-container",
				"iot-installer",
				"iot-qcow2",
				"iot-raw-xz",
				"minimal-raw-xz",
				"minimal-raw-zst",
				"server-oci",
				"server-openstack",
				"server-qcow2",
				"server-vagrant-libvirt",
			},
			verTypes: map[string][]string{
				"40": {
					"iot-bootable-container",
					"iot-simplified-installer",
				},
				"41": {
					"iot-bootable-container",
					"iot-simplified-installer",
				},
			},
		},
	}

	for _, dist := range fedoraFamilyDistros {
		t.Run(dist.Name(), func(t *testing.T) {
			for _, mapping := range imgMap {
				arch, err := dist.GetArch(mapping.arch)
				if assert.NoError(t, err) {
					imgTypes := append(mapping.imgNames, mapping.verTypes[dist.Releasever()]...)
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

func TestFedoraImageTypeAliases(t *testing.T) {
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
		t.Run(dist.Name(), func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
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
func TestFedoraDistro_ManifestError(t *testing.T) {
	// Currently, the only unsupported configuration is OSTree commit types
	// with Kernel boot options
	bp := blueprint.Blueprint{
		Customizations: &blueprint.Customizations{
			Kernel: &blueprint.KernelCustomization{
				Append: "debug",
			},
		},
	}

	for _, fedoraDistro := range fedoraFamilyDistros {
		for _, archName := range fedoraDistro.ListArches() {
			arch, _ := fedoraDistro.GetArch(archName)
			for _, imgTypeName := range arch.ListImageTypes() {
				t.Run(fmt.Sprintf("%s/%s", archName, imgTypeName), func(t *testing.T) {
					imgType, _ := arch.GetImageType(imgTypeName)
					imgOpts := distro.ImageOptions{
						Size: imgType.Size(0),
					}
					_, _, err := imgType.Manifest(&bp, imgOpts, nil, nil)
					if imgTypeName == "iot-commit" || imgTypeName == "iot-container" || imgTypeName == "iot-bootable-container" {
						assert.EqualError(t, err, "kernel boot parameter customizations are not supported for ostree types")
					} else if imgTypeName == "iot-installer" || imgTypeName == "iot-simplified-installer" {
						assert.EqualError(t, err, fmt.Sprintf("boot ISO image type \"%s\" requires specifying a URL from which to retrieve the OSTree commit", imgTypeName))
					} else if imgTypeName == "minimal-installer" {
						assert.EqualError(t, err, fmt.Sprintf(distro.UnsupportedCustomizationError, imgTypeName, "User, Group, FIPS, Installer, Timezone, Locale"))
					} else if imgTypeName == "workstation-live-installer" {
						assert.EqualError(t, err, fmt.Sprintf(distro.NoCustomizationsAllowedError, imgTypeName))
					} else if imgTypeName == "iot-raw-xz" || imgTypeName == "iot-qcow2" {
						assert.EqualError(t, err, fmt.Sprintf(distro.UnsupportedCustomizationError, imgTypeName, "User, Group, Directories, Files, Services, FIPS"))
					} else {
						assert.NoError(t, err)
					}
				})
			}
		}
	}
}

func TestFedoraArchitecture_ListImageTypes(t *testing.T) {
	imgMap := []struct {
		arch     string
		imgNames []string
		verTypes map[string][]string
	}{
		{
			arch: "x86_64",
			imgNames: []string{
				"server-ami",
				"container",
				"minimal-installer",
				"iot-commit",
				"iot-container",
				"iot-installer",
				"iot-qcow2",
				"iot-raw-xz",
				"workstation-live-installer",
				"minimal-raw-xz",
				"minimal-raw-zst",
				"server-oci",
				"server-openstack",
				"server-ova",
				"server-qcow2",
				"server-vhd",
				"server-vmdk",
				"server-vagrant-libvirt",
				"server-vagrant-virtualbox",
				"wsl",
				"iot-bootable-container",
				"iot-simplified-installer",
				"everything-netinst",
			},
		},
		{
			arch: "aarch64",
			imgNames: []string{
				"server-ami",
				"container",
				"minimal-installer",
				"iot-commit",
				"iot-container",
				"iot-installer",
				"iot-qcow2",
				"iot-raw-xz",
				"workstation-live-installer",
				"minimal-raw-xz",
				"minimal-raw-zst",
				"server-oci",
				"server-openstack",
				"server-qcow2",
				"server-vagrant-libvirt",
				"iot-bootable-container",
				"iot-simplified-installer",
				"everything-netinst",
			},
		},
		{
			arch: "ppc64le",
			imgNames: []string{
				"container",
				"server-qcow2",
				"iot-bootable-container",
			},
		},
		{
			arch: "s390x",
			imgNames: []string{
				"container",
				"server-qcow2",
				"iot-bootable-container",
			},
		},
		{
			arch: "riscv64",
			imgNames: []string{
				"container",
				"minimal-raw-xz",
				"minimal-raw-zst",
			},
		},
	}

	for _, dist := range fedoraFamilyDistros {
		t.Run(dist.Name(), func(t *testing.T) {
			for _, mapping := range imgMap {
				arch, err := dist.GetArch(mapping.arch)
				require.NoError(t, err)
				imageTypes := arch.ListImageTypes()

				var expectedImageTypes []string
				expectedImageTypes = append(expectedImageTypes, mapping.imgNames...)
				expectedImageTypes = append(expectedImageTypes, mapping.verTypes[dist.Releasever()]...)

				sort.Strings(expectedImageTypes)
				sort.Strings(imageTypes)
				require.Equal(t, expectedImageTypes, imageTypes, "extra images for arch %v", arch.Name())
			}
		})
	}
}

func TestFedoraFedora_ListArches(t *testing.T) {
	for _, fedoraDistro := range fedoraFamilyDistros {
		t.Run(fedoraDistro.Name(), func(t *testing.T) {
			arches := fedoraDistro.ListArches()
			assert.Equal(t, []string{"aarch64", "ppc64le", "riscv64", "s390x", "x86_64"}, arches)
		})
	}
}

func TestFedoraFedora38_GetArch(t *testing.T) {
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
		t.Run(dist.Name(), func(t *testing.T) {
			for _, a := range arches {
				actualArch, err := dist.GetArch(a.name)
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

func TestFedoraFedora_KernelOption(t *testing.T) {
	for _, fedoraDistro := range fedoraFamilyDistros {
		t.Run(fedoraDistro.Name(), func(t *testing.T) {
			distro_test_common.TestDistro_KernelOption(t, fedoraDistro)
		})
	}
}

func TestFedoraFedora_OSTreeOptions(t *testing.T) {
	for _, fedoraDistro := range fedoraFamilyDistros {
		t.Run(fedoraDistro.Name(), func(t *testing.T) {
			distro_test_common.TestDistro_OSTreeOptions(t, fedoraDistro)
		})
	}
}

func TestFedoraDistro_CustomFileSystemManifestError(t *testing.T) {
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
	for _, fedoraDistro := range fedoraFamilyDistros {
		for _, archName := range fedoraDistro.ListArches() {
			arch, _ := fedoraDistro.GetArch(archName)
			for _, imgTypeName := range arch.ListImageTypes() {
				imgType, _ := arch.GetImageType(imgTypeName)
				_, _, err := imgType.Manifest(&bp, distro.ImageOptions{}, nil, nil)
				if imgTypeName == "iot-commit" || imgTypeName == "iot-container" || imgTypeName == "iot-bootable-container" {
					assert.EqualError(t, err, "Custom mountpoints and partitioning are not supported for ostree types")
				} else if imgTypeName == "iot-raw-xz" || imgTypeName == "iot-qcow2" {
					assert.EqualError(t, err, fmt.Sprintf(distro.UnsupportedCustomizationError, imgTypeName, "User, Group, Directories, Files, Services, FIPS"))
				} else if imgTypeName == "iot-installer" || imgTypeName == "iot-simplified-installer" || imgTypeName == "minimal-installer" {
					continue
				} else if imgTypeName == "workstation-live-installer" {
					assert.EqualError(t, err, fmt.Sprintf(distro.NoCustomizationsAllowedError, imgTypeName))
				} else {
					assert.EqualError(t, err, "The following custom mountpoints are not supported [\"/etc\"]")
				}
			}
		}
	}
}

func TestFedoraDistro_TestRootMountPoint(t *testing.T) {
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
	for _, fedoraDistro := range fedoraFamilyDistros {
		for _, archName := range fedoraDistro.ListArches() {
			arch, _ := fedoraDistro.GetArch(archName)
			for _, imgTypeName := range arch.ListImageTypes() {
				imgType, _ := arch.GetImageType(imgTypeName)
				_, _, err := imgType.Manifest(&bp, distro.ImageOptions{}, nil, nil)
				if imgTypeName == "iot-commit" || imgTypeName == "iot-container" || imgTypeName == "iot-bootable-container" {
					assert.EqualError(t, err, "Custom mountpoints and partitioning are not supported for ostree types")
				} else if imgTypeName == "iot-raw-xz" || imgTypeName == "iot-qcow2" {
					assert.EqualError(t, err, fmt.Sprintf(distro.UnsupportedCustomizationError, imgTypeName, "User, Group, Directories, Files, Services, FIPS"))
				} else if imgTypeName == "iot-installer" || imgTypeName == "iot-simplified-installer" || imgTypeName == "minimal-installer" {
					continue
				} else if imgTypeName == "workstation-live-installer" {
					assert.EqualError(t, err, fmt.Sprintf(distro.NoCustomizationsAllowedError, imgTypeName))
				} else {
					assert.NoError(t, err)
				}
			}
		}
	}
}

func TestFedoraDistro_CustomFileSystemSubDirectories(t *testing.T) {
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
	for _, fedoraDistro := range fedoraFamilyDistros {
		for _, archName := range fedoraDistro.ListArches() {
			arch, _ := fedoraDistro.GetArch(archName)
			for _, imgTypeName := range arch.ListImageTypes() {
				imgType, _ := arch.GetImageType(imgTypeName)
				_, _, err := imgType.Manifest(&bp, distro.ImageOptions{}, nil, nil)
				if strings.HasPrefix(imgTypeName, "iot-") || imgTypeName == "minimal-installer" {
					continue
				} else if imgTypeName == "workstation-live-installer" {
					assert.EqualError(t, err, fmt.Sprintf(distro.NoCustomizationsAllowedError, imgTypeName))
				} else {
					assert.NoError(t, err)
				}
			}
		}
	}
}

func TestFedoraDistro_MountpointsWithArbitraryDepthAllowed(t *testing.T) {
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
	for _, fedoraDistro := range fedoraFamilyDistros {
		for _, archName := range fedoraDistro.ListArches() {
			arch, _ := fedoraDistro.GetArch(archName)
			for _, imgTypeName := range arch.ListImageTypes() {
				imgType, _ := arch.GetImageType(imgTypeName)
				_, _, err := imgType.Manifest(&bp, distro.ImageOptions{}, nil, nil)
				if strings.HasPrefix(imgTypeName, "iot-") || imgTypeName == "minimal-installer" {
					continue
				} else if imgTypeName == "workstation-live-installer" {
					assert.EqualError(t, err, fmt.Sprintf(distro.NoCustomizationsAllowedError, imgTypeName))
				} else {
					assert.NoError(t, err)
				}
			}
		}
	}
}

func TestFedoraDistro_DirtyMountpointsNotAllowed(t *testing.T) {
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
	for _, fedoraDistro := range fedoraFamilyDistros {
		for _, archName := range fedoraDistro.ListArches() {
			arch, _ := fedoraDistro.GetArch(archName)
			for _, imgTypeName := range arch.ListImageTypes() {
				imgType, _ := arch.GetImageType(imgTypeName)
				_, _, err := imgType.Manifest(&bp, distro.ImageOptions{}, nil, nil)
				if strings.HasPrefix(imgTypeName, "iot-") || imgTypeName == "minimal-installer" {
					continue
				} else if imgTypeName == "workstation-live-installer" {
					assert.EqualError(t, err, fmt.Sprintf(distro.NoCustomizationsAllowedError, imgTypeName))
				} else {
					assert.EqualError(t, err, "The following custom mountpoints are not supported [\"//\" \"/var//\" \"/var//log/audit/\"]")
				}
			}
		}
	}
}

func TestFedoraDistro_CustomUsrPartitionNotLargeEnough(t *testing.T) {
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
	for _, fedoraDistro := range fedoraFamilyDistros {
		for _, archName := range fedoraDistro.ListArches() {
			arch, _ := fedoraDistro.GetArch(archName)
			for _, imgTypeName := range arch.ListImageTypes() {
				imgType, _ := arch.GetImageType(imgTypeName)
				_, _, err := imgType.Manifest(&bp, distro.ImageOptions{}, nil, nil)
				if imgTypeName == "iot-commit" || imgTypeName == "iot-container" || imgTypeName == "iot-bootable-container" {
					assert.EqualError(t, err, "Custom mountpoints and partitioning are not supported for ostree types")
				} else if imgTypeName == "iot-raw-xz" || imgTypeName == "iot-qcow2" {
					assert.EqualError(t, err, fmt.Sprintf(distro.UnsupportedCustomizationError, imgTypeName, "User, Group, Directories, Files, Services, FIPS"))
				} else if imgTypeName == "iot-installer" || imgTypeName == "iot-simplified-installer" || imgTypeName == "minimal-installer" {
					continue
				} else if imgTypeName == "workstation-live-installer" {
					assert.EqualError(t, err, fmt.Sprintf(distro.NoCustomizationsAllowedError, imgTypeName))
				} else {
					assert.NoError(t, err)
				}
			}
		}
	}
}

func TestFedoraDistro_PartitioningConflict(t *testing.T) {
	bp := blueprint.Blueprint{
		Customizations: &blueprint.Customizations{
			Filesystem: []blueprint.FilesystemCustomization{
				{
					MinSize:    1024,
					Mountpoint: "/",
				},
			},
			Disk: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						MinSize: 19,
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							FSType:     "ext4",
							Mountpoint: "/home",
						},
					},
				},
			},
		},
	}
	for _, fedoraDistro := range fedoraFamilyDistros {
		for _, archName := range fedoraDistro.ListArches() {
			arch, _ := fedoraDistro.GetArch(archName)
			for _, imgTypeName := range arch.ListImageTypes() {
				imgType, _ := arch.GetImageType(imgTypeName)
				_, _, err := imgType.Manifest(&bp, distro.ImageOptions{}, nil, nil)
				if imgTypeName == "iot-commit" || imgTypeName == "iot-container" || imgTypeName == "iot-bootable-container" {
					assert.EqualError(t, err, "Custom mountpoints and partitioning are not supported for ostree types")
				} else if imgTypeName == "iot-raw-xz" || imgTypeName == "iot-qcow2" {
					assert.EqualError(t, err, fmt.Sprintf(distro.UnsupportedCustomizationError, imgTypeName, "User, Group, Directories, Files, Services, FIPS"))
				} else if imgTypeName == "iot-installer" || imgTypeName == "iot-simplified-installer" || imgTypeName == "minimal-installer" {
					continue
				} else if imgTypeName == "workstation-live-installer" {
					assert.EqualError(t, err, fmt.Sprintf(distro.NoCustomizationsAllowedError, imgTypeName))
				} else {
					assert.EqualError(t, err, "partitioning customizations cannot be used with custom filesystems (mountpoints)")
				}
			}
		}
	}

}

func TestFedoraDistroFactory(t *testing.T) {
	type testCase struct {
		strID    string
		expected distro.Distro
	}

	testCases := []testCase{
		{
			strID:    "fedora-40",
			expected: generic.DistroFactory("fedora-40"),
		},
		{
			strID:    "fedora-40.1",
			expected: nil,
		},
		{
			strID:    "fedora",
			expected: nil,
		},
		{
			strID:    "fedora-043",
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.strID, func(t *testing.T) {
			d := generic.DistroFactory(tc.strID)
			if tc.expected == nil {
				assert.Nil(t, d)
			} else {
				assert.NotNil(t, d)
				assert.Equal(t, tc.expected.Name(), d.Name())
			}
		})
	}
}

func TestFedoraDistro_DiskCustomizationRunsValidateLayoutConstraints(t *testing.T) {
	bp := blueprint.Blueprint{
		Customizations: &blueprint.Customizations{
			Disk: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type:            "lvm",
						VGCustomization: blueprint.VGCustomization{},
					},
					{
						Type:            "lvm",
						VGCustomization: blueprint.VGCustomization{},
					},
				},
			},
		},
	}

	for _, fedoraDistro := range fedoraFamilyDistros {
		for _, archName := range fedoraDistro.ListArches() {
			arch, err := fedoraDistro.GetArch(archName)
			assert.NoError(t, err)

			// XXX: enable once we support qcow2 on riscv64
			if arch.Name() == "riscv64" {
				continue
			}

			imgType, err := arch.GetImageType("server-qcow2")
			assert.NoError(t, err, archName)
			t.Run(fmt.Sprintf("%s/%s", archName, imgType.Name()), func(t *testing.T) {
				imgType, _ := arch.GetImageType(imgType.Name())
				imgOpts := distro.ImageOptions{
					Size: imgType.Size(0),
				}
				_, _, err := imgType.Manifest(&bp, imgOpts, nil, nil)
				assert.EqualError(t, err, "multiple LVM volume groups are not yet supported")
			})
		}
	}
}

func TestFedoraESP(t *testing.T) {
	distro_test_common.TestESP(t, fedoraFamilyDistros, func(it distro.ImageType) (*disk.PartitionTable, error) {
		return generic.GetPartitionTable(it)
	})
}

func TestFedoraDistroBootstrapRef(t *testing.T) {
	for _, fedoraDistro := range fedoraFamilyDistros {
		for _, archName := range fedoraDistro.ListArches() {
			arch, err := fedoraDistro.GetArch(archName)
			require.NoError(t, err)
			for _, imgTypeName := range arch.ListImageTypes() {
				imgType, err := arch.GetImageType(imgTypeName)
				require.NoError(t, err)
				if arch.Name() == "riscv64" {
					require.Equal(t, "ghcr.io/mvo5/fedora-buildroot:"+fedoraDistro.OsVersion(), generic.BootstrapContainerFor(imgType))
				} else {
					require.Equal(t, "registry.fedoraproject.org/fedora-toolbox:"+fedoraDistro.OsVersion(), generic.BootstrapContainerFor(imgType))
				}
			}
		}
	}
}

func TestFedoraDistro_PartioningModeConstraints(t *testing.T) {
	for _, fedoraDistro := range fedoraFamilyDistros {
		for _, archName := range fedoraDistro.ListArches() {
			arch, err := fedoraDistro.GetArch(archName)
			assert.NoError(t, err)

			for _, imgTypeName := range arch.ListImageTypes() {
				bp := blueprint.Blueprint{}

				imgType, err := arch.GetImageType(imgTypeName)
				assert.NoError(t, err, imgTypeName)
				if imgType.OSTreeRef() == "" || imgType.PartitionType() == disk.PT_NONE {
					continue
				}

				t.Run(fmt.Sprintf("%s/%s", archName, imgTypeName), func(t *testing.T) {
					imgType, _ := arch.GetImageType(imgType.Name())
					imgOpts := distro.ImageOptions{
						PartitioningMode: partition.RawPartitioningMode,
						OSTree: &ostree.ImageOptions{
							URL: "http://example.com/ostree",
						},
					}
					if imgType.Name() == "iot-simplified-installer" {
						bp.Customizations = &blueprint.Customizations{
							InstallationDevice: "/dev/foo",
						}
					}
					_, _, err := imgType.Manifest(&bp, imgOpts, nil, nil)
					assert.ErrorContains(t, err, "partitioning mode raw not supported for")
				})
			}
		}
	}
}
