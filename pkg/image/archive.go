package image

import (
	"fmt"
	"math/rand"

	"github.com/osbuild/images/internal/environment"
	"github.com/osbuild/images/internal/workload"
	"github.com/osbuild/images/pkg/artifact"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/platform"
	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/osbuild/images/pkg/runner"
)

type Archive struct {
	Base
	Platform         platform.Platform
	OSCustomizations manifest.OSCustomizations
	Environment      environment.Environment
	Workload         workload.Workload
	Filename         string
	Compression      string

	OSVersion string
}

func NewArchive() *Archive {
	return &Archive{
		Base: NewBase("archive"),
	}
}

func (img *Archive) InstantiateManifest(m *manifest.Manifest,
	repos []rpmmd.RepoConfig,
	runner runner.Runner,
	rng *rand.Rand) (*artifact.Artifact, error) {
	buildPipeline := addBuildBootstrapPipelines(m, runner, repos, nil)
	buildPipeline.Checkpoint()

	osPipeline := manifest.NewOS(buildPipeline, img.Platform, repos)
	osPipeline.OSCustomizations = img.OSCustomizations
	osPipeline.Environment = img.Environment
	osPipeline.Workload = img.Workload
	osPipeline.OSVersion = img.OSVersion

	tarPipeline := manifest.NewTar(buildPipeline, osPipeline, "archive")
	tarPipeline.SetFilename(img.Filename)

	switch img.Compression {
	case "xz":
		xzPipeline := manifest.NewXZ(buildPipeline, tarPipeline)
		xzPipeline.SetFilename(img.Filename)
		return xzPipeline.Export(), nil
	case "zstd":
		zstdPipeline := manifest.NewZstd(buildPipeline, tarPipeline)
		zstdPipeline.SetFilename(img.Filename)
		return zstdPipeline.Export(), nil
	case "gzip":
		gzipPipeline := manifest.NewGzip(buildPipeline, tarPipeline)
		gzipPipeline.SetFilename(img.Filename)
		return gzipPipeline.Export(), nil
	case "":
		// don't compress, but make sure the pipeline's filename is set
		tarPipeline.SetFilename(img.Filename)
		return tarPipeline.Export(), nil
	default:
		// panic on unknown strings
		panic(fmt.Sprintf("unsupported compression type %q", img.Compression))
	}
}
