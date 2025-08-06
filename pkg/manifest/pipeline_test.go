package manifest_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/runner"
)

func TestPipelineRoleBuild(t *testing.T) {
	var mf manifest.Manifest
	pipi := manifest.NewBuild(&mf, &runner.Linux{}, nil, nil)
	assert.Equal(t, manifest.PipelineRoleBuild, pipi.(manifest.Pipeline).Role())
	assert.Equal(t, []string{"build"}, mf.BuildPipelines())
	assert.Equal(t, 0, len(mf.PayloadPipelines()))
}

func TestPipelineRoleBuildFromContainer(t *testing.T) {
	var mf manifest.Manifest
	pipi := manifest.NewBuildFromContainer(&mf, &runner.Linux{}, nil, nil)
	assert.Equal(t, manifest.PipelineRoleBuild, pipi.(manifest.Pipeline).Role())
	assert.Equal(t, []string{"build"}, mf.BuildPipelines())
	assert.Equal(t, 0, len(mf.PayloadPipelines()))
}

func TestPipelineRolePayload(t *testing.T) {
	var mf manifest.Manifest

	bp := manifest.NewBuild(&mf, &runner.Linux{}, nil, nil)

	pipi := manifest.NewXZ(bp, nil)
	assert.Equal(t, manifest.PipelineRolePayload, pipi.Role())
	assert.Equal(t, []string{"xz"}, mf.PayloadPipelines())
	assert.Equal(t, []string{"build"}, mf.BuildPipelines())
}
