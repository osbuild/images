package image_test

import (
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/internal/testdisk"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/image"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/platform"
	"github.com/osbuild/images/pkg/runner"
)

func TestBootcDiskImageNew(t *testing.T) {
	containerSource := container.SourceSpec{
		Source: "source-spec",
		Name:   "name",
	}

	img := image.NewBootcDiskImage(containerSource)
	require.NotNil(t, img)
	assert.Equal(t, img.OSTreeDiskImage.Base.Name(), "bootc-raw-image")
}

func makeFakeDigest(t *testing.T) string {
	data := make([]byte, 32)
	_, err := rand.Read(data) // nolint:gosec
	require.Nil(t, err)
	return "sha256:" + hex.EncodeToString(data[:])
}

func makeFakePlatform() platform.Platform {
	return &platform.X86{
		BasePlatform: platform.BasePlatform{
			ImageFormat: platform.FORMAT_QCOW2,
		},
	}
}

func TestBootcDiskImageInstantiateNoBuildpipelineForQcow2(t *testing.T) {
	containerSource := container.SourceSpec{
		Source: "some-src",
		Name:   "name",
	}
	containers := []container.SourceSpec{containerSource}

	img := image.NewBootcDiskImage(containerSource)
	require.NotNil(t, img)
	img.Platform = makeFakePlatform()
	img.PartitionTable = testdisk.MakeFakePartitionTable("/")

	m := &manifest.Manifest{}
	runi := &runner.Fedora{}
	_, err := img.InstantiateManifestFromContainers(m, containers, runi, nil)
	require.Nil(t, err)
	sourceSpecs := map[string][]container.Spec{
		"build":             []container.Spec{{Source: "some-src", Digest: makeFakeDigest(t), ImageID: makeFakeDigest(t)}},
		"ostree-deployment": []container.Spec{{Source: "other-src", Digest: makeFakeDigest(t), ImageID: makeFakeDigest(t)}},
	}
	osbuildManifest, err := m.Serialize(nil, sourceSpecs, nil)
	require.Nil(t, err)

	var mani map[string]interface{}
	err = json.Unmarshal(osbuildManifest, &mani)
	require.Nil(t, err)
	pipelines := mani["pipelines"].([]interface{})
	findQcowStage := func() map[string]interface{} {
		for _, stageIf := range pipelines {
			stage := stageIf.(map[string]interface{})
			if stage["name"].(string) == "qcow2" {
				return stage
			}
		}
		return nil
	}
	qcowStage := findQcowStage()
	require.NotNil(t, qcowStage)
	// no build pipeline for qcow2
	assert.Equal(t, qcowStage["build"], nil)
}
