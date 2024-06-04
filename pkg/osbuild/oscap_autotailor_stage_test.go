package osbuild

import (
	"testing"

	"github.com/osbuild/images/internal/common"
	"github.com/stretchr/testify/assert"
)

func TestNewOscapAutotailorStage(t *testing.T) {
	stageOptions := &OscapAutotailorStageOptions{
		Filepath: "tailoring.xml",
		Config: OscapAutotailorConfig{
			Datastream: "test_stream",
			ProfileID:  common.ToPtr("test_profile"),
			NewProfile: "test_profile_osbuild_profile",
			Selected:   []string{"fast_rule"},
			Unselected: []string{"slow_rule"},
			Overrides: []AutotailorOverride{
				{
					Var:   "rule1",
					Value: "value1",
				},
			},
		},
	}

	expectedStage := &Stage{
		Type:    "org.osbuild.oscap.autotailor",
		Options: stageOptions,
	}
	actualStage := NewOscapAutotailorStage(stageOptions)
	assert.Equal(t, expectedStage, actualStage)
}

func TestOscapAutotailorStageOptionsValidate(t *testing.T) {
	tests := []struct {
		name    string
		options OscapAutotailorStageOptions
		err     bool
	}{
		{
			name:    "empty-options",
			options: OscapAutotailorStageOptions{},
			err:     true,
		},
		{
			name: "empty-datastream",
			options: OscapAutotailorStageOptions{
				Config: OscapAutotailorConfig{
					ProfileID: common.ToPtr("test-profile"),
				},
			},
			err: true,
		},
		{
			name: "empty-profile-id",
			options: OscapAutotailorStageOptions{
				Config: OscapAutotailorConfig{
					Datastream: "test-datastream",
				},
			},
			err: true,
		},
		{
			name: "empty-new-profile-name",
			options: OscapAutotailorStageOptions{
				Config: OscapAutotailorConfig{
					ProfileID:  common.ToPtr("test-profile"),
					Datastream: "test-datastream",
				},
			},
			err: true,
		},
		{
			name: "invalid-override-value",
			options: OscapAutotailorStageOptions{
				Config: OscapAutotailorConfig{
					ProfileID:  common.ToPtr("test-profile"),
					Datastream: "test-datastream",
					NewProfile: "test-profile-osbuild-profile",
					Overrides: []AutotailorOverride{
						{
							Var: "test-var",
						},
					},
				},
			},
			err: true,
		},
		{
			name: "invalid-override-value-string",
			options: OscapAutotailorStageOptions{
				Config: OscapAutotailorConfig{
					ProfileID:  common.ToPtr("test-profile"),
					Datastream: "test-datastream",
					NewProfile: "test-profile-osbuild-profile",
					Overrides: []AutotailorOverride{
						{
							Var:   "test-var",
							Value: "30",
						},
					},
				},
			},
			err: false,
		},
		{
			name: "valid-override-value-int",
			options: OscapAutotailorStageOptions{
				Config: OscapAutotailorConfig{
					ProfileID:  common.ToPtr("test-profile"),
					Datastream: "test-datastream",
					NewProfile: "test-profile-osbuild-profile",
					Overrides: []AutotailorOverride{
						{
							Var:   "test-var",
							Value: uint64(30),
						},
					},
				},
			},
			err: false,
		},
		{
			name: "valid-data",
			options: OscapAutotailorStageOptions{
				Config: OscapAutotailorConfig{
					ProfileID:  common.ToPtr("test-profile"),
					Datastream: "test-datastream",
					NewProfile: "test-profile-osbuild-profile",
				},
			},
			err: false,
		},
	}
	for idx := range tests {
		tt := tests[idx]
		t.Run(tt.name, func(t *testing.T) {
			if tt.err {
				assert.Errorf(t, tt.options.Config.validate(), "%q didn't return an error [idx: %d]", tt.name, idx)
				assert.Panics(t, func() { NewOscapAutotailorStage(&tt.options) })
			} else {
				assert.NoErrorf(t, tt.options.Config.validate(), "%q returned an error [idx: %d]", tt.name, idx)
				assert.NotPanics(t, func() { NewOscapAutotailorStage(&tt.options) })
			}
		})
	}
}
