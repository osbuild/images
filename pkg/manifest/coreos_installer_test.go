package manifest

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/depsolvednf"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/platform"
	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/osbuild/images/pkg/runner"
)

func newCoreOSInstaller() *CoreOSInstaller {
	m := &Manifest{}
	runner := &runner.Linux{}
	build := NewBuild(m, runner, nil, nil)

	x86plat := &platform.Data{Arch: arch.ARCH_X86_64}

	product := ""
	osversion := ""

	installer := NewCoreOSInstaller(build, x86plat, nil, "kernel", product, osversion)
	return installer
}

func TestCoreOSInstallerDracutModulesAndDrivers(t *testing.T) {
	pkgs := []rpmmd.PackageSpec{
		{
			Name:     "kernel",
			Checksum: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		},
	}

	coiPipeline := newCoreOSInstaller()
	coiPipeline.AdditionalDracutModules = []string{"test-module"}
	coiPipeline.AdditionalDrivers = []string{"test-driver"}
	coiPipeline.serializeStart(Inputs{Depsolved: depsolvednf.DepsolveResult{Packages: pkgs}})
	pipeline := coiPipeline.serialize()

	require := require.New(t)
	require.NotNil(pipeline)
	require.NotNil(pipeline.Stages)

	var stageOptions *osbuild.DracutStageOptions
	for _, stage := range pipeline.Stages {
		if stage.Type == "org.osbuild.dracut" {
			stageOptions = stage.Options.(*osbuild.DracutStageOptions)
		}
	}

	require.NotNil(stageOptions, "serialized anaconda pipeline does not contain an org.osbuild.anaconda stage")
	require.Contains(stageOptions.Modules, "test-module")
	require.Contains(stageOptions.AddDrivers, "test-driver")
}
