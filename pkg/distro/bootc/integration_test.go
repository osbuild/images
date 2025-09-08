package bootc_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/blueprint/pkg/blueprint"

	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/bootc"
	"github.com/osbuild/images/pkg/distro/bootc/bootctest"
	"github.com/osbuild/images/pkg/manifestgen"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/osbuild/manifesttest"
	"github.com/osbuild/images/pkg/rpmmd"
)

func canRunIntegration(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("test needs root")
	}
	if _, err := exec.LookPath("podman"); err != nil {
		t.Skip("test needs installed podman")
	}
	if _, err := exec.LookPath("systemd-detect-virt"); err != nil {
		t.Skip("test needs systemd-detect-virt")
	}
	// exit code "0" means the container is detected
	if err := exec.Command("systemd-detect-virt", "-c", "-q").Run(); err == nil {
		t.Skip("test cannot run inside a container")
	}
}

func genManifest(t *testing.T, imgType distro.ImageType) string {
	var bp blueprint.Blueprint

	var manifestJson bytes.Buffer
	mg, err := manifestgen.New(nil, &manifestgen.Options{
		Output: &manifestJson,
		OverrideRepos: []rpmmd.RepoConfig{
			{Id: "not-used", BaseURLs: []string{"not-used"}},
		},
	})
	assert.NoError(t, err)
	err = mg.Generate(&bp, imgType.Arch().Distro(), imgType, imgType.Arch(), nil)
	assert.NoError(t, err)

	// XXX: it would be nice to return an *osbuild.Manifest here
	// and do all of this more structed, however this is not
	// working currently as osbuild.NewManifestsFromBytes() cannot
	// unmarshal our manifests because of:
	// "unexpected source name: org.osbuild.containers-storage"
	return manifestJson.String()
}

func TestBuildContainerHandling(t *testing.T) {
	canRunIntegration(t)

	imgTag := bootctest.NewFakeContainer(t, "bootc", nil)
	buildImgTag := bootctest.NewFakeContainer(t, "build", nil)

	for _, withBuildContainer := range []bool{true, false} {
		t.Run(fmt.Sprintf("build-cnt:%v", withBuildContainer), func(t *testing.T) {
			distri, err := bootc.NewBootcDistro(imgTag)
			require.NoError(t, err)
			if withBuildContainer {
				err = distri.SetBuildContainer(buildImgTag)
				require.NoError(t, err)
			}

			archi, err := distri.GetArch(arch.Current().String())
			require.NoError(t, err)
			imgType, err := archi.GetImageType("qcow2")
			assert.NoError(t, err)

			manifestJson := genManifest(t, imgType)
			pipelineNames, err := manifesttest.PipelineNamesFrom([]byte(manifestJson))
			require.NoError(t, err)
			buildStages, err := manifesttest.StagesForPipeline([]byte(manifestJson), "build")
			require.NoError(t, err)
			// the bootc container is always pulled
			assert.Contains(t, manifestJson, imgTag)
			if withBuildContainer {
				assert.Contains(t, manifestJson, buildImgTag)
				// validate that the usr/lib/bootc/install/ dir is copied
				assert.Contains(t, manifestJson, "usr/lib/bootc/install/")
				assert.Contains(t, buildStages, "org.osbuild.copy")
				// validate that we have a "target" pipeline for raw content
				assert.Contains(t, pipelineNames, "target")
			} else {
				assert.NotContains(t, manifestJson, buildImgTag)
				assert.NotContains(t, manifestJson, "usr/lib/bootc/install/")
				assert.NotContains(t, buildStages, "org.osbuild.copy")
				assert.NotContains(t, pipelineNames, "target")
			}
		})
	}
}

func TestInteratedBuildDiskYAML(t *testing.T) {
	canRunIntegration(t)

	diskYAML := `
partition_table:
  type: gpt
  partitions:
    - size: 100_000_000
      payload_type: raw
      payload:
        source_path: /lib/modules/6.17/aboot.img
    - size: 10_000_000_000
      payload_type: filesystem
      payload:
        type: ext4
        mountpoint: /
`
	extraFiles := map[string]string{
		"/usr/lib/bootc-image-builder/disk.yaml": diskYAML,
		"/lib/modules/6.17/aboot.img":            "fake aboot.img content",
	}

	imgTag := bootctest.NewFakeContainer(t, "bootc", extraFiles)
	buildImgTag := bootctest.NewFakeContainer(t, "build", nil)

	for _, withBuildContainer := range []bool{true, false} {
		t.Run(fmt.Sprintf("build-cnt:%v", withBuildContainer), func(t *testing.T) {
			distri, err := bootc.NewBootcDistro(imgTag)
			require.NoError(t, err)
			if withBuildContainer {
				err = distri.SetBuildContainer(buildImgTag)
				assert.NoError(t, err)
			}

			archi, err := distri.GetArch(arch.Current().String())
			require.NoError(t, err)
			imgType, err := archi.GetImageType("qcow2")
			assert.NoError(t, err)

			manifestJson := genManifest(t, imgType)
			mani, err := manifesttest.NewManifestFromBytes([]byte(manifestJson))
			require.NoError(t, err)
			var stage *manifesttest.Stage
			var refPipeline string
			// The binary file comes from the target bootc
			// container. We mount the target as the build env
			// by default but when using a custom build container
			// we setup a special "target" pipeline that points
			// to the real bootc container. Ensure this is honored.
			if withBuildContainer {
				assert.Equal(t, []string{"target", "build", "image", "qcow2"}, mani.PipelineNames()[:4])

				stage = mani.Pipelines[2].Stage("org.osbuild.write-device")
				assert.NotNil(t, stage)
				refPipeline = "name:target"
			} else {
				stage = mani.Pipelines[1].Stage("org.osbuild.write-device")
				assert.NotNil(t, stage)
				assert.Equal(t, []string{"build", "image", "qcow2"}, mani.PipelineNames()[:3])
				refPipeline = "name:build"
			}
			// check write device stage options
			var opts osbuild.WriteDeviceStageOptions
			err = json.Unmarshal(stage.Options, &opts)
			require.NoError(t, err)
			assert.Equal(t, osbuild.WriteDeviceStageOptions{From: "input://tree/lib/modules/6.17/aboot.img"}, opts)
			// check write device stage inputs
			var inputs osbuild.PipelineTreeInputs
			err = json.Unmarshal(stage.Inputs, &inputs)
			require.NoError(t, err)
			expected := osbuild.PipelineTreeInputs{
				"tree": *osbuild.NewTreeInput(refPipeline),
			}
			assert.Equal(t, expected, inputs)
		})
	}
}
