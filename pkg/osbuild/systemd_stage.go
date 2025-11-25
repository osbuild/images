package osbuild

import (
	"reflect"
)

type SystemdStageOptions struct {
	EnabledServices  []string `json:"enabled_services,omitempty"`
	DisabledServices []string `json:"disabled_services,omitempty"`
	MaskedServices   []string `json:"masked_services,omitempty"`
	DefaultTarget    string   `json:"default_target,omitempty"`
}

func (SystemdStageOptions) isStageOptions() {}

var _ = PathChanger(SystemdStageOptions{})

func (s SystemdStageOptions) PathsChanged() []string {
	if reflect.ValueOf(s).IsZero() {
		return nil
	}
	// we don't know what exactly systemctl will do so give a coarse view here
	// (which is okay)
	return []string{"/etc"}
}

func NewSystemdStage(options *SystemdStageOptions) *Stage {
	return &Stage{
		Type:    "org.osbuild.systemd",
		Options: options,
	}
}
