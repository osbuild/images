package generic

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/blueprint/pkg/blueprint"
	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/defs"
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

func testImageType() *imageType {
	return &imageType{
		arch: &architecture{
			distro: &distribution{},
		},
	}
}

func TestOSCustomizationsRedactsPasswords(t *testing.T) {
	password := "super-secret-password"
	bp := &blueprint.Blueprint{
		Customizations: &blueprint.Customizations{
			User: []blueprint.UserCustomization{
				{
					Name:     "testuser",
					Password: &password,
				},
			},
		},
	}

	it := testImageType()
	osc, err := osCustomizations(it, rpmmd.PackageSet{}, distro.ImageOptions{}, nil, bp)
	require.NoError(t, err)
	require.NotEmpty(t, osc.BlueprintTOML)

	tomlStr := string(osc.BlueprintTOML)
	assert.True(t, strings.Contains(tomlStr, "testuser"), "blueprint TOML should contain the username")
	assert.False(t, strings.Contains(tomlStr, password), "blueprint TOML should not contain the password")

	// DeepCopy must not mutate the original blueprint
	require.NotNil(t, bp.Customizations.User[0].Password)
	assert.Equal(t, password, *bp.Customizations.User[0].Password)
}

func TestOSCustomizationsBlueprintTOMLPopulated(t *testing.T) {
	bp := &blueprint.Blueprint{
		Name: "my-test-blueprint",
	}

	it := testImageType()
	osc, err := osCustomizations(it, rpmmd.PackageSet{}, distro.ImageOptions{}, []container.SourceSpec{}, bp)
	require.NoError(t, err)
	require.NotEmpty(t, osc.BlueprintTOML)

	tomlStr := string(osc.BlueprintTOML)
	assert.True(t, strings.Contains(tomlStr, "my-test-blueprint"))
}
