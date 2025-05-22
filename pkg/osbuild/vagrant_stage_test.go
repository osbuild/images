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

func TestNewVagrantStageWithVirtualBox(t *testing.T) {
	input := NewFilesInput(NewFilesInputPipelineObjectRef("stage", "img.raw", nil))
	inputs := VagrantStageInputs{Image: input}

	options := VagrantStageOptions{
		Provider: VagrantProviderLibvirt,
		VirtualBox: &VagrantVirtualBoxStageOptions{
			MacAddress: "ffffffffffff",
		},
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
			Provider: VagrantProviderVirtualBox,
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

func TestVagrantStageOptionsSyncedFolders(t *testing.T) {
	stageOptions := NewVagrantStageOptions(VagrantProviderVirtualBox)
	stageOptions.SyncedFolders = map[string]*VagrantSyncedFolderStageOptions{
		"/vagrant": &VagrantSyncedFolderStageOptions{
			Type: VagrantSyncedFolderTypeRsync,
		},
	}

	assert.NoError(t, stageOptions.validate())
}

func TestVagrantStageOptionsSyncedFoldersNoVirtualbox(t *testing.T) {
	stageOptions := NewVagrantStageOptions(VagrantProviderLibvirt)
	stageOptions.SyncedFolders = map[string]*VagrantSyncedFolderStageOptions{
		"/vagrant": &VagrantSyncedFolderStageOptions{
			Type: VagrantSyncedFolderTypeRsync,
		},
	}

	assert.EqualError(t, stageOptions.validate(), `syncedfolders are only available for the virtualbox provider not for "libvirt"`)
}
