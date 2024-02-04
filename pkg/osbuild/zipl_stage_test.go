package osbuild

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewZiplStage(t *testing.T) {
	expectedStage := &Stage{
		Type:    "org.osbuild.zipl",
		Options: &ZiplStageOptions{},
	}
	actualStage := NewZiplStage(&ZiplStageOptions{})
	assert.Equal(t, expectedStage, actualStage)
}
