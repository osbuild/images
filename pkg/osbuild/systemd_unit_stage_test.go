package osbuild

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSystemdUnitStage(t *testing.T) {
	expectedStage := &Stage{
		Type:    "org.osbuild.systemd.unit",
		Options: &SystemdUnitStageOptions{},
	}
	actualStage := NewSystemdUnitStage(&SystemdUnitStageOptions{})
	assert.Equal(t, expectedStage, actualStage)
}

func TestNewSystemdGlobalUnitStage(t *testing.T) {
	var options = SystemdUnitStageOptions{
		Unit:   "test.timer",
		Dropin: "10-greenboot.conf",
		Config: SystemdServiceUnitDropin{
			Unit: &SystemdUnitSection{
				FileExists: "/usr/lib/test",
			},
		},
		UnitType: Global,
	}

	expectedStage := &Stage{
		Type:    "org.osbuild.systemd.unit",
		Options: &options,
	}
	actualStage := NewSystemdUnitStage(&options)
	assert.Equal(t, expectedStage, actualStage)
}
