package generic_test

import (
	"testing"

	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/defs"
	"github.com/osbuild/images/pkg/distro/generic"
	"github.com/stretchr/testify/assert"
)

func TestCheckOptionsFedora(t *testing.T) {
	type testCase struct {
		it      string
		bp      blueprint.Blueprint
		options distro.ImageOptions
		expErr  string
	}

	// For this test, we just need ImageType instances with a couple of fields
	// set (name, RPMOSTree). However, it's impossible to create one with a
	// given name, because the name is private inside the ImageTypeYAML and
	// meant to only be set by the loader. So we use the real image types,
	// loaded from the YAML files into ImageTypeYAML and create the ImageType
	// itself directly.
	fedora, err := defs.NewDistroYAML("fedora-42")
	assert.NoError(t, err)
	imageTypes := fedora.ImageTypes()

	testCases := map[string]testCase{
		"qcow2-ok": {
			it:      "qcow2",
			bp:      blueprint.Blueprint{},
			options: distro.ImageOptions{},
			expErr:  "",
		},
		"qcow2-no-installer": {
			it: "qcow2",
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Installer: &blueprint.InstallerCustomization{
						Unattended: true,
					},
				},
			},
			options: distro.ImageOptions{},
			expErr:  "installer customizations are not supported for \"\"", // the name is still not set, because
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			it := generic.ImageType{ImageTypeYAML: imageTypes[tc.it]}
			_, err := generic.CheckOptionsFedora(&it, &tc.bp, tc.options)
			if tc.expErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expErr)
			}
		})
	}
}
