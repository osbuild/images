package generic

import (
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

func diskTestImageType() *imageType {
	return &imageType{
		arch: &architecture{
			distro: &distribution{},
		},
		ImageTypeYAML: defs.ImageTypeYAML{},
	}
}

func TestOSCustomizationsPodmanDefaultNetBackend(t *testing.T) {
	netavark := container.NetworkBackendNetavark

	tests := []struct {
		name        string
		backend     *container.NetworkBackend
		containers  []container.SourceSpec
		expectFile  bool
		expectedVal string
	}{
		{
			name:    "backend set with containers creates file",
			backend: &netavark,
			containers: []container.SourceSpec{
				{Source: "registry.example.com/test:latest"},
			},
			expectFile:  true,
			expectedVal: "netavark",
		},
		{
			name:    "nil backend with containers does not create file",
			backend: nil,
			containers: []container.SourceSpec{
				{Source: "registry.example.com/test:latest"},
			},
			expectFile: false,
		},
		{
			name:       "backend set without containers does not create file",
			backend:    &netavark,
			containers: nil,
			expectFile: false,
		},
		{
			name:       "nil backend without containers does not create file",
			backend:    nil,
			containers: nil,
			expectFile: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			it := diskTestImageType()
			it.ImageConfigYAML.ImageConfig = &distro.ImageConfig{
				PodmanDefaultNetBackend: tt.backend,
			}

			bp := &blueprint.Blueprint{}
			osc, err := osCustomizations(it, rpmmd.PackageSet{}, distro.ImageOptions{}, tt.containers, bp)
			require.NoError(t, err)

			const backendPath = "/var/lib/containers/storage/defaultNetworkBackend"
			var found bool
			for _, f := range osc.Files {
				if f.Path() == backendPath {
					found = true
					assert.Equal(t, []byte(tt.expectedVal), f.Data())
					break
				}
			}
			assert.Equal(t, tt.expectFile, found,
				"expected file present=%v at %s", tt.expectFile, backendPath)
		})
	}
}
