package generic

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/blueprint/pkg/blueprint"
	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/defs"
	"github.com/osbuild/images/pkg/image"
	"github.com/osbuild/images/pkg/ostree"
	"github.com/osbuild/images/pkg/rpmmd"
)

func isoTestImageType() *imageType {
	return &imageType{
		arch: &architecture{
			distro: &distribution{},
		},
		ImageTypeYAML: defs.ImageTypeYAML{
			BootISO: true,
		},
		isoLabel: func(*imageType) string { return "iso-label" },
	}
}

func TestInstallerCustomizationsHonorKernelOptions(t *testing.T) {
	for _, tc := range []struct {
		imageConfig          *distro.ImageConfig
		kernelCustomizations *blueprint.KernelCustomization
		expected             []string
	}{
		{
			nil,
			nil,
			nil,
		},
		{
			nil,
			&blueprint.KernelCustomization{
				Append: "debug",
			},
			[]string{"debug"},
		},
		{
			&distro.ImageConfig{
				KernelOptions: []string{"default"},
			},
			nil,
			[]string{"default"},
		},
		{
			&distro.ImageConfig{
				KernelOptions: []string{"default"},
			},
			&blueprint.KernelCustomization{
				Append: "debug",
			},
			[]string{"default", "debug"},
		},
	} {
		it := isoTestImageType()
		it.ImageConfigYAML.ImageConfig = tc.imageConfig
		c := &blueprint.Customizations{Kernel: tc.kernelCustomizations}

		isc, err := installerCustomizations(it, c, distro.ImageOptions{})
		require.NoError(t, err)
		assert.Equal(t, tc.expected, isc.KernelOptionsAppend)
	}
}

func TestInstallerCustomizationsOverridePreview(t *testing.T) {
	for _, tc := range []struct {
		distroPreview bool
		imageOptions  distro.ImageOptions
		expected      bool
	}{
		{
			true,
			distro.ImageOptions{},
			true,
		},
		{
			false,
			distro.ImageOptions{},
			false,
		},
		{
			true,
			distro.ImageOptions{Preview: common.ToPtr(false)},
			false,
		},
		{
			false,
			distro.ImageOptions{Preview: common.ToPtr(true)},
			true,
		},
	} {
		it := isoTestImageType()
		distro := it.arch.distro.(*distribution)
		distro.Preview = tc.distroPreview

		isc, err := installerCustomizations(it, nil, tc.imageOptions)
		require.NoError(t, err)
		assert.Equal(t, tc.expected, isc.Preview)
	}
}

// TestKickstartKernelOptionsAppend tests that the kernel.append blueprint
// customization is properly propagated to the kickstart configuration for the
// installed system's bootloader in all installer image types.
func TestKickstartKernelOptionsAppend(t *testing.T) {
	// math/rand is good enough for testing
	/* #nosec G404 */
	rng := rand.New(rand.NewSource(0))

	kernelAppend := "debug console=ttyS0"

	bp := &blueprint.Blueprint{
		Customizations: &blueprint.Customizations{
			Kernel: &blueprint.KernelCustomization{
				Append: kernelAppend,
			},
		},
	}

	packageSets := map[string]rpmmd.PackageSet{
		osPkgsKey:        {},
		installerPkgsKey: {},
	}

	testCases := []struct {
		name     string
		imageFn  imageFunc
		isOSTree bool
	}{
		{
			name:    "imageInstallerImage",
			imageFn: imageInstallerImage,
		},
		{
			name:     "iotInstallerImage",
			imageFn:  iotInstallerImage,
			isOSTree: true,
		},
		{
			name:    "networkInstallerImage",
			imageFn: networkInstallerImage,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			it := isoTestImageType()
			io := distro.ImageOptions{}
			if tc.isOSTree {
				it.ImageTypeYAML.OSTree.Name = "fedora"
				it.ImageTypeYAML.OSTree.RemoteName = "fedora"
				io.OSTree = &ostree.ImageOptions{
					ImageRef: "test/ref",
					URL:      "http://example.com/repo",
				}
			}

			img, err := tc.imageFn(it, bp, io, packageSets, nil, nil, rng)
			require.NoError(t, err)

			// All installer image types embed AnacondaInstallerBase which has Kickstart
			var kickstartOpts []string
			switch v := img.(type) {
			case *image.AnacondaTarInstaller:
				require.NotNil(t, v.Kickstart)
				kickstartOpts = v.Kickstart.KernelOptionsAppend
			case *image.AnacondaOSTreeInstaller:
				require.NotNil(t, v.Kickstart)
				kickstartOpts = v.Kickstart.KernelOptionsAppend
			case *image.AnacondaNetInstaller:
				require.NotNil(t, v.Kickstart)
				kickstartOpts = v.Kickstart.KernelOptionsAppend
			default:
				t.Fatalf("unexpected image type: %T", img)
			}

			assert.Contains(t, kickstartOpts, kernelAppend,
				"Kickstart.KernelOptionsAppend should contain the kernel.append from blueprint")
		})
	}
}
