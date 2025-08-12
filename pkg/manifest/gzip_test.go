package manifest_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/runner"
)

func TestGzipSerialize(t *testing.T) {
	mani := manifest.New()
	runner := &runner.Linux{}
	build := manifest.NewBuild(&mani, runner, nil, nil)

	// setup
	rawImage := manifest.NewRawImage(build, nil)
	gzipPipeline := manifest.NewGzip(build, rawImage)
	gzipPipeline.SetFilename("filename.gz")

	// run
	osbuildPipeline := manifest.Serialize(gzipPipeline)

	// assert
	assert.Equal(t, "gzip", osbuildPipeline.Name)
	assert.Equal(t, 1, len(osbuildPipeline.Stages))
	gzipStage := osbuildPipeline.Stages[0]
	assert.Equal(t, &osbuild.GzipStageOptions{
		Filename: "filename.gz",
	}, gzipStage.Options.(*osbuild.GzipStageOptions))
}
