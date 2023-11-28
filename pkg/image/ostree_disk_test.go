package image_test

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/internal/workload"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/image"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/platform"
	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/osbuild/images/pkg/runner"
)

func TestOSTreeDiskImageManifestSetsContainerBuildable(t *testing.T) {
	rng := rand.New(rand.NewSource(0)) // nolint:gosec

	repos := []rpmmd.RepoConfig{}
	r := &runner.Fedora{Version: 39}

	ref := "ostree/1/1/0"
	containerSource := container.SourceSpec{
		Source: "source-spec",
		Name:   "name",
	}

	var buildPipeline *manifest.Build
	restore := image.MockManifestNewBuild(func(m *manifest.Manifest, r runner.Runner, repos []rpmmd.RepoConfig) *manifest.Build {
		buildPipeline = manifest.NewBuild(m, r, repos)
		return buildPipeline
	})
	defer restore()

	for _, containerBuildable := range []bool{true, false} {
		mf := manifest.New()
		img := image.NewOSTreeDiskImageFromContainer(containerSource, ref)
		require.NotNil(t, img)
		img.Platform = &platform.X86{
			BasePlatform: platform.BasePlatform{
				ImageFormat: platform.FORMAT_QCOW2,
			},
			BIOS:       true,
			UEFIVendor: "fedora",
		}
		img.Workload = &workload.BaseWorkload{}
		img.OSName = "osname"
		img.ContainerBuildable = containerBuildable

		_, err := img.InstantiateManifest(&mf, repos, r, rng)
		require.Nil(t, err)
		require.NotNil(t, img)
		require.NotNil(t, buildPipeline)

		require.Equal(t, buildPipeline.ContainerBuildable, containerBuildable)
	}
}
