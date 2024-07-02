package manifest

import (
	"testing"

	"github.com/osbuild/images/pkg/customizations/anaconda"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/platform"
	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/osbuild/images/pkg/runner"
	"github.com/stretchr/testify/require"
)

func newAnacondaInstaller() *AnacondaInstaller {
	m := &IntManifest{}
	runner := &runner.Linux{}
	build := NewBuild(m, runner, nil, nil)

	x86plat := &platform.X86{}

	product := ""
	osversion := ""

	preview := false

	installer := NewAnacondaInstaller(AnacondaInstallerTypePayload, build, x86plat, nil, "kernel", product, osversion, preview)
	return installer
}

func TestAnacondaInstallerModules(t *testing.T) {
	pkgs := []rpmmd.PackageSpec{
		{
			Name:     "kernel",
			Checksum: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		},
	}
	type testCase struct {
		enable   []string
		disable  []string
		expected []string
	}

	testCases := map[string]testCase{
		"empty-args": {
			expected: []string{
				anaconda.ModulePayloads,
				anaconda.ModuleNetwork,
				anaconda.ModuleStorage,
			},
		},
		"no-op": {
			enable: []string{
				anaconda.ModulePayloads,
				anaconda.ModuleNetwork,
				anaconda.ModuleStorage,
			},
			expected: []string{
				anaconda.ModulePayloads,
				anaconda.ModuleNetwork,
				anaconda.ModuleStorage,
			},
		},
		"enable-users": {
			enable: []string{
				anaconda.ModuleUsers,
			},
			expected: []string{
				anaconda.ModulePayloads,
				anaconda.ModuleNetwork,
				anaconda.ModuleStorage,
				anaconda.ModuleUsers,
			},
		},
		"disable-storage": {
			disable: []string{
				anaconda.ModuleStorage,
			},
			expected: []string{
				anaconda.ModulePayloads,
				anaconda.ModuleNetwork,
			},
		},
		"enable-users-disable-storage": {
			enable: []string{
				anaconda.ModuleUsers,
			},
			disable: []string{
				anaconda.ModuleStorage,
			},
			expected: []string{
				anaconda.ModulePayloads,
				anaconda.ModuleNetwork,
				anaconda.ModuleUsers,
			},
		},
	}

	for name := range testCases {
		tc := testCases[name]
		// Run each test case twice: once with activatable-modules and once with kickstart-modules.
		// Remove this when we drop support for RHEL 8.
		t.Run(name, func(t *testing.T) {
			for _, legacy := range []bool{true, false} {
				installerPipeline := newAnacondaInstaller()
				installerPipeline.UseLegacyAnacondaConfig = legacy
				installerPipeline.AdditionalAnacondaModules = tc.enable
				installerPipeline.DisabledAnacondaModules = tc.disable
				installerPipeline.serializeStart(pkgs, nil, nil, nil)
				pipeline := installerPipeline.serialize()

				require := require.New(t)
				require.NotNil(pipeline)
				require.NotNil(pipeline.Stages)

				var anacondaStageOptions *osbuild.AnacondaStageOptions
				for _, stage := range pipeline.Stages {
					if stage.Type == "org.osbuild.anaconda" {
						anacondaStageOptions = stage.Options.(*osbuild.AnacondaStageOptions)
					}
				}

				require.NotNil(anacondaStageOptions, "serialized anaconda pipeline does not contain an org.osbuild.anaconda stage")
				if legacy {
					require.ElementsMatch(anacondaStageOptions.KickstartModules, tc.expected)
					require.Empty(anacondaStageOptions.ActivatableModules)
				} else {
					require.ElementsMatch(anacondaStageOptions.ActivatableModules, tc.expected)
					require.Empty(anacondaStageOptions.KickstartModules)
				}
			}
		})
	}
}
