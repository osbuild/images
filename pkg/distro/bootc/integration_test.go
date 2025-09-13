package bootc_test

import (
	"bytes"
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

	imgTag := bootctest.NewFakeContainer(t, "bootc")
	buildImgTag := bootctest.NewFakeContainer(t, "build")

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
