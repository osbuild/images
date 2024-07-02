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

	var buildOpts []*manifest.BuildOptions
	restore := image.MockManifestNewBuild(func(m *manifest.Manifest, r runner.Runner, repos []rpmmd.RepoConfig, opts *manifest.BuildOptions) manifest.Build {
		buildOpts = append(buildOpts, opts)
		return manifest.NewBuild(m, r, repos, opts)
	})
	defer restore()

	for _, containerBuildable := range []bool{true, false} {
		buildOpts = nil

		mf := manifest.New(manifest.DISTRO_FEDORA)
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

		require.Equal(t, len(buildOpts), 1)
		require.Equal(t, buildOpts[0].ContainerBuildable, containerBuildable)
	}
}
