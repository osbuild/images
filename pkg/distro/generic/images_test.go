package generic

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/blueprint/pkg/blueprint"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/defs"
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

		isc, err := installerCustomizations(it, c)
		require.NoError(t, err)
		assert.Equal(t, tc.expected, isc.KernelOptionsAppend)
	}
}
