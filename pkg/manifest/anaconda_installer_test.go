package manifest

import (
	"testing"

	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/platform"
	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/osbuild/images/pkg/runner"
	"github.com/stretchr/testify/require"
)

func newAnacondaInstaller() *AnacondaInstaller {
	m := &Manifest{}
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
				"org.fedoraproject.Anaconda.Modules.Payloads",
				"org.fedoraproject.Anaconda.Modules.Network",
				"org.fedoraproject.Anaconda.Modules.Storage",
			},
		},
		"no-op": {
			enable: []string{
				"org.fedoraproject.Anaconda.Modules.Payloads",
				"org.fedoraproject.Anaconda.Modules.Network",
				"org.fedoraproject.Anaconda.Modules.Storage",
			},
			expected: []string{
				"org.fedoraproject.Anaconda.Modules.Payloads",
				"org.fedoraproject.Anaconda.Modules.Network",
				"org.fedoraproject.Anaconda.Modules.Storage",
			},
		},
		"enable-users": {
			enable: []string{
				"org.fedoraproject.Anaconda.Modules.Users",
			},
			expected: []string{
				"org.fedoraproject.Anaconda.Modules.Payloads",
				"org.fedoraproject.Anaconda.Modules.Network",
				"org.fedoraproject.Anaconda.Modules.Storage",
				"org.fedoraproject.Anaconda.Modules.Users",
			},
		},
		"disable-storage": {
			disable: []string{
				"org.fedoraproject.Anaconda.Modules.Storage",
			},
			expected: []string{
				"org.fedoraproject.Anaconda.Modules.Payloads",
				"org.fedoraproject.Anaconda.Modules.Network",
			},
		},
		"enable-users-disable-storage": {
			enable: []string{
				"org.fedoraproject.Anaconda.Modules.Users",
			},
			disable: []string{
				"org.fedoraproject.Anaconda.Modules.Storage",
			},
			expected: []string{
				"org.fedoraproject.Anaconda.Modules.Payloads",
				"org.fedoraproject.Anaconda.Modules.Network",
				"org.fedoraproject.Anaconda.Modules.Users",
			},
		},
	}

	for name := range testCases {
		tc := testCases[name]
		t.Run(name, func(t *testing.T) {
			installerPipeline := newAnacondaInstaller()
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
			require.ElementsMatch(anacondaStageOptions.KickstartModules, tc.expected)
		})
	}
}
