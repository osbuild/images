package image

import (
	"fmt"
	"math/rand"

	"github.com/osbuild/images/pkg/artifact"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/platform"
	"github.com/osbuild/images/pkg/runner"
)

type BootcDiskImage struct {
	*OSTreeDiskImage
}

func NewBootcDiskImage(container container.SourceSpec) *BootcDiskImage {
	// XXX: hardcoded for now
	ref := "ostree/1/1/0"

	return &BootcDiskImage{
		&OSTreeDiskImage{
			Base:            NewBase("bootc-raw-image"),
			ContainerSource: &container,
			Ref:             ref,
			OSName:          "default",
		},
	}
}

func (img *BootcDiskImage) InstantiateManifestFromContainers(m *manifest.Manifest,
	containers []container.SourceSpec,
	runner runner.Runner,
	rng *rand.Rand) (*artifact.Artifact, error) {

	buildPipeline := manifest.NewBuildFromContainer(m, runner, containers, &manifest.BuildOptions{ContainerBuildable: true})
	buildPipeline.Checkpoint()

	// don't support compressing non-raw images
	imgFormat := img.Platform.GetImageFormat()
	if imgFormat == platform.FORMAT_UNSET {
		// treat unset as raw for this check
		imgFormat = platform.FORMAT_RAW
	}
	if imgFormat != platform.FORMAT_RAW && img.Compression != "" {
		panic(fmt.Sprintf("no compression is allowed with %q format for %q", imgFormat, img.name))
	}

	// In the bootc flow, we reuse the host container context for tools;
	// this is signified by passing nil to the below pipelines.
	var hostPipeline manifest.Build = nil

	baseImage := baseRawOstreeImage(img.OSTreeDiskImage, buildPipeline, &baseRawOstreeImageOpts{useBootupd: true})

	opts := &imagePipelineOpts{
		QCOW2Compat: img.Platform.GetQCOW2Compat(),
		Filename:    img.Filename,
	}
	imagePipeline := makeImagePipeline(img.Platform.GetImageFormat(), baseImage, hostPipeline, opts)
	compressionPipeline := makeCompressionPipeline(img.Compression, img.Filename, imagePipeline, buildPipeline)
	return compressionPipeline.Export(), nil
}
