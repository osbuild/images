package fedora_core_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/fedora_core"
)

type fedoraFamilyDistro struct {
	name   string
	distro distro.Distro
}

var fedoraFamilyDistros = []fedoraFamilyDistro{
	{
		name:   "fedora-core",
		distro: fedora_core.NewF39(),
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
	tests := []struct {
		name string
		args args
		want wantResult
	}{
		{
			name: "disk-raw",
			args: args{"disk-raw"},
			want: wantResult{
				filename: "raw.img",
				mimeType: "application/disk",
			},
		},
		{
			name: "iso-live",
			args: args{"iso-live"},
			want: wantResult{
				filename: "live.iso",
				mimeType: "application/x-iso9660-image",
			},
		},
	}
	for _, dist := range fedoraFamilyDistros {
		t.Run(dist.name, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					dist := dist.distro
					arch, _ := dist.GetArch("x86_64")
					imgType, err := arch.GetImageType(tt.args.outputFormat)
					if (err != nil) != tt.want.wantErr {
						t.Errorf("Arch.GetImageType() error = %v, wantErr %v", err, tt.want.wantErr)
						return
					}
					if !tt.want.wantErr {
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
		t.Run(dist.name, func(t *testing.T) {
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
	}{
		{
			arch: "x86_64",
			imgNames: []string{
				"disk-raw",
			},
		},
		{
			arch: "aarch64",
			imgNames: []string{
				"disk-raw",
			},
		},
	}

	for _, dist := range fedoraFamilyDistros {
		t.Run(dist.name, func(t *testing.T) {
			for _, mapping := range imgMap {
				if mapping.arch == "s390x" {
					continue
				}
				arch, err := dist.distro.GetArch(mapping.arch)
				if assert.NoError(t, err) {
					for _, imgName := range mapping.imgNames {
						if imgName == "iot-commit" {
							continue
						}
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

// Check that Manifest() function returns an error for unsupported
// configurations.
func TestDistro_ManifestError(t *testing.T) {
	// Currently, the only unsupported configuration is OSTree commit types
	// with Kernel boot options
	fedoraDistro := fedora_core.NewF39()
	bp := blueprint.Blueprint{
		Customizations: &blueprint.Customizations{
			Kernel: &blueprint.KernelCustomization{
				Append: "debug",
			},
		},
	}

	for _, archName := range fedoraDistro.ListArches() {
		arch, _ := fedoraDistro.GetArch(archName)
		for _, imgTypeName := range arch.ListImageTypes() {
			imgType, _ := arch.GetImageType(imgTypeName)
			imgOpts := distro.ImageOptions{
				Size: imgType.Size(0),
			}
			_, _, err := imgType.Manifest(&bp, imgOpts, nil, 0)
			assert.NoError(t, err)
		}
	}
}

func TestArchitecture_ListImageTypes(t *testing.T) {
	imgMap := []struct {
		arch                       string
		imgNames                   []string
		fedoraAdditionalImageTypes []string
	}{
		{
			arch: "x86_64",
			imgNames: []string{
				"disk-raw",
				"iso-live",
			},
		},
		{
			arch: "aarch64",
			imgNames: []string{
				"disk-raw",
				"iso-live",
			},
		},
	}

	for _, dist := range fedoraFamilyDistros {
		t.Run(dist.name, func(t *testing.T) {
			for _, mapping := range imgMap {
				arch, err := dist.distro.GetArch(mapping.arch)
				require.NoError(t, err)
				imageTypes := arch.ListImageTypes()

				var expectedImageTypes []string
				expectedImageTypes = append(expectedImageTypes, mapping.imgNames...)
				if dist.name == "fedora" {
					expectedImageTypes = append(expectedImageTypes, mapping.fedoraAdditionalImageTypes...)
				}

				require.ElementsMatch(t, expectedImageTypes, imageTypes)
			}
		})
	}
}

func TestFedora_ListArches(t *testing.T) {
	arches := fedora_core.NewF39().ListArches()
	assert.Equal(t, []string{"aarch64", "x86_64"}, arches)
}

func TestFedoraCore39_GetArch(t *testing.T) {
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
			name:          "s390x",
			errorExpected: true,
		},
		{
			name:          "ppc64le",
			errorExpected: true,
		},
		{
			name:          "foo-arch",
			errorExpected: true,
		},
	}

	for _, dist := range fedoraFamilyDistros {
		t.Run(dist.name, func(t *testing.T) {
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

func TestDistro_CustomFileSystemManifestError(t *testing.T) {
	fedoraDistro := fedora_core.NewF39()
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
	for _, archName := range fedoraDistro.ListArches() {
		arch, _ := fedoraDistro.GetArch(archName)
		for _, imgTypeName := range arch.ListImageTypes() {
			imgType, _ := arch.GetImageType(imgTypeName)
			_, _, err := imgType.Manifest(&bp, distro.ImageOptions{}, nil, 0)

			assert.EqualError(t, err, "The following custom mountpoints are not supported [\"/etc\"]")
		}
	}
}

func TestDistro_TestRootMountPoint(t *testing.T) {
	fedoraDistro := fedora_core.NewF39()
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
	for _, archName := range fedoraDistro.ListArches() {
		arch, _ := fedoraDistro.GetArch(archName)
		for _, imgTypeName := range arch.ListImageTypes() {
			imgType, _ := arch.GetImageType(imgTypeName)
			_, _, err := imgType.Manifest(&bp, distro.ImageOptions{}, nil, 0)
			if imgTypeName == "iot-commit" || imgTypeName == "iot-container" {
				assert.EqualError(t, err, "Custom mountpoints are not supported for ostree types")
			} else if imgTypeName == "iot-raw-image" {
				assert.EqualError(t, err, fmt.Sprintf("unsupported blueprint customizations found for image type %q: (allowed: User, Group, Directories, Files, Services)", imgTypeName))
			} else if imgTypeName == "iot-installer" || imgTypeName == "image-installer" {
				continue
			} else if imgTypeName == "live-installer" {
				assert.EqualError(t, err, fmt.Sprintf("unsupported blueprint customizations found for boot ISO image type \"%s\": (allowed: None)", imgTypeName))
			} else {
				assert.NoError(t, err)
			}
		}
	}
}

func TestDistro_CustomFileSystemSubDirectories(t *testing.T) {
	fedoraDistro := fedora_core.NewF39()
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
	for _, archName := range fedoraDistro.ListArches() {
		arch, _ := fedoraDistro.GetArch(archName)
		for _, imgTypeName := range arch.ListImageTypes() {
			imgType, _ := arch.GetImageType(imgTypeName)
			_, _, err := imgType.Manifest(&bp, distro.ImageOptions{}, nil, 0)
			if strings.HasPrefix(imgTypeName, "iot-") || strings.HasPrefix(imgTypeName, "image-") {
				continue
			} else if imgTypeName == "live-installer" {
				assert.EqualError(t, err, fmt.Sprintf("unsupported blueprint customizations found for boot ISO image type \"%s\": (allowed: None)", imgTypeName))
			} else {
				assert.NoError(t, err)
			}
		}
	}
}

func TestDistro_MountpointsWithArbitraryDepthAllowed(t *testing.T) {
	fedoraDistro := fedora_core.NewF39()
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
	for _, archName := range fedoraDistro.ListArches() {
		arch, _ := fedoraDistro.GetArch(archName)
		for _, imgTypeName := range arch.ListImageTypes() {
			imgType, _ := arch.GetImageType(imgTypeName)
			_, _, err := imgType.Manifest(&bp, distro.ImageOptions{}, nil, 0)
			if strings.HasPrefix(imgTypeName, "iot-") || strings.HasPrefix(imgTypeName, "image-") {
				continue
			} else if imgTypeName == "live-installer" {
				assert.EqualError(t, err, fmt.Sprintf("unsupported blueprint customizations found for boot ISO image type \"%s\": (allowed: None)", imgTypeName))
			} else {
				assert.NoError(t, err)
			}
		}
	}
}

func TestDistro_DirtyMountpointsNotAllowed(t *testing.T) {
	fedoraDistro := fedora_core.NewF39()
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
	for _, archName := range fedoraDistro.ListArches() {
		arch, _ := fedoraDistro.GetArch(archName)
		for _, imgTypeName := range arch.ListImageTypes() {
			imgType, _ := arch.GetImageType(imgTypeName)
			_, _, err := imgType.Manifest(&bp, distro.ImageOptions{}, nil, 0)
			if strings.HasPrefix(imgTypeName, "iot-") || strings.HasPrefix(imgTypeName, "image-") {
				continue
			} else if imgTypeName == "live-installer" {
				assert.EqualError(t, err, fmt.Sprintf("unsupported blueprint customizations found for boot ISO image type \"%s\": (allowed: None)", imgTypeName))
			} else {
				assert.EqualError(t, err, "The following custom mountpoints are not supported [\"//\" \"/var//\" \"/var//log/audit/\"]")
			}
		}
	}
}

func TestDistro_CustomFileSystemPatternMatching(t *testing.T) {
	fedoraDistro := fedora_core.NewF39()
	bp := blueprint.Blueprint{
		Customizations: &blueprint.Customizations{
			Filesystem: []blueprint.FilesystemCustomization{
				{
					MinSize:    1024,
					Mountpoint: "/variable",
				},
				{
					MinSize:    1024,
					Mountpoint: "/variable/log/audit",
				},
			},
		},
	}
	for _, archName := range fedoraDistro.ListArches() {
		arch, _ := fedoraDistro.GetArch(archName)
		for _, imgTypeName := range arch.ListImageTypes() {
			imgType, _ := arch.GetImageType(imgTypeName)
			_, _, err := imgType.Manifest(&bp, distro.ImageOptions{}, nil, 0)
			if imgTypeName == "iot-commit" || imgTypeName == "iot-container" {
				assert.EqualError(t, err, "Custom mountpoints are not supported for ostree types")
			} else if imgTypeName == "iot-raw-image" {
				assert.EqualError(t, err, fmt.Sprintf("unsupported blueprint customizations found for image type %q: (allowed: User, Group, Directories, Files, Services)", imgTypeName))
			} else if imgTypeName == "iot-installer" || imgTypeName == "image-installer" {
				continue
			} else if imgTypeName == "live-installer" {
				assert.EqualError(t, err, fmt.Sprintf("unsupported blueprint customizations found for boot ISO image type \"%s\": (allowed: None)", imgTypeName))
			} else {
				assert.EqualError(t, err, "The following custom mountpoints are not supported [\"/variable\" \"/variable/log/audit\"]")
			}
		}
	}
}

func TestDistro_CustomUsrPartitionNotLargeEnough(t *testing.T) {
	fedoraDistro := fedora_core.NewF39()
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
	for _, archName := range fedoraDistro.ListArches() {
		arch, _ := fedoraDistro.GetArch(archName)
		for _, imgTypeName := range arch.ListImageTypes() {
			imgType, _ := arch.GetImageType(imgTypeName)
			_, _, err := imgType.Manifest(&bp, distro.ImageOptions{}, nil, 0)
			if imgTypeName == "iot-commit" || imgTypeName == "iot-container" {
				assert.EqualError(t, err, "Custom mountpoints are not supported for ostree types")
			} else if imgTypeName == "iot-raw-image" {
				assert.EqualError(t, err, fmt.Sprintf("unsupported blueprint customizations found for image type %q: (allowed: User, Group, Directories, Files, Services)", imgTypeName))
			} else if imgTypeName == "iot-installer" || imgTypeName == "image-installer" {
				continue
			} else if imgTypeName == "live-installer" {
				assert.EqualError(t, err, fmt.Sprintf("unsupported blueprint customizations found for boot ISO image type \"%s\": (allowed: None)", imgTypeName))
			} else {
				assert.NoError(t, err)
			}
		}
	}
}
