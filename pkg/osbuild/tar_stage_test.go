package osbuild

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTarStage(t *testing.T) {
	stageOptions := &TarStageOptions{Filename: "archive.tar.xz"}
	stageInputs := &PipelineTreeInputs{"tree": TreeInput{
		inputCommon: inputCommon{
			Type:   "org.osbuild.tree",
			Origin: "org.osbuild.pipeline",
		},
		References: []string{
			"name:pipeline33",
		},
	},
	}
	expectedStage := &Stage{
		Type:    "org.osbuild.tar",
		Options: stageOptions,
		Inputs:  stageInputs,
	}
	actualStage := NewTarStage(stageOptions, "pipeline33")
	assert.Equal(t, expectedStage, actualStage)
}

func TestTarStageOptionsValidate(t *testing.T) {
	tests := []struct {
		name    string
		options TarStageOptions
		err     bool
	}{
		{
			name:    "empty-options",
			options: TarStageOptions{},
			err:     false,
		},
		{
			name: "invalid-archive-format",
			options: TarStageOptions{
				Filename: "archive.tar.xz",
				Format:   "made-up-format",
			},
			err: true,
		},
		{
			name: "invalid-archive-compression",
			options: TarStageOptions{
				Filename:    "archive.tar.xz",
				Format:      "made-up-format",
				Compression: "interpretative-dance",
			},
			err: true,
		},
		{
			name: "invalid-root-node",
			options: TarStageOptions{
				Filename: "archive.tar.xz",
				RootNode: "I-don't-care",
			},
			err: true,
		},
		{
			name: "valid-data",
			options: TarStageOptions{
				Filename:    "archive.tar.xz",
				Format:      TarArchiveFormatOldgnu,
				Compression: TarArchiveCompressionZstd,
				RootNode:    TarRootNodeOmit,
			},
			err: false,
		},
	}
	for idx := range tests {
		tt := tests[idx]
		t.Run(tt.name, func(t *testing.T) {
			if tt.err {
				assert.Errorf(t, tt.options.validate(), "%q didn't return an error [idx: %d]", tt.name, idx)
				assert.Panics(t, func() { NewTarStage(&tt.options, "") })
			} else {
				assert.NoErrorf(t, tt.options.validate(), "%q returned an error [idx: %d]", tt.name, idx)
				assert.NotPanics(t, func() { NewTarStage(&tt.options, "") })
			}
		})
	}
}

func TestTarStageOptionsJSON(t *testing.T) {
	stageOptions := &TarStageOptions{
		Filename:  "archive.tar.xz",
		Transform: "s/foo/bar/",
	}
	b, err := json.MarshalIndent(stageOptions, "", "  ")
	assert.NoError(t, err)
	assert.Equal(t, string(b), `{
  "filename": "archive.tar.xz",
  "transform": "s/foo/bar/"
}`)
}
