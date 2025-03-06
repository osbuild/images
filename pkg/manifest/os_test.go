package manifest

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/customizations/bootc"
	"github.com/osbuild/images/pkg/customizations/subscription"
	"github.com/osbuild/images/pkg/dnfjson"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/platform"
	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/osbuild/images/pkg/runner"
)

// NewTestOS returns a minimally populated OS struct for use in testing
func NewTestOS() *OS {
	repos := []rpmmd.RepoConfig{}
	manifest := New()
	runner := &runner.Fedora{Version: 38}
	build := NewBuild(&manifest, runner, repos, nil)
	build.Checkpoint()

	// create an x86_64 platform with bios boot
	platform := &platform.X86{
		BIOS: true,
	}

	os := NewOS(build, platform, repos)
	packages := []rpmmd.PackageSpec{
		{Name: "pkg1", Checksum: "sha1:c02524e2bd19490f2a7167958f792262754c5f46"},
	}
	os.serializeStart(Inputs{
		Depsolved: dnfjson.DepsolveResult{
			Packages: packages,
			Repos:    repos,
		},
	})

	return os
}

func findStage(name string, stages []*osbuild.Stage) *osbuild.Stage {
	for _, s := range stages {
		if s.Type == name {
			return s
		}
	}
	return nil
}

// CheckSystemdStageOptions checks the Command strings
func CheckSystemdStageOptions(t *testing.T, stages []*osbuild.Stage, commands []string) {
	// Find the systemd.unit.create stage
	s := findStage("org.osbuild.systemd.unit.create", stages)
	require.NotNil(t, s)

	require.NotNil(t, s.Options)
	options, ok := s.Options.(*osbuild.SystemdUnitCreateStageOptions)
	require.True(t, ok)

	// unit must be conditioned on the keyfile
	require.Len(t, options.Config.Unit.ConditionPathExists, 1)
	keyfile := options.Config.Unit.ConditionPathExists[0]
	// keyfile is also the EnvironmentFile
	require.Len(t, options.Config.Service.EnvironmentFile, 1)
	assert.Equal(t, keyfile, options.Config.Service.EnvironmentFile[0])

	execStart := options.Config.Service.ExecStart
	// the rm command gets prepended in every case
	commands = append(commands, fmt.Sprintf("/usr/bin/rm %s", keyfile))
	require.Equal(t, len(execStart), len(commands))

	// Make sure the commands are the same
	for idx, cmd := range commands {
		assert.Equal(t, cmd, options.Config.Service.ExecStart[idx])
	}
}

// CheckPkgSetInclude makes sure the packages named in pkgs are all included
func CheckPkgSetInclude(t *testing.T, pkgSetChain []rpmmd.PackageSet, pkgs []string) {

	// Gather up all the includes
	var includes []string
	for _, ps := range pkgSetChain {
		includes = append(includes, ps.Include...)
	}

	for _, p := range pkgs {
		assert.Contains(t, includes, p)
	}
}

func TestSubscriptionManagerCommands(t *testing.T) {
	os := NewTestOS()
	os.Subscription = &subscription.ImageOptions{
		Organization:  "2040324",
		ActivationKey: "my-secret-key",
		ServerUrl:     "subscription.rhsm.redhat.com",
		BaseUrl:       "http://cdn.redhat.com/",
	}
	pipeline := os.serialize()
	CheckSystemdStageOptions(t, pipeline.Stages, []string{
		"/usr/sbin/subscription-manager register --org=${ORG_ID} --activationkey=${ACTIVATION_KEY} --serverurl subscription.rhsm.redhat.com --baseurl http://cdn.redhat.com/",
	})
}

func TestSubscriptionManagerInsightsCommands(t *testing.T) {
	os := NewTestOS()
	os.Subscription = &subscription.ImageOptions{
		Organization:  "2040324",
		ActivationKey: "my-secret-key",
		ServerUrl:     "subscription.rhsm.redhat.com",
		BaseUrl:       "http://cdn.redhat.com/",
		Insights:      true,
	}
	pipeline := os.serialize()
	CheckSystemdStageOptions(t, pipeline.Stages, []string{
		"/usr/sbin/subscription-manager register --org=${ORG_ID} --activationkey=${ACTIVATION_KEY} --serverurl subscription.rhsm.redhat.com --baseurl http://cdn.redhat.com/",
		"/usr/bin/insights-client --register",
		"restorecon -R /root/.gnupg",
	})
}

func TestRhcInsightsCommands(t *testing.T) {
	os := NewTestOS()
	os.Subscription = &subscription.ImageOptions{
		Organization:  "2040324",
		ActivationKey: "my-secret-key",
		ServerUrl:     "subscription.rhsm.redhat.com",
		BaseUrl:       "http://cdn.redhat.com/",
		Insights:      false,
		Rhc:           true,
	}
	pipeline := os.serialize()
	CheckSystemdStageOptions(t, pipeline.Stages, []string{
		"/usr/bin/rhc connect --organization=${ORG_ID} --activation-key=${ACTIVATION_KEY} --server subscription.rhsm.redhat.com",
		"restorecon -R /root/.gnupg",
		"/usr/sbin/semanage permissive --add rhcd_t",
	})
}

func TestSubscriptionManagerPackages(t *testing.T) {
	os := NewTestOS()
	os.Subscription = &subscription.ImageOptions{
		Organization:  "2040324",
		ActivationKey: "my-secret-key",
		ServerUrl:     "subscription.rhsm.redhat.com",
		BaseUrl:       "http://cdn.redhat.com/",
	}

	CheckPkgSetInclude(t, os.getPackageSetChain(DISTRO_NULL), []string{"subscription-manager"})
}

func TestSubscriptionManagerInsightsPackages(t *testing.T) {
	os := NewTestOS()
	os.Subscription = &subscription.ImageOptions{
		Organization:  "2040324",
		ActivationKey: "my-secret-key",
		ServerUrl:     "subscription.rhsm.redhat.com",
		BaseUrl:       "http://cdn.redhat.com/",
		Insights:      true,
	}
	CheckPkgSetInclude(t, os.getPackageSetChain(DISTRO_NULL), []string{"subscription-manager", "insights-client"})
}

func TestRhcInsightsPackages(t *testing.T) {
	os := NewTestOS()
	os.Subscription = &subscription.ImageOptions{
		Organization:  "2040324",
		ActivationKey: "my-secret-key",
		ServerUrl:     "subscription.rhsm.redhat.com",
		BaseUrl:       "http://cdn.redhat.com/",
		Insights:      false,
		Rhc:           true,
	}
	CheckPkgSetInclude(t, os.getPackageSetChain(DISTRO_NULL), []string{"rhc", "subscription-manager", "insights-client"})
}

func TestBootupdStage(t *testing.T) {
	os := NewTestOS()
	os.OSTreeRef = "some/ref"
	os.Bootupd = true
	pipeline := os.serialize()
	st := findStage("org.osbuild.bootupd.gen-metadata", pipeline.Stages)
	require.NotNil(t, st)
}

func TestTomlLibUsedNoneByDefault(t *testing.T) {
	os := NewTestOS()
	buildPkgs := os.getBuildPackages(DISTRO_FEDORA)
	for _, pkg := range []string{"python3-pytoml", "python3-toml", "python3-tomli-w"} {
		assert.NotContains(t, buildPkgs, pkg)
	}
}

func TestTomlLibUsedForContainer(t *testing.T) {
	os := NewTestOS()
	os.OSCustomizations.Containers = []container.SourceSpec{
		{Source: "some-source"},
	}
	os.OSCustomizations.ContainersStorage = common.ToPtr("foo")

	testTomlPkgsFor(t, os)
}

func TestTomlLibUsedForBootcConfig(t *testing.T) {
	os := NewTestOS()
	os.BootcConfig = &bootc.Config{Filename: "something"}

	testTomlPkgsFor(t, os)
}

func testTomlPkgsFor(t *testing.T, os *OS) {
	for _, tc := range []struct {
		distro          Distro
		expectedTomlPkg string
	}{
		{DISTRO_EL8, "python3-pytoml"},
		{DISTRO_EL9, "python3-toml"},
		{DISTRO_EL10, "python3-tomli-w"},
		{DISTRO_FEDORA, "python3-tomli-w"},
	} {
		buildPkgs := os.getBuildPackages(tc.distro)
		assert.Contains(t, buildPkgs, tc.expectedTomlPkg)
	}
}

func TestMachineIdUninitializedIncludesMachineIdStage(t *testing.T) {
	os := NewTestOS()

	os.MachineIdUninitialized = true

	pipeline := os.serialize()
	st := findStage("org.osbuild.machine-id", pipeline.Stages)
	require.NotNil(t, st)
}

func TestMachineIdUninitializedDoesNotIncludeMachineIdStage(t *testing.T) {
	os := NewTestOS()

	pipeline := os.serialize()
	st := findStage("org.osbuild.machine-id", pipeline.Stages)
	require.Nil(t, st)
}

func TestModularityIncludesConfigStage(t *testing.T) {
	os := NewTestOS()

	testModuleConfigPath := filepath.Join(t.TempDir(), "module-config")
	testFailsafeConfigPath := filepath.Join(t.TempDir(), "failsafe-config")

	os.moduleSpecs = []rpmmd.ModuleSpec{
		{
			ModuleConfigFile: rpmmd.ModuleConfigFile{
				Path: testModuleConfigPath,
			},
			FailsafeFile: rpmmd.ModuleFailsafeFile{
				Path: testFailsafeConfigPath,
			},
		},
	}
	pipeline := os.serialize()
	st := findStage("org.osbuild.dnf.module-config", pipeline.Stages)
	require.NotNil(t, st)
}

func TestModularityDoesNotIncludeConfigStage(t *testing.T) {
	os := NewTestOS()

	pipeline := os.serialize()
	st := findStage("org.osbuild.dnf.module-config", pipeline.Stages)
	require.Nil(t, st)
}
