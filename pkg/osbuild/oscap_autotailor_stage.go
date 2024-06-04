package osbuild

import (
	"fmt"

	"github.com/osbuild/images/pkg/customizations/oscap"
)

type OscapAutotailorStageOptions struct {
	Filepath string                `json:"filepath"`
	Config   OscapAutotailorConfig `json:"config"`
}

type OscapAutotailorConfig struct {
	NewProfile string               `json:"new_profile"`
	Datastream string               `json:"datastream"`
	ProfileID  string               `json:"profile_id"`
	Selected   []string             `json:"selected,omitempty"`
	Unselected []string             `json:"unselected,omitempty"`
	Overrides  []AutotailorOverride `json:"overrides,omitempty"`
}

type AutotailorOverride struct {
	Var   string      `json:"var"`
	Value interface{} `json:"value"`
}

func (OscapAutotailorStageOptions) isStageOptions() {}

func (c OscapAutotailorConfig) validate() error {
	if c.Datastream == "" {
		return fmt.Errorf("'datastream' must be specified")
	}

	if c.ProfileID == "" {
		return fmt.Errorf("'profile_id' must be specified")
	}

	if c.NewProfile == "" {
		return fmt.Errorf("'new_profile' must be specified")
	}

	for _, override := range c.Overrides {
		if _, ok := override.Value.(uint64); ok {
			continue
		}

		if _, ok := override.Value.(string); ok {
			continue
		}

		return fmt.Errorf("override 'value' for 'var' %s must be an integer or a string", override.Var)
	}

	return nil
}

func NewOscapAutotailorStage(options *OscapAutotailorStageOptions) *Stage {
	if err := options.Config.validate(); err != nil {
		panic(err)
	}

	return &Stage{
		Type:    "org.osbuild.oscap.autotailor",
		Options: options,
	}
}

func NewOscapAutotailorStageOptions(options *oscap.TailoringConfig) *OscapAutotailorStageOptions {
	if options == nil {
		return nil
	}

	var overrides []AutotailorOverride
	for _, override := range options.Overrides {
		overrides = append(overrides, AutotailorOverride{
			Var:   override.Var,
			Value: override.Value,
		})
	}

	return &OscapAutotailorStageOptions{
		Filepath: options.Filepath,
		Config: OscapAutotailorConfig{
			NewProfile: options.NewProfile,
			Datastream: options.RemediationConfig.Datastream,
			ProfileID:  options.RemediationConfig.ProfileID,
			Selected:   options.Selected,
			Unselected: options.Unselected,
			Overrides:  overrides,
		},
	}
}
