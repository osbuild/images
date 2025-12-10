package bootc

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/blueprint/pkg/blueprint"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/internal/randutil"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/depsolvednf"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/osbuild/manifesttest"
)

type manifestTestCase struct {
	config            *blueprint.Blueprint
	imageOptions      distro.ImageOptions
	imageRef          string
	imageType         string
	depsolved         map[string]depsolvednf.DepsolveResult
	containers        map[string][]container.Spec
	expStages         map[string][]string
	notExpectedStages map[string][]string
	err               string
	warnings          []string
}

func TestManifestGenerationEmptyConfig(t *testing.T) {
	testCases := map[string]manifestTestCase{
		"qcow2-base": {
			imageRef:  "example-img-ref",
			imageType: "qcow2",
		},
		"qcow2-empty-imgref": {
			imageRef:  "",
			imageType: "qcow2",
			err:       "internal error: no base image defined",
		},
		"pxe-base": {
			imageRef:  "example-img-ref",
			imageType: "pxe-tar-xz",
		},
		"pxe-empty-imgref": {
			imageRef:  "",
			imageType: "pxe-tar-xz",
			err:       "internal error: no base image defined",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			imgType := NewTestBootcImageType(tc.imageType)
			imgType.arch.distro.imgref = tc.imageRef
			_, _, err := imgType.Manifest(tc.config, tc.imageOptions, nil, common.ToPtr(int64(0)))
			if tc.err != "" {
				assert.EqualError(t, err, tc.err)
			}
		})
	}
}

func getUserConfig() *blueprint.Blueprint {
	// add a user
	pass := randutil.String(20)
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
	userConfig := getUserConfig()
	testCases := map[string]manifestTestCase{
		"qcow2-user": {
			config:    userConfig,
			imageType: "qcow2",
		},
		"pxe-user": {
			config:    userConfig,
			imageType: "pxe-tar-xz",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			imgType := NewTestBootcImageType(tc.imageType)
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
			imageType:  "qcow2",
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
			imageType:  "qcow2",
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
			config:    userConfig,
			imageType: "qcow2",
			err:       `cannot serialize pipeline "build": BuildrootFromContainer: serialization not started`,
		},
		"pxe-base": {
			config:     baseConfig,
			imageType:  "pxe-tar-xz",
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
		"pxe-user": {
			config:     userConfig,
			imageType:  "pxe-tar-xz",
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
		"pxe-nocontainer": {
			config:    userConfig,
			imageType: "pxe-tar-xz",
			err:       `cannot serialize pipeline "build": BuildrootFromContainer: serialization not started`,
		},
	}

	// Use an empty config: only the imgref is required
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			imgType := NewTestBootcImageType(tc.imageType)

			assert := assert.New(t)
			mf, _, err := imgType.Manifest(tc.config, tc.imageOptions, nil, common.ToPtr(int64(0)))
			assert.NoError(err) // this isn't the error we're testing for

			if tc.err != "" {
				_, err := mf.Serialize(tc.depsolved, tc.containers, nil, nil)
				assert.EqualError(err, tc.err)
			} else {
				manifestJson, err := mf.Serialize(tc.depsolved, tc.containers, nil, nil)
				assert.NoError(err)
				assert.NoError(checkStages(manifestJson, tc.expStages, tc.notExpectedStages))
			}
		})
	}
}

func TestBootcDistroGetArch(t *testing.T) {
	imgType := NewTestBootcImageType("qcow2")
	distro := imgType.Arch().Distro()

	arch, err := distro.GetArch("x86_64")
	assert.NoError(t, err)
	assert.Equal(t, arch, imgType.Arch())

	_, err = distro.GetArch("aarch64")
	assert.EqualError(t, err, `requested bootc arch "aarch64" does not match available arches [x86_64]`)
}

func TestManifestGenerationOvaFilename(t *testing.T) {
	bp := getUserConfig()
	imgOptions := distro.ImageOptions{}

	bd := NewTestBootcDistro()
	imgType, err := bd.arches["x86_64"].GetImageType("ova")
	assert.NoError(t, err)

	mf, _, err := imgType.Manifest(bp, imgOptions, nil, common.ToPtr(int64(0)))
	assert.NoError(t, err)
	manifestJson, err := mf.Serialize(nil, diskContainers, nil, nil)
	assert.NoError(t, err)
	mani, err := manifesttest.NewManifestFromBytes(manifestJson)
	assert.NoError(t, err)
	archivePipeline := mani.Pipeline("archive")
	assert.NotNil(t, archivePipeline)
	stages := archivePipeline.Stages
	assert.Len(t, stages, 1)
	var tarStageOptions osbuild.TarStageOptions
	err = json.Unmarshal(stages[0].Options, &tarStageOptions)
	assert.NoError(t, err)
	assert.Equal(t, "image.ova", tarStageOptions.Filename)
}

func TestManifestGenerationBlueprintValidation(t *testing.T) {
	imageOptions := distro.ImageOptions{}
	config := &blueprint.Blueprint{
		Customizations: &blueprint.Customizations{
			Repositories: []blueprint.RepositoryCustomization{
				{
					Id: "foo",
				},
			},
		},
	}

	testCases := map[string]manifestTestCase{
		"qcow2-base": {
			config:       config,
			imageOptions: imageOptions,
			imageRef:     "example-img-ref",
			imageType:    "qcow2",
			warnings:     []string{`blueprint validation failed for image type "qcow2": customizations.repositories: not supported`},
		},
		"pxe-base": {
			config:       config,
			imageOptions: imageOptions,
			imageRef:     "example-img-ref",
			imageType:    "pxe-tar-xz",
			warnings:     []string{`blueprint validation failed for image type "pxe-tar-xz": customizations.repositories: not supported`},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			imgType := NewTestBootcImageType(tc.imageType)
			assert := assert.New(t)
			_, warnings, err := imgType.Manifest(config, imageOptions, nil, common.ToPtr(int64(0)))
			if tc.err != "" {
				assert.EqualError(err, tc.err)
			}
			if len(tc.warnings) > 0 {
				assert.Equal(tc.warnings, warnings)
			}
		})
	}
}
