package manifest_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/dnfjson"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/platform"
	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/osbuild/images/pkg/runner"
)

func TestAnacondaInstallerCustomLoraxTemplatePath(t *testing.T) {
	pkgs := []rpmmd.PackageSpec{
		{
			Name:     "kernel",
			Checksum: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		},
	}

	type testCase struct {
		name                     string
		customLoraxTemplatePath  string
		useRHELLoraxTemplates    bool
		expectedLoraxScriptPath  string
		expectedUseRHELTemplates bool
	}

	testCases := []testCase{
		{
			name:                     "custom-lorax-template",
			customLoraxTemplatePath:  "custom/my-template.tmpl",
			useRHELLoraxTemplates:    false, // Should be overridden by custom path
			expectedLoraxScriptPath:  "custom/my-template.tmpl",
			expectedUseRHELTemplates: false,
		},
		{
			name:                     "rhel-templates",
			customLoraxTemplatePath:  "",
			useRHELLoraxTemplates:    true,
			expectedLoraxScriptPath:  "80-rhel/runtime-postinstall.tmpl",
			expectedUseRHELTemplates: true,
		},
		{
			name:                     "generic-templates",
			customLoraxTemplatePath:  "",
			useRHELLoraxTemplates:    false,
			expectedLoraxScriptPath:  "99-generic/runtime-postinstall.tmpl",
			expectedUseRHELTemplates: false,
		},
		{
			name:                     "custom-overrides-rhel-flag",
			customLoraxTemplatePath:  "override/custom.tmpl",
			useRHELLoraxTemplates:    true, // Should be ignored when custom path is set
			expectedLoraxScriptPath:  "override/custom.tmpl",
			expectedUseRHELTemplates: true, // Flag value preserved but custom path takes precedence
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := &manifest.Manifest{}
			runner := &runner.Linux{}
			build := manifest.NewBuild(m, runner, nil, nil)
			x86plat := &platform.Data{Arch: arch.ARCH_X86_64}
			product := ""
			osversion := ""
			preview := false

			installer := manifest.NewAnacondaInstaller(manifest.AnacondaInstallerTypePayload, build, x86plat, nil, "kernel", product, osversion, preview)
			
			// Configure installer customizations
			installer.InstallerCustomizations.CustomLoraxTemplatePath = tc.customLoraxTemplatePath
			installer.InstallerCustomizations.UseRHELLoraxTemplates = tc.useRHELLoraxTemplates

			pipeline := manifest.SerializeWith(installer, manifest.Inputs{Depsolved: dnfjson.DepsolveResult{Packages: pkgs}})
			
			require := require.New(t)
			require.NotNil(pipeline)
			require.NotNil(pipeline.Stages)

			// Find the lorax-script stage
			var loraxScriptStage *osbuild.Stage
			for _, stage := range pipeline.Stages {
				if stage.Type == "org.osbuild.lorax-script" {
					loraxScriptStage = stage
					break
				}
			}

			require.NotNil(loraxScriptStage, "serialized anaconda pipeline does not contain an org.osbuild.lorax-script stage")
			
			// Check lorax script stage options
			loraxOptions, ok := loraxScriptStage.Options.(*osbuild.LoraxScriptStageOptions)
			require.True(ok, "lorax-script stage options are not of correct type")
			
			// Verify the correct lorax template path is used
			require.Equal(tc.expectedLoraxScriptPath, loraxOptions.Path, 
				"lorax template path should match expected value")
		})
	}
}

func TestAnacondaInstallerLoraxTemplatePathPriority(t *testing.T) {
	pkgs := []rpmmd.PackageSpec{
		{
			Name:     "kernel", 
			Checksum: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		},
	}

	m := &manifest.Manifest{}
	runner := &runner.Linux{}
	build := manifest.NewBuild(m, runner, nil, nil)
	x86plat := &platform.Data{Arch: arch.ARCH_X86_64}
	product := ""
	osversion := ""
	preview := false

	installer := manifest.NewAnacondaInstaller(manifest.AnacondaInstallerTypePayload, build, x86plat, nil, "kernel", product, osversion, preview)

	// Test that custom path takes priority over RHEL templates flag
	installer.InstallerCustomizations.CustomLoraxTemplatePath = "priority-test/custom.tmpl"
	installer.InstallerCustomizations.UseRHELLoraxTemplates = true // This should be ignored

	pipeline := manifest.SerializeWith(installer, manifest.Inputs{Depsolved: dnfjson.DepsolveResult{Packages: pkgs}})
	
	require := require.New(t)
	require.NotNil(pipeline)

	// Find the lorax-script stage
	var loraxScriptStage *osbuild.Stage
	for _, stage := range pipeline.Stages {
		if stage.Type == "org.osbuild.lorax-script" {
			loraxScriptStage = stage
			break
		}
	}

	require.NotNil(loraxScriptStage, "serialized anaconda pipeline does not contain an org.osbuild.lorax-script stage")

	loraxOptions, ok := loraxScriptStage.Options.(*osbuild.LoraxScriptStageOptions)
	require.True(ok, "lorax-script stage options are not of correct type")
	
	// Custom path should take priority
	require.Equal("priority-test/custom.tmpl", loraxOptions.Path,
		"custom lorax template path should take priority over UseRHELLoraxTemplates flag")
}

func TestAnacondaInstallerEmptyCustomLoraxTemplatePath(t *testing.T) {
	pkgs := []rpmmd.PackageSpec{
		{
			Name:     "kernel",
			Checksum: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		},
	}

	testCases := []struct {
		name                     string
		useRHELLoraxTemplates    bool
		expectedLoraxScriptPath  string
	}{
		{
			name:                     "empty-custom-path-with-rhel-templates",
			useRHELLoraxTemplates:    true,
			expectedLoraxScriptPath:  "80-rhel/runtime-postinstall.tmpl",
		},
		{
			name:                     "empty-custom-path-with-generic-templates",
			useRHELLoraxTemplates:    false,
			expectedLoraxScriptPath:  "99-generic/runtime-postinstall.tmpl",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := &manifest.Manifest{}
			runner := &runner.Linux{}
			build := manifest.NewBuild(m, runner, nil, nil)
			x86plat := &platform.Data{Arch: arch.ARCH_X86_64}
			product := ""
			osversion := ""
			preview := false

			installer := manifest.NewAnacondaInstaller(manifest.AnacondaInstallerTypePayload, build, x86plat, nil, "kernel", product, osversion, preview)
			
			// Empty custom path should fall back to UseRHELLoraxTemplates flag
			installer.InstallerCustomizations.CustomLoraxTemplatePath = ""
			installer.InstallerCustomizations.UseRHELLoraxTemplates = tc.useRHELLoraxTemplates

			pipeline := manifest.SerializeWith(installer, manifest.Inputs{Depsolved: dnfjson.DepsolveResult{Packages: pkgs}})
			
			require := require.New(t)
			require.NotNil(pipeline)

			// Find the lorax-script stage
			var loraxScriptStage *osbuild.Stage
			for _, stage := range pipeline.Stages {
				if stage.Type == "org.osbuild.lorax-script" {
					loraxScriptStage = stage
					break
				}
			}

			require.NotNil(loraxScriptStage, "serialized anaconda pipeline does not contain an org.osbuild.lorax-script stage")

			loraxOptions, ok := loraxScriptStage.Options.(*osbuild.LoraxScriptStageOptions)
			require.True(ok, "lorax-script stage options are not of correct type")
			
			// Should fall back to automatic template selection
			require.Equal(tc.expectedLoraxScriptPath, loraxOptions.Path,
				"should fall back to automatic lorax template selection when custom path is empty")
		})
	}
}
