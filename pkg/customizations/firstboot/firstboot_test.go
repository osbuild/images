package firstboot_test

import (
	"testing"

	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/customizations/firstboot"
	"github.com/stretchr/testify/assert"
)

func TestFirstbootOptionsFromValidBP(t *testing.T) {
	validBP := blueprint.FirstbootCustomization{
		Custom: []blueprint.CustomFirstbootCustomization{
			{Contents: "echo hello", Name: "greet"},
		},
		Satellite: &blueprint.SatelliteFirstbootCustomization{
			Command: "satellite-command",
			CACerts: []string{"cert1", "cert2"},
		},
		AAP: &blueprint.AAPFirstbootCustomization{
			JobTemplateURL: "https://example.com/job-template",
			HostConfigKey:  "host-config-key",
			CACerts:        []string{"cert3"},
		},
	}

	options, err := firstboot.FirstbootOptionsFromBP(validBP)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.Len(t, options.Custom, 1)
	assert.Equal(t, "echo hello", options.Custom[0].Contents)

	assert.NotNil(t, options.Satellite)
	assert.Equal(t, "satellite-command", options.Satellite.Command)
	assert.ElementsMatch(t, []string{"cert1", "cert2"}, options.Satellite.CACerts)

	assert.NotNil(t, options.AAP)
	assert.Equal(t, "https://example.com/job-template", options.AAP.JobTemplateURL)
	assert.Equal(t, "host-config-key", options.AAP.HostConfigKey)
	assert.ElementsMatch(t, []string{"cert3"}, options.AAP.CACerts)
}

func TestFirstbootOptionsFromEmptyBP(t *testing.T) {
	emptyBP := blueprint.FirstbootCustomization{}
	options, err := firstboot.FirstbootOptionsFromBP(emptyBP)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.Empty(t, options.Custom)
	assert.Nil(t, options.Satellite)
	assert.Nil(t, options.AAP)
}

func TestFirstbootOptionsFromBPWithEmptyCustomScript(t *testing.T) {
	bpWithEmptyCustom := blueprint.FirstbootCustomization{
		Custom: []blueprint.CustomFirstbootCustomization{
			{Contents: "", Name: "empty-script"},
		},
	}

	_, err := firstboot.FirstbootOptionsFromBP(bpWithEmptyCustom)
	assert.Error(t, err, "expected error for empty custom script contents")
}

func TestFirstbootOptionsFromBPWithEmptySatelliteCommand(t *testing.T) {
	bpWithEmptySatellite := blueprint.FirstbootCustomization{
		Satellite: &blueprint.SatelliteFirstbootCustomization{
			Command: "",
		},
	}

	_, err := firstboot.FirstbootOptionsFromBP(bpWithEmptySatellite)
	assert.Error(t, err, "expected error for empty satellite command")
}

func TestFirstbootOptionsFromBPWithEmptyAAPJobTemplateURL(t *testing.T) {
	bpWithEmptyAAP := blueprint.FirstbootCustomization{
		AAP: &blueprint.AAPFirstbootCustomization{
			JobTemplateURL: "",
		},
	}

	_, err := firstboot.FirstbootOptionsFromBP(bpWithEmptyAAP)
	assert.Error(t, err, "expected error for empty AAP job template URL")
}

func TestFirstbootOptionsFromBPWithInvalidAAPJobTemplateURL(t *testing.T) {
	bpWithInvalidAAP := blueprint.FirstbootCustomization{
		AAP: &blueprint.AAPFirstbootCustomization{
			JobTemplateURL: "not-a-valid-url",
		},
	}

	_, err := firstboot.FirstbootOptionsFromBP(bpWithInvalidAAP)
	assert.Error(t, err, "expected error for invalid AAP job template URL")
}
