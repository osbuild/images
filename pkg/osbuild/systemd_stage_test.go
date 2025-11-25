package osbuild

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSystemdStage(t *testing.T) {
	expectedStage := &Stage{
		Type:    "org.osbuild.systemd",
		Options: &SystemdStageOptions{},
	}
	actualStage := NewSystemdStage(&SystemdStageOptions{})
	assert.Equal(t, expectedStage, actualStage)
	assert.Len(t, actualStage.Options.(PathChanger).PathsChanged(), 0)
}

func TestNewSystemdStageFilesChanged(t *testing.T) {
	st := NewSystemdStage(&SystemdStageOptions{EnabledServices: []string{"foo"}})
	assert.Equal(t, []string{"/etc"}, st.Options.(PathChanger).PathsChanged())
}
