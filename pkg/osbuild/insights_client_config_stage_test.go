package osbuild

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewInsightsClientConfigStage(t *testing.T) {
	expectedStage := &Stage{
		Type:    "org.osbuild.insights-client.config",
		Options: &InsightsClientConfigStageOptions{},
	}
	actualStage := NewInsightsClientConfigStage(&InsightsClientConfigStageOptions{})
	assert.Equal(t, expectedStage, actualStage)
}
