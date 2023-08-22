package osbuild

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenServicesPresetStage(t *testing.T) {
	type TestCase struct {
		Name          string
		Enabled       []string
		Disabled      []string
		ExpectedStage *Stage
	}

	tests := []TestCase{
		{
			Name:          "empty",
			ExpectedStage: nil,
		},
		{
			Name:    "enabled-only",
			Enabled: []string{"foo.service", "bar.service"},
			ExpectedStage: NewSystemdPresetStage(&SystemdPresetStageOptions{
				Presets: []Preset{
					{Name: "foo.service", State: StateEnable},
					{Name: "bar.service", State: StateEnable},
				},
			}),
		},
		{
			Name:     "disabled-only",
			Disabled: []string{"foo.service", "bar.service"},
			ExpectedStage: NewSystemdPresetStage(&SystemdPresetStageOptions{
				Presets: []Preset{
					{Name: "foo.service", State: StateDisable},
					{Name: "bar.service", State: StateDisable},
				},
			}),
		},
		{
			Name:     "enabled-and-disabled",
			Enabled:  []string{"foo.service", "bar.service"},
			Disabled: []string{"baz.service", "bob.service"},
			ExpectedStage: NewSystemdPresetStage(&SystemdPresetStageOptions{
				Presets: []Preset{
					{Name: "foo.service", State: StateEnable},
					{Name: "bar.service", State: StateEnable},
					{Name: "baz.service", State: StateDisable},
					{Name: "bob.service", State: StateDisable},
				},
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			actualStage := GenServicesPresetStage(tt.Enabled, tt.Disabled)
			assert.Equal(t, tt.ExpectedStage, actualStage)
		})
	}
}
