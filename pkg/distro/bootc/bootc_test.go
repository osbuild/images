package bootc

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/blueprint/pkg/blueprint"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/dnfjson"
	"github.com/osbuild/images/pkg/manifest"
)

type manifestTestCase struct {
	config            *blueprint.Blueprint
	imageOptions      distro.ImageOptions
	imageRef          string
	imageTypes        []string
	depsolved         map[string]dnfjson.DepsolveResult
	containers        map[string][]container.Spec
	expStages         map[string][]string
	notExpectedStages map[string][]string
	err               interface{}
}

func TestManifestGenerationEmptyConfig(t *testing.T) {
	imgType := NewTestBootcImageType()

	testCases := map[string]manifestTestCase{
		"qcow2-base": {
			imageRef:   "example-img-ref",
			imageTypes: []string{"qcow2"},
		},
		"empty-imgref": {
			imageRef:   "",
			imageTypes: []string{"qcow2"},
			err:        errors.New("internal error: no base image defined"),
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			imgType.arch.distro.imgref = tc.imageRef
			_, _, err := imgType.Manifest(tc.config, tc.imageOptions, nil, common.ToPtr(int64(0)))
			assert.Equal(t, err, tc.err)
		})
	}
}

func getUserConfig() *blueprint.Blueprint {
	// add a user
	pass := "super-secret-password-42"
	key := "ssh-ed25519 AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	return &blueprint.Blueprint{
		Customizations: &blueprint.Customizations{
			User: []blueprint.UserCustomization{
				{
					Name:     "tester",
					Password: &pass,
					Key:      &key,
				},
			},
		},
	}
}

func TestManifestGenerationUserConfig(t *testing.T) {
	imgType := NewTestBootcImageType()

	userConfig := getUserConfig()
	testCases := map[string]manifestTestCase{
		"qcow2-user": {
			config:     userConfig,
			imageTypes: []string{"qcow2"},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			_, _, err := imgType.Manifest(tc.config, tc.imageOptions, nil, common.ToPtr(int64(0)))
			assert.NoError(t, err)
		})
	}
}

// Disk images require a container for the build/image pipelines
var containerSpec = container.Spec{
	Source:  "test-container",
	Digest:  "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
	ImageID: "sha256:1111111111111111111111111111111111111111111111111111111111111111",
}

// diskContainers can be passed to Serialize() to get a minimal disk image
var diskContainers = map[string][]container.Spec{
	"build": {
		containerSpec,
	},
	"image": {
		containerSpec,
	},
	"target": {
		containerSpec,
	},
}

// simplified representation of a manifest
type testManifest struct {
	Pipelines []pipeline `json:"pipelines"`
}
type pipeline struct {
	Name   string  `json:"name"`
	Stages []stage `json:"stages"`
}
type stage struct {
	Type string `json:"type"`
}

func checkStages(serialized manifest.OSBuildManifest, pipelineStages map[string][]string, missingStages map[string][]string) error {
	mf := &testManifest{}
	if err := json.Unmarshal(serialized, mf); err != nil {
		return err
	}
	pipelineMap := map[string]pipeline{}
	for _, pl := range mf.Pipelines {
		pipelineMap[pl.Name] = pl
	}

	for plname, stages := range pipelineStages {
		pl, found := pipelineMap[plname]
		if !found {
			return fmt.Errorf("pipeline %q not found", plname)
		}

		stageMap := map[string]bool{}
		for _, stage := range pl.Stages {
			stageMap[stage.Type] = true
		}
		for _, stage := range stages {
			if _, found := stageMap[stage]; !found {
				return fmt.Errorf("pipeline %q - stage %q - not found", plname, stage)
			}
		}
	}

	for plname, stages := range missingStages {
		pl, found := pipelineMap[plname]
		if !found {
			return fmt.Errorf("pipeline %q not found", plname)
		}

		stageMap := map[string]bool{}
		for _, stage := range pl.Stages {
			stageMap[stage.Type] = true
		}
		for _, stage := range stages {
			if _, found := stageMap[stage]; found {
				return fmt.Errorf("pipeline %q - stage %q - found (but should not be)", plname, stage)
			}
		}
	}

	return nil
}

func TestManifestSerialization(t *testing.T) {
	baseConfig := &blueprint.Blueprint{}
	userConfig := getUserConfig()
	testCases := map[string]manifestTestCase{
		"qcow2-base": {
			config:     baseConfig,
			imageTypes: []string{"qcow2"},
			containers: diskContainers,
			expStages: map[string][]string{
				"build": {"org.osbuild.container-deploy"},
				"image": {
					"org.osbuild.bootc.install-to-filesystem",
				},
			},
			notExpectedStages: map[string][]string{
				"build": {"org.osbuild.rpm"},
				"image": {
					"org.osbuild.users",
				},
			},
		},
		"qcow2-user": {
			config:     userConfig,
			imageTypes: []string{"qcow2"},
			containers: diskContainers,
			expStages: map[string][]string{
				"build": {"org.osbuild.container-deploy"},
				"image": {
					"org.osbuild.users", // user creation stage when we add users
					"org.osbuild.bootc.install-to-filesystem",
				},
			},
			notExpectedStages: map[string][]string{
				"build": {"org.osbuild.rpm"},
			},
		},
		"qcow2-nocontainer": {
			config:     userConfig,
			imageTypes: []string{"qcow2"},
			// errors come from BuildrootFromContainer()
			// TODO: think about better error and testing here (not the ideal layer or err msg)
			err: "serialization not started",
		},
	}

	// Use an empty config: only the imgref is required
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			imgType := NewTestBootcImageType()

			assert := assert.New(t)
			mf, _, err := imgType.Manifest(tc.config, tc.imageOptions, nil, common.ToPtr(int64(0)))
			assert.NoError(err) // this isn't the error we're testing for

			if tc.err != nil {
				assert.PanicsWithValue(tc.err, func() {
					_, err := mf.Serialize(tc.depsolved, tc.containers, nil, nil)
					assert.NoError(err)
				})
			} else {
				manifestJson, err := mf.Serialize(tc.depsolved, tc.containers, nil, nil)
				assert.NoError(err)
				assert.NoError(checkStages(manifestJson, tc.expStages, tc.notExpectedStages))
			}
		})
	}
}
