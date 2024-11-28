package osbuild

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

var expectedJSON = `{
  "type": "org.osbuild.mkswap",
  "options": {
    "uuid": "8a1fc521-02a0-4917-92a9-90a44d7e6503",
    "label": "some-label"
  },
  "devices": {
    "root": {
      "type": "org.osbuild.loopback"
    }
  }
}`

func TestNewMkswapStage(t *testing.T) {
	devices := make(map[string]Device)
	devices["root"] = Device{
		Type: "org.osbuild.loopback",
	}

	options := MkswapStageOptions{
		UUID:  "8a1fc521-02a0-4917-92a9-90a44d7e6503",
		Label: "some-label",
	}
	stage := NewMkswapStage(&options, devices)
	b, err := json.MarshalIndent(stage, "", "  ")
	assert.NoError(t, err)
	assert.Equal(t, expectedJSON, string(b))
}
