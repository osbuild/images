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
			},
			expectedDistro: &BootcDistro{
				imgref:          "example.com/containers/distro-bootc:version12",
				imageID:         "acf88e518194fac963a1b2e2e4110e38a4ce5fb3fceddd624fae8997d4566930",
				buildImgref:     "example.com/containers/distro-bootc:version12",
				sourceInfo:      &osinfo.Info{},
				buildSourceInfo: &osinfo.Info{},
				id: distro.ID{
					Name: "bootc",
				},

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
			},
			expectedError: "failed to initialize bootc distro: missing required info: Imgref",
		},

		"noimageid": {
			info: &bootc.Info{
				Imgref:        "example.com/containers/distro-bootc:version12",
				Arch:          "amd64",
				DefaultRootFs: "xfs",
				Size:          100 * datasizes.MiB,
			},
			expectedDistro: &BootcDistro{
				imgref:          "example.com/containers/distro-bootc:version12",
				buildImgref:     "example.com/containers/distro-bootc:version12",
				sourceInfo:      &osinfo.Info{},
				buildSourceInfo: &osinfo.Info{},
				id: distro.ID{
					Name: "bootc",
				},

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
			expectedError: "failed to initialize bootc distro: missing required info: Arch, DefaultRootFs, Size",
		},

		"unknown-arch": {
			info: &bootc.Info{
				Imgref:        "example.com/containers/distro-bootc:version12",
				ImageID:       "acf88e518194fac963a1b2e2e4110e38a4ce5fb3fceddd624fae8997d4566930",
				Arch:          "not-an-arch",
				DefaultRootFs: "xfs",
				Size:          100 * datasizes.MiB,
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
