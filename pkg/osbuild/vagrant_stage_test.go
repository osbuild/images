package osbuild

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewVagrantStage(t *testing.T) {
	input := NewFilesInput(NewFilesInputPipelineObjectRef("stage", "img.raw", nil))
	inputs := VagrantStageInputs{Image: input}

	options := VagrantStageOptions{
		Provider: VagrantProviderLibvirt,
	}

	expectedStage := &Stage{
		Type:    "org.osbuild.vagrant",
		Options: &options,
		Inputs:  &inputs,
	}

	actualStage := NewVagrantStage(&options, &inputs)
	assert.Equal(t, expectedStage, actualStage)
}

func TestVagrantStageOptions(t *testing.T) {
	tests := []struct {
		Provider VagrantProvider
		Error    bool
	}{
		{
			Provider: VagrantProviderLibvirt,
		},
		{
			Provider: VagrantProviderVirtualbox,
		},
		// mismatch between format and format options type
		{
			Provider: "DoesNotExist",
			Error:    true,
		},
	}

	for idx, test := range tests {
		t.Run(fmt.Sprintf("test-(%d/%d)", idx, len(tests)), func(t *testing.T) {
			stageOptions := NewVagrantStageOptions(test.Provider)

			if test.Error {
				assert.Panics(t, func() { NewVagrantStage(stageOptions, nil) })
			} else {
				assert.EqualValues(t, test.Provider, stageOptions.Provider)
			}
		})
	}
}
