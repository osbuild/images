package image_test

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/internal/testdisk"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/customizations/users"
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
	assert.Equal(t, img.Base.Name(), "bootc-raw-image")
}

func makeFakeDigest(t *testing.T) string {
	data := make([]byte, 32)
	_, err := rand.Read(data)
	require.Nil(t, err)
	return "sha256:" + hex.EncodeToString(data[:])
}

type bootcDiskImageTestOpts struct {
	ImageFormat platform.ImageFormat
	BIOS        bool
	SELinux     string
	Users       []users.User
	Groups      []users.Group

	KernelOptionsAppend []string
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
	img.Filename = "fake-disk"
	require.NotNil(t, img)
	img.Platform = makeFakePlatform(opts)
	img.PartitionTable = testdisk.MakeFakePartitionTable("/", "/boot", "/boot/efi")
	img.KernelOptionsAppend = opts.KernelOptionsAppend
	img.Users = opts.Users
	img.Groups = opts.Groups
	img.SELinux = opts.SELinux

	m := &manifest.Manifest{}
	runi := &runner.Fedora{}
	err := img.InstantiateManifestFromContainers(m, containers, runi, nil)
	require.Nil(t, err)

	fakeSourceSpecs := map[string][]container.Spec{
		"build": []container.Spec{{Source: "some-src", Digest: makeFakeDigest(t), ImageID: makeFakeDigest(t)}},
		"image": []container.Spec{{Source: "other-src", Digest: makeFakeDigest(t), ImageID: makeFakeDigest(t)}},
	}

	osbuildManifest, err := m.Serialize(nil, fakeSourceSpecs, nil, nil)
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

func TestBootcDiskImageInstantiateNoBuildpipelineForVpc(t *testing.T) {
	osbuildManifest := makeBootcDiskImageOsbuildManifest(t, nil)

	vpcPipeline := findPipelineFromOsbuildManifest(t, osbuildManifest, "vpc")
	require.NotNil(t, vpcPipeline)
	// no build pipeline for vpc
	assert.Equal(t, vpcPipeline["build"], nil)
}

func TestBootcDiskImageInstantiateVmdk(t *testing.T) {
	opts := &bootcDiskImageTestOpts{ImageFormat: platform.FORMAT_VMDK}
	osbuildManifest := makeBootcDiskImageOsbuildManifest(t, opts)

	pipeline := findPipelineFromOsbuildManifest(t, osbuildManifest, "vmdk")
	require.NotNil(t, pipeline)
}

func TestBootcDiskImageUsesBootcInstallToFs(t *testing.T) {
	opts := &bootcDiskImageTestOpts{
		KernelOptionsAppend: []string{"karg1", "karg2"},
	}
	osbuildManifest := makeBootcDiskImageOsbuildManifest(t, opts)

	// check that bootc.install-to-filesystem is part of the "image" pipeline
	imagePipeline := findPipelineFromOsbuildManifest(t, osbuildManifest, "image")
	require.NotNil(t, imagePipeline)
	bootcStage := findStageFromOsbuildPipeline(t, imagePipeline, "org.osbuild.bootc.install-to-filesystem")
	require.NotNil(t, bootcStage)

	// ensure loopback for the entire disk with partscan is used or install
	// to-filesystem will fail
	devicesDisk := bootcStage["devices"].(map[string]interface{})["disk"].(map[string]interface{})
	assert.Equal(t, "org.osbuild.loopback", devicesDisk["type"])
	devicesDiskOpts := devicesDisk["options"].(map[string]interface{})
	expectedDiskOpts := map[string]interface{}{
		"partscan": true,
		"filename": "fake-disk.raw",
	}
	assert.Equal(t, expectedDiskOpts, devicesDiskOpts)

	// ensure options got passed
	bootcOpts := bootcStage["options"].(map[string]interface{})
	assert.Equal(t, []interface{}{"karg1", "karg2"}, bootcOpts["kernel-args"])
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

	// vpc pipeline for the vhd
	vpcPipeline := findPipelineFromOsbuildManifest(t, osbuildManifest, "vpc")
	require.NotNil(vpcPipeline)

	// tar pipeline for ova
	tarPipeline := findPipelineFromOsbuildManifest(t, osbuildManifest, "archive")
	require.NotNil(tarPipeline)

	// gce pipeline
	gcePipeline := findPipelineFromOsbuildManifest(t, osbuildManifest, "gce")
	require.NotNil(gcePipeline)
}

func TestBootcDiskImageInstantiateUsers(t *testing.T) {
	for _, withUsers := range []bool{true, false} {
		opts := &bootcDiskImageTestOpts{}
		if withUsers {
			opts.Users = []users.User{{Name: "foo"}}
		}
		osbuildManifest := makeBootcDiskImageOsbuildManifest(t, opts)
		imagePipeline := findPipelineFromOsbuildManifest(t, osbuildManifest, "image")
		require.NotNil(t, imagePipeline)
		usersStage := findStageFromOsbuildPipeline(t, imagePipeline, "org.osbuild.users")
		if withUsers {
			require.NotNil(t, usersStage)
		} else {
			require.Nil(t, usersStage)
		}
	}
}

func TestBootcDiskImageInstantiateSELinuxForUsers(t *testing.T) {
	for _, withSELinux := range []string{"", "targeted"} {
		opts := &bootcDiskImageTestOpts{
			Users: []users.User{
				{Name: "foo"},
			},
			SELinux: withSELinux,
		}
		osbuildManifest := makeBootcDiskImageOsbuildManifest(t, opts)

		imagePipeline := findPipelineFromOsbuildManifest(t, osbuildManifest, "image")
		require.NotNil(t, imagePipeline)
		selinuxStage := findStageFromOsbuildPipeline(t, imagePipeline, "org.osbuild.selinux")
		if withSELinux != "" {
			require.NotNil(t, selinuxStage)
		} else {
			require.Nil(t, selinuxStage)
		}
	}
}

func TestBootcDiskImageInstantiateGroups(t *testing.T) {
	for _, withGroup := range []bool{true, false} {
		opts := &bootcDiskImageTestOpts{}
		if withGroup {
			opts.Groups = []users.Group{{Name: "foo-grp"}}
		}
		osbuildManifest := makeBootcDiskImageOsbuildManifest(t, opts)
		imagePipeline := findPipelineFromOsbuildManifest(t, osbuildManifest, "image")
		require.NotNil(t, imagePipeline)
		groupsStage := findStageFromOsbuildPipeline(t, imagePipeline, "org.osbuild.groups")
		if withGroup {
			require.NotNil(t, groupsStage)
		} else {
			require.Nil(t, groupsStage)
		}
	}
}
