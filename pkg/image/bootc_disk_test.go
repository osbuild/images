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

type bootcDiskImageTestOpts struct {
	ImageFormat platform.ImageFormat
	BIOS        bool
}

func makeFakePlatform(opts *bootcDiskImageTestOpts) platform.Platform {
	return &platform.X86{
		BasePlatform: platform.BasePlatform{
			ImageFormat: opts.ImageFormat,
		},
		BIOS: opts.BIOS,
	}
}

func makeBootcDiskImageOsbuildManifest(t *testing.T, opts *bootcDiskImageTestOpts) manifest.OSBuildManifest {
	if opts == nil {
		opts = &bootcDiskImageTestOpts{
			ImageFormat: platform.FORMAT_QCOW2,
		}
	}

	containerSource := container.SourceSpec{
		Source: "some-src",
		Name:   "name",
	}
	containers := []container.SourceSpec{containerSource}

	img := image.NewBootcDiskImage(containerSource)
	require.NotNil(t, img)
	img.Platform = makeFakePlatform(opts)
	img.PartitionTable = testdisk.MakeFakePartitionTable("/", "/boot", "/boot/efi")

	m := &manifest.Manifest{}
	runi := &runner.Fedora{}
	err := img.InstantiateManifestFromContainers(m, containers, runi, nil)
	require.Nil(t, err)

	fakeSourceSpecs := map[string][]container.Spec{
		"build":             []container.Spec{{Source: "some-src", Digest: makeFakeDigest(t), ImageID: makeFakeDigest(t)}},
		"ostree-deployment": []container.Spec{{Source: "other-src", Digest: makeFakeDigest(t), ImageID: makeFakeDigest(t)}},
	}

	osbuildManifest, err := m.Serialize(nil, fakeSourceSpecs, nil)
	require.Nil(t, err)

	return osbuildManifest
}

func findPipelineFromOsbuildManifest(t *testing.T, osbm manifest.OSBuildManifest, pipelineName string) map[string]interface{} {
	var mani map[string]interface{}

	err := json.Unmarshal(osbm, &mani)
	require.Nil(t, err)

	pipelines := mani["pipelines"].([]interface{})
	for _, pipelineIf := range pipelines {
		pipeline := pipelineIf.(map[string]interface{})
		if pipeline["name"].(string) == pipelineName {
			return pipeline
		}
	}
	return nil
}

func findStageFromOsbuildPipeline(t *testing.T, pipeline map[string]interface{}, stageName string) map[string]interface{} {
	stages := pipeline["stages"].([]interface{})
	for _, stageIf := range stages {
		stage := stageIf.(map[string]interface{})
		if stage["type"].(string) == stageName {
			return stage
		}
	}
	return nil
}

func TestBootcDiskImageInstantiateNoBuildpipelineForQcow2(t *testing.T) {
	osbuildManifest := makeBootcDiskImageOsbuildManifest(t, nil)

	qcowPipeline := findPipelineFromOsbuildManifest(t, osbuildManifest, "qcow2")
	require.NotNil(t, qcowPipeline)
	// no build pipeline for qcow2
	assert.Equal(t, qcowPipeline["build"], nil)
}

func TestBootcDiskImageInstantiateVmdk(t *testing.T) {
	opts := &bootcDiskImageTestOpts{ImageFormat: platform.FORMAT_VMDK}
	osbuildManifest := makeBootcDiskImageOsbuildManifest(t, opts)

	pipeline := findPipelineFromOsbuildManifest(t, osbuildManifest, "vmdk")
	require.NotNil(t, pipeline)
}

func TestBootcDiskImageUsesBootupd(t *testing.T) {
	osbuildManifest := makeBootcDiskImageOsbuildManifest(t, nil)

	// check that bootupd is part of the "image" pipeline
	imagePipeline := findPipelineFromOsbuildManifest(t, osbuildManifest, "image")
	require.NotNil(t, imagePipeline)
	bootupdStage := findStageFromOsbuildPipeline(t, imagePipeline, "org.osbuild.bootupd")
	require.NotNil(t, bootupdStage)

	// ensure that "grub2" is not part of the ostree pipeline
	ostreeDeployPipeline := findPipelineFromOsbuildManifest(t, osbuildManifest, "ostree-deployment")
	require.NotNil(t, ostreeDeployPipeline)
	grubStage := findStageFromOsbuildPipeline(t, ostreeDeployPipeline, "org.osbuild.grub2")
	require.Nil(t, grubStage)
}

func TestBootcDiskImageBootupdBiosSupport(t *testing.T) {
	for _, withBios := range []bool{false, true} {
		osbuildManifest := makeBootcDiskImageOsbuildManifest(t, &bootcDiskImageTestOpts{BIOS: withBios, ImageFormat: platform.FORMAT_QCOW2})

		imagePipeline := findPipelineFromOsbuildManifest(t, osbuildManifest, "image")
		require.NotNil(t, imagePipeline)
		bootupdStage := findStageFromOsbuildPipeline(t, imagePipeline, "org.osbuild.bootupd")
		require.NotNil(t, bootupdStage)

		opts := bootupdStage["options"].(map[string]interface{})
		if withBios {
			biosOpts := opts["bios"].(map[string]interface{})
			assert.Equal(t, biosOpts["device"], "disk")
		} else {
			require.Nil(t, opts["bios"])
		}
	}
}

func TestBootcDiskImageExportPipelines(t *testing.T) {
	require := require.New(t)

	osbuildManifest := makeBootcDiskImageOsbuildManifest(t, &bootcDiskImageTestOpts{BIOS: true, ImageFormat: platform.FORMAT_QCOW2})
	imagePipeline := findPipelineFromOsbuildManifest(t, osbuildManifest, "image")
	require.NotNil(imagePipeline)
	truncateStage := findStageFromOsbuildPipeline(t, imagePipeline, "org.osbuild.truncate") // check the truncate stage that creates the disk file
	require.NotNil(truncateStage)
	sfdiskStage := findStageFromOsbuildPipeline(t, imagePipeline, "org.osbuild.sfdisk") // and the sfdisk stage that creates partitions
	require.NotNil(sfdiskStage)

	// qcow2 pipeline for the qcow2
	qcowPipeline := findPipelineFromOsbuildManifest(t, osbuildManifest, "qcow2")
	require.NotNil(qcowPipeline)

	// vmdk pipeline for the vmdk
	vmdkPipeline := findPipelineFromOsbuildManifest(t, osbuildManifest, "vmdk")
	require.NotNil(vmdkPipeline)

	// tar pipeline for ova
	tarPipeline := findPipelineFromOsbuildManifest(t, osbuildManifest, "archive")
	require.NotNil(tarPipeline)
}
