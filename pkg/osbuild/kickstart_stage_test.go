package osbuild_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/pkg/osbuild"
)

func TestKickstartStageJsonHappy(t *testing.T) {
	opts := &osbuild.KickstartStageOptions{
		Path: "/osbuild.ks",
		Bootloader: &osbuild.BootloaderOptions{
			Append: "karg1 karg2=0",
		},
	}
	stage := osbuild.NewKickstartStage(opts)
	require.NotNil(t, stage)
	stageJson, err := json.MarshalIndent(stage, "", "  ")
	require.Nil(t, err)
	assert.Equal(t, string(stageJson), `{
  "type": "org.osbuild.kickstart",
  "options": {
    "path": "/osbuild.ks",
    "bootloader": {
      "append": "karg1 karg2=0"
    }
  }
}`)
}
