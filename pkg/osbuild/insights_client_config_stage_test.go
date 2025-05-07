package osbuild

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewInsightsClientConfigStage(t *testing.T) {
	stageOptions := &InsightsClientConfigStageOptions{
		Config: InsightsClientConfig{Path: "foo/bar", Proxy: "proxy"},
	}
	expectedStage := &Stage{
		Type:    "org.osbuild.insights-client.config",
		Options: stageOptions,
	}
	actualStage := NewInsightsClientConfigStage(stageOptions)
	assert.Equal(t, expectedStage, actualStage)
}

func TestInsightsClientConfigOptionsJSON(t *testing.T) {
	stageOptions := &InsightsClientConfigStageOptions{
		Config: InsightsClientConfig{
			Proxy: "some-proxy",
			Path:  "some-path",
		},
	}
	b, err := json.Marshal(stageOptions)
	assert.NoError(t, err)
	assert.Equal(t, string(b), `{"config":{"proxy":"some-proxy","path":"some-path"}}`)
}
