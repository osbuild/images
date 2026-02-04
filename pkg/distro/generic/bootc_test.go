package generic

import (
	"testing"

	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/bib/osinfo"
	"github.com/osbuild/images/pkg/bootc"
	"github.com/osbuild/images/pkg/datasizes"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/defs"
	"github.com/stretchr/testify/require"
)

func TestNewBootc(t *testing.T) {
	type testCase struct {
		info           *bootc.Info
		expectedDistro *BootcDistro
		expectedError  string
	}

	testCases := map[string]testCase{
		"empty": {
			expectedError: "failed to initialize bootc distro: container info is empty",
		},

		"ok": {
			info: &bootc.Info{
				Imgref:        "example.com/containers/distro-bootc:version12",
				ImageID:       "acf88e518194fac963a1b2e2e4110e38a4ce5fb3fceddd624fae8997d4566930",
				Arch:          "arm64",
				DefaultRootFs: "xfs",
				Size:          100 * datasizes.MiB,
				OSInfo: &osinfo.Info{
					OSRelease: osinfo.OSRelease{
						ID:        "distroID",
						VersionID: "83",
					},
				},
			},
			expectedDistro: &BootcDistro{
				imgref:      "example.com/containers/distro-bootc:version12",
				imageID:     "acf88e518194fac963a1b2e2e4110e38a4ce5fb3fceddd624fae8997d4566930",
				buildImgref: "example.com/containers/distro-bootc:version12",
				sourceInfo: &osinfo.Info{
					OSRelease: osinfo.OSRelease{
						ID:        "distroID",
						VersionID: "83",
					},
				},
				buildSourceInfo: &osinfo.Info{
					OSRelease: osinfo.OSRelease{
						ID:        "distroID",
						VersionID: "83",
					},
				},
				id: distro.ID{
					Name:         "bootc-distroID",
					MajorVersion: 83,
					MinorVersion: -1,
				},

				releasever:    "83",
				defaultFs:     "xfs",
				rootfsMinSize: 200 * datasizes.MiB,
				arches: map[string]distro.Arch{
					"aarch64": &architecture{
						arch: arch.ARCH_AARCH64,
					},
				},
			},
		},

		"noimgref": {
			info: &bootc.Info{
				ImageID:       "acf88e518194fac963a1b2e2e4110e38a4ce5fb3fceddd624fae8997d4566930",
				Arch:          "aarch64",
				DefaultRootFs: "xfs",
				Size:          100 * datasizes.MiB,
				OSInfo: &osinfo.Info{
					OSRelease: osinfo.OSRelease{
						ID:        "fedora",
						VersionID: "2000",
					},
				},
			},
			expectedError: "failed to initialize bootc distro: missing required info: Imgref",
		},

		"noimageid": {
			info: &bootc.Info{
				Imgref:        "example.com/containers/distro-bootc:version12",
				Arch:          "amd64",
				DefaultRootFs: "xfs",
				Size:          100 * datasizes.MiB,
				OSInfo: &osinfo.Info{
					OSRelease: osinfo.OSRelease{
						ID:        "aos",
						VersionID: "5000",
					},
				},
			},
			expectedDistro: &BootcDistro{
				imgref:      "example.com/containers/distro-bootc:version12",
				buildImgref: "example.com/containers/distro-bootc:version12",
				sourceInfo: &osinfo.Info{
					OSRelease: osinfo.OSRelease{
						ID:        "aos",
						VersionID: "5000",
					},
				},
				buildSourceInfo: &osinfo.Info{
					OSRelease: osinfo.OSRelease{
						ID:        "aos",
						VersionID: "5000",
					},
				},
				id: distro.ID{
					Name:         "bootc-aos",
					MajorVersion: 5000,
					MinorVersion: -1,
				},

				releasever:    "5000",
				defaultFs:     "xfs",
				rootfsMinSize: 200 * datasizes.MiB,
				arches: map[string]distro.Arch{
					"x86_64": &architecture{
						arch: arch.ARCH_X86_64,
					},
				},
			},
		},

		"missing-multiple": {
			info: &bootc.Info{
				Imgref: "example.com/containers/distro-bootc:version12",
			},
			expectedError: "failed to initialize bootc distro: missing required info: Arch, DefaultRootFs, Size, OSInfo",
		},

		"osinfo-without-values": {
			info: &bootc.Info{
				Imgref:        "example.com/containers/distro-bootc:version12",
				ImageID:       "acf88e518194fac963a1b2e2e4110e38a4ce5fb3fceddd624fae8997d4566930",
				Arch:          "aarch64",
				DefaultRootFs: "xfs",
				Size:          100 * datasizes.MiB,
				OSInfo:        &osinfo.Info{},
			},
			expectedError: "failed to initialize bootc distro: missing required info: OSInfo.OSRelease.ID, OSInfo.OSRelease.VersionID",
		},

		"unknown-arch": {
			info: &bootc.Info{
				Imgref:        "example.com/containers/distro-bootc:version12",
				ImageID:       "acf88e518194fac963a1b2e2e4110e38a4ce5fb3fceddd624fae8997d4566930",
				Arch:          "not-an-arch",
				DefaultRootFs: "xfs",
				Size:          100 * datasizes.MiB,
				OSInfo: &osinfo.Info{
					OSRelease: osinfo.OSRelease{
						ID:        "aos",
						VersionID: "5000",
					},
				},
			},
			expectedError: "failed to set bootc distro architecture: unsupported architecture \"not-an-arch\"",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			d, err := NewBootc("bootc", tc.info)

			if tc.expectedError != "" {
				require.EqualError(err, tc.expectedError)
				return
			}

			require.NotNil(d)
			loadImageTypes(t, tc.expectedDistro)
			require.Equal(tc.expectedDistro, d)
		})
	}
}

func TestSetBuildContainer(t *testing.T) {
	// base bootc container info to initialise the distro before setting the
	// build container info
	baseBootcInfo := &bootc.Info{
		Imgref:        "example.com/containers/distro-bootc:version12",
		ImageID:       "acf88e518194fac963a1b2e2e4110e38a4ce5fb3fceddd624fae8997d4566930",
		Arch:          "aarch64",
		DefaultRootFs: "xfs",
		Size:          100 * datasizes.MiB,
		OSInfo: &osinfo.Info{
			OSRelease: osinfo.OSRelease{
				ID:        "whatever",
				VersionID: "39",
			},
		},
	}

	type testCase struct {
		buildInfo       *bootc.Info
		expectedImgref  string
		expectedImageID string
		expectedError   string
	}

	testCases := map[string]testCase{
		"empty": {
			expectedError: "failed to set build container for bootc distro: container info is empty",
		},

		"ok": {
			buildInfo: &bootc.Info{
				Imgref:  "example.com/containers/distro-bootc:build42",
				ImageID: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				Arch:    "arm64",
			},
			expectedImgref:  "example.com/containers/distro-bootc:build42",
			expectedImageID: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		},

		"noimgref": {
			buildInfo: &bootc.Info{
				ImageID: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				Arch:    "arm64",
			},
			expectedError: "failed to set build container for bootc distro: missing required info: Imgref",
		},

		"missing-multiple": {
			buildInfo: &bootc.Info{
				ImageID: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			},
			expectedError: "failed to set build container for bootc distro: missing required info: Imgref, Arch",
		},

		"noimageid": {
			buildInfo: &bootc.Info{
				Imgref: "example.com/containers/distro-bootc:build13",
				Arch:   "arm64",
			},
			expectedImgref: "example.com/containers/distro-bootc:build13",
		},

		"arch-mismatch": {
			buildInfo: &bootc.Info{
				Imgref: "example.com/containers/distro-bootc:build99",
				Arch:   "amd64",
			},
			expectedError: "failed to set build container for bootc distro: build container architecture \"x86_64\" does not match base container \"aarch64\"",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			bd, err := NewBootc("bootc", baseBootcInfo)
			require.NoError(err)
			require.NotNil(bd)

			err = bd.SetBuildContainer(tc.buildInfo)
			if tc.expectedError != "" {
				require.EqualError(err, tc.expectedError)
				return
			}

			require.Equal(tc.expectedImgref, bd.buildImgref)
			require.Equal(tc.expectedImageID, bd.buildImageID)
		})
	}
}

func TestSetBuildContainerWrongNumArches(t *testing.T) {
	baseBootcInfo := &bootc.Info{
		Imgref:        "example.com/containers/distro-bootc:version12",
		ImageID:       "acf88e518194fac963a1b2e2e4110e38a4ce5fb3fceddd624fae8997d4566930",
		Arch:          "aarch64",
		DefaultRootFs: "xfs",
		Size:          100 * datasizes.MiB,
		OSInfo: &osinfo.Info{
			OSRelease: osinfo.OSRelease{
				ID:        "whatever",
				VersionID: "39",
			},
		},
	}
	buildInfo := &bootc.Info{
		Imgref: "example.com/containers/distro-bootc:build99",
		Arch:   "aarch64",
	}

	require := require.New(t)
	bd, err := NewBootc("bootc", baseBootcInfo)
	require.NoError(err)
	require.NotNil(bd)

	require.Len(bd.arches, 1)

	// add a second architecture to test the error handling
	bd.arches["s390x"] = &architecture{
		distro:     bd,
		arch:       arch.ARCH_S390X,
		imageTypes: map[string]distro.ImageType{},
	}
	require.EqualError(bd.SetBuildContainer(buildInfo), "found 2 architectures for bootc distro while setting build container: bootc distro should have exactly 1 architecture")

	// remove the architectures to test the error handling
	bd.arches = nil
	require.EqualError(bd.SetBuildContainer(buildInfo), "found 0 architectures for bootc distro while setting build container: bootc distro should have exactly 1 architecture")
}

// Helper function for loading static bootc image type definitions onto the
// expected distro object.
func loadImageTypes(t *testing.T, d *BootcDistro) {
	t.Helper()

	require := require.New(t)

	distroYAML, err := defs.LoadDistroWithoutImageTypes("bootc-generic-1")
	require.NoError(err)

	fs, err := disk.NewFSType(d.defaultFs)
	require.NoError(err)

	distroYAML.DefaultFSType = fs // It's very weird that this is required here
	require.NoError(distroYAML.LoadImageTypes())

	for archName, arch := range d.arches {
		darch := arch.(*architecture)
		darch.imageTypes = map[string]distro.ImageType{}
		darch.distro = d // link distro to architecture as well
		require.NotNil(darch)
		for _, imgTypeYaml := range distroYAML.ImageTypes() {
			require.NoError(darch.addBootcImageType(bootcImageType{ImageTypeYAML: imgTypeYaml}))
		}
		d.arches[archName] = darch
	}
}
