package oscap

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/osbuild/images/pkg/blueprint"
	"github.com/stretchr/testify/assert"
)

func TestOscapConfigGeneration(t *testing.T) {
	tests := []struct {
		name                string
		options             blueprint.OpenSCAPCustomization
		expectedRemediation *RemediationConfig
		expectedTailoring   *TailoringConfig
		err                 error
	}{
		{
			name:    "no-datastream",
			options: blueprint.OpenSCAPCustomization{},
			err:     fmt.Errorf("No OSCAP datastream specified and the distro does not have any default set"),
		},
		{
			name: "multiple-tailoring-options",
			options: blueprint.OpenSCAPCustomization{
				DataStream:    "datastream",
				Tailoring:     &blueprint.OpenSCAPTailoringCustomizations{},
				JSONTailoring: &blueprint.OpenSCAPJSONTailoringCustomizations{},
			},
			err: fmt.Errorf("Multiple tailoring types set, only one type can be chosen (JSON/Override rules)"),
		},
		{
			name: "no-json-filepath",
			options: blueprint.OpenSCAPCustomization{
				DataStream:    "datastream",
				JSONTailoring: &blueprint.OpenSCAPJSONTailoringCustomizations{},
			},
			err: fmt.Errorf("Filepath to an JSON tailoring file is required"),
		},
		{
			name: "no-json-tailoring-id",
			options: blueprint.OpenSCAPCustomization{
				DataStream: "datastream",
				JSONTailoring: &blueprint.OpenSCAPJSONTailoringCustomizations{
					Filepath: "/some/filepath.json",
				},
			},
			err: fmt.Errorf("Tailoring profile ID is required for an JSON tailoring file"),
		},
		{
			name: "valid-json-tailoring",
			options: blueprint.OpenSCAPCustomization{
				DataStream: "datastream",
				ProfileID:  "some-profile-id",
				JSONTailoring: &blueprint.OpenSCAPJSONTailoringCustomizations{
					Filepath:  "/some/filepath.json",
					ProfileID: "some-tailored-id",
				},
			},
			expectedRemediation: &RemediationConfig{
				Datastream:         "datastream",
				ProfileID:          "some-tailored-id",
				TailoringPath:      filepath.Join(DataDir, "tailoring.xml"),
				CompressionEnabled: true,
			},
			expectedTailoring: &TailoringConfig{
				RemediationConfig: RemediationConfig{
					Datastream:    "datastream",
					ProfileID:     "some-profile-id",
					TailoringPath: filepath.Join(DataDir, "tailoring.xml"),
				},
				JSONFilepath:      "/some/filepath.json",
				TailoredProfileID: "some-tailored-id",
			},
			err: nil,
		},
		{
			name: "valid-tailoring",
			options: blueprint.OpenSCAPCustomization{
				DataStream: "datastream",
				ProfileID:  "some-profile-id",
				Tailoring: &blueprint.OpenSCAPTailoringCustomizations{
					Selected:   []string{"one", "three"},
					Unselected: []string{"two", "four"},
				},
			},
			expectedRemediation: &RemediationConfig{
				Datastream:         "datastream",
				ProfileID:          "some-profile-id_osbuild_tailoring",
				TailoringPath:      filepath.Join(DataDir, "tailoring.xml"),
				CompressionEnabled: true,
			},
			expectedTailoring: &TailoringConfig{
				RemediationConfig: RemediationConfig{
					Datastream:    "datastream",
					ProfileID:     "some-profile-id",
					TailoringPath: filepath.Join(DataDir, "tailoring.xml"),
				},
				TailoredProfileID: "some-profile-id_osbuild_tailoring",
				Selected:          []string{"one", "three"},
				Unselected:        []string{"two", "four"},
			},
			err: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remediationConfig, tailoringConfig, err := NewConfigs(tt.options, nil)
			if tt.err != nil {
				assert.NotNil(t, err)
				assert.EqualValues(t, tt.err, err)
				assert.Nil(t, remediationConfig)
				assert.Nil(t, tailoringConfig)
			} else {
				assert.NoError(t, err)
				assert.EqualValues(t, tt.expectedRemediation, remediationConfig)
				assert.EqualValues(t, tt.expectedTailoring, tailoringConfig)
			}
		})
	}
}
