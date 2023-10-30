package osbuild

import (
	"testing"

	"github.com/google/uuid"
	"github.com/osbuild/images/pkg/container"
	"github.com/stretchr/testify/assert"
)

func TestOSTreeDeployContainersStageOptionsValidate(t *testing.T) {
	// options are validated first, so this doesn't necessarily need to be
	// valid, but we might change the order at some point.
	validInputs := NewContainersInputForSources([]container.Spec{
		{
			ImageID: "id-0",
			Source:  "registry.example.org/reg/img",
		},
	})

	type testCase struct {
		options OSTreeDeployContainerStageOptions
		valid   bool
	}

	testCases := map[string]testCase{
		"empty": {
			options: OSTreeDeployContainerStageOptions{},
			valid:   false,
		},
		"minimal": {
			options: OSTreeDeployContainerStageOptions{
				OsName:       "default",
				TargetImgref: "ostree-remote-registry:example.org/registry/image",
			},
			valid: true,
		},
		"no-target": {
			options: OSTreeDeployContainerStageOptions{
				OsName: "os",
			},
			valid: false,
		},
		"no-os": {
			options: OSTreeDeployContainerStageOptions{
				TargetImgref: "ostree-image-unverified-registry:example.org/registry/image",
			},
			valid: false,
		},
		"bad-target": {
			options: OSTreeDeployContainerStageOptions{
				OsName:       "os",
				TargetImgref: "bad",
			},
			valid: false,
		},
		"full": {
			options: OSTreeDeployContainerStageOptions{
				OsName:       "default",
				KernelOpts:   []string{},
				TargetImgref: "ostree-image-signed:example.org/registry/image",
				Rootfs: &Rootfs{
					// defining both is redundant but not invalid
					Label: "root",
					UUID:  uuid.New().String(),
				},
				Mounts: []string{"/data"},
			},
			valid: true,
		},
	}

	for name := range testCases {
		tc := testCases[name]
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			if tc.valid {
				assert.NoError(tc.options.validate())
				assert.NotPanics(func() { NewOSTreeDeployContainerStage(&tc.options, validInputs) })
			} else {
				assert.Error(tc.options.validate())
				assert.Panics(func() { NewOSTreeDeployContainerStage(&tc.options, validInputs) })
			}
		})
	}

}

func TestOSTreeDeployContainersStageInputsValidate(t *testing.T) {
	validOptions := &OSTreeDeployContainerStageOptions{
		OsName:       "default",
		TargetImgref: "ostree-remote-registry:example.org/registry/image",
	}

	type testCase struct {
		inputs OSTreeDeployContainerInputs
		valid  bool
	}

	testCases := map[string]testCase{
		"empty": {
			inputs: OSTreeDeployContainerInputs{},
			valid:  false,
		},
		"nil": {
			inputs: OSTreeDeployContainerInputs{
				Images: ContainersInput{
					References: nil,
				},
			},
			valid: false,
		},
		"zero": {
			inputs: OSTreeDeployContainerInputs{
				Images: NewContainersInputForSources([]container.Spec{}),
			},
			valid: false,
		},
		"one": {
			inputs: OSTreeDeployContainerInputs{
				Images: NewContainersInputForSources([]container.Spec{
					{
						ImageID: "id-0",
						Source:  "registry.example.org/reg/img",
					},
				}),
			},
			valid: true,
		},
		"two": {
			inputs: OSTreeDeployContainerInputs{
				Images: NewContainersInputForSources([]container.Spec{
					{
						ImageID: "id-1",
						Source:  "registry.example.org/reg/img-one",
					},
					{
						ImageID: "id-2",
						Source:  "registry.example.org/reg/img-two",
					},
				}),
			},
			valid: false,
		},
	}
	for name := range testCases {
		tc := testCases[name]
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			if tc.valid {
				assert.NoError(tc.inputs.validate())
				assert.NotPanics(func() { NewOSTreeDeployContainerStage(validOptions, tc.inputs.Images) })
			} else {
				assert.Error(tc.inputs.validate())
				assert.Panics(func() { NewOSTreeDeployContainerStage(validOptions, tc.inputs.Images) })
			}
		})
	}
}
