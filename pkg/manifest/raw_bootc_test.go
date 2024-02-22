package manifest_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/internal/testdisk"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/runner"
)

func hasPipeline(haystack []manifest.Pipeline, needle manifest.Pipeline) bool {
	for _, p := range haystack {
		if p == needle {
			return true
		}
	}
	return false
}

func TestNewRawBootcImage(t *testing.T) {
	mani := manifest.New()
	runner := &runner.Linux{}
	buildIf := manifest.NewBuildFromContainer(&mani, runner, nil, nil)
	build := buildIf.(*manifest.BuildrootFromContainer)

	rawBootcPipeline := manifest.NewRawBootcImage(build, nil, nil)
	require.NotNil(t, rawBootcPipeline)

	assert.True(t, hasPipeline(build.Dependents(), rawBootcPipeline))

	// disk.img is hardcoded for filename
	assert.Equal(t, "disk.img", rawBootcPipeline.Filename())
}

func TestRawBootcImageSerializeHasInstallToFilesystem(t *testing.T) {
	mani := manifest.New()
	runner := &runner.Linux{}
	build := manifest.NewBuildFromContainer(&mani, runner, nil, nil)

	rawBootcPipeline := manifest.NewRawBootcImage(build, nil, nil)
	rawBootcPipeline.PartitionTable = testdisk.MakeFakePartitionTable("/", "/boot", "/boot/efi")
	rawBootcPipeline.SerializeStart(nil, []container.Spec{{Source: "foo"}}, nil)
	imagePipeline := rawBootcPipeline.Serialize()
	assert.Equal(t, "image", imagePipeline.Name)

	require.NotNil(t, manifest.FindStage("org.osbuild.bootc.install-to-filesystem", imagePipeline.Stages))
}

func TestRawBootcImageSerializeMountsValidated(t *testing.T) {
	mani := manifest.New()
	runner := &runner.Linux{}
	build := manifest.NewBuildFromContainer(&mani, runner, nil, nil)

	rawBootcPipeline := manifest.NewRawBootcImage(build, nil, nil)
	// note that we create a partition table without /boot here
	rawBootcPipeline.PartitionTable = testdisk.MakeFakePartitionTable("/", "/missing-boot")
	rawBootcPipeline.SerializeStart(nil, []container.Spec{{Source: "foo"}}, nil)
	assert.PanicsWithError(t, `required mounts for bootupd stage [/boot /boot/efi] missing`, func() {
		rawBootcPipeline.Serialize()
	})
}
