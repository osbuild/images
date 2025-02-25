package manifest_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/internal/testdisk"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/customizations/bootc"
	"github.com/osbuild/images/pkg/customizations/subscription"
	"github.com/osbuild/images/pkg/dnfjson"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/rpmmd"
)

func findStages(name string, stages []*osbuild.Stage) []*osbuild.Stage {
	var foundStages []*osbuild.Stage
	for _, s := range stages {
		if s.Type == name {
			foundStages = append(foundStages, s)
		}
	}
	return foundStages
}

// CheckSystemdStageOptions checks the Command strings
func CheckSystemdStageOptions(t *testing.T, stages []*osbuild.Stage, commands []string) {
	// Find the systemd.unit.create stage
	s := manifest.FindStage("org.osbuild.systemd.unit.create", stages)
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
	os := manifest.NewTestOS()
	os.Subscription = &subscription.ImageOptions{
		Organization:  "2040324",
		ActivationKey: "my-secret-key",
		ServerUrl:     "subscription.rhsm.redhat.com",
		BaseUrl:       "http://cdn.redhat.com/",
	}
	pipeline := os.Serialize()
	CheckSystemdStageOptions(t, pipeline.Stages, []string{
		"/usr/sbin/subscription-manager register --org=${ORG_ID} --activationkey=${ACTIVATION_KEY} --serverurl subscription.rhsm.redhat.com --baseurl http://cdn.redhat.com/",
	})
}

func TestSubscriptionManagerInsightsCommands(t *testing.T) {
	os := manifest.NewTestOS()
	os.Subscription = &subscription.ImageOptions{
		Organization:  "2040324",
		ActivationKey: "my-secret-key",
		ServerUrl:     "subscription.rhsm.redhat.com",
		BaseUrl:       "http://cdn.redhat.com/",
		Insights:      true,
	}
	pipeline := os.Serialize()
	CheckSystemdStageOptions(t, pipeline.Stages, []string{
		"/usr/sbin/subscription-manager register --org=${ORG_ID} --activationkey=${ACTIVATION_KEY} --serverurl subscription.rhsm.redhat.com --baseurl http://cdn.redhat.com/",
		"/usr/bin/insights-client --register",
		"restorecon -R /root/.gnupg",
	})
}

func TestRhcInsightsCommands(t *testing.T) {
	os := manifest.NewTestOS()
	os.Subscription = &subscription.ImageOptions{
		Organization:  "2040324",
		ActivationKey: "my-secret-key",
		ServerUrl:     "subscription.rhsm.redhat.com",
		BaseUrl:       "http://cdn.redhat.com/",
		Insights:      false,
		Rhc:           true,
	}
	pipeline := os.Serialize()
	CheckSystemdStageOptions(t, pipeline.Stages, []string{
		"/usr/bin/rhc connect --organization=${ORG_ID} --activation-key=${ACTIVATION_KEY} --server subscription.rhsm.redhat.com",
		"restorecon -R /root/.gnupg",
		"/usr/sbin/semanage permissive --add rhcd_t",
	})
}

func TestSubscriptionManagerPackages(t *testing.T) {
	os := manifest.NewTestOS()
	os.Subscription = &subscription.ImageOptions{
		Organization:  "2040324",
		ActivationKey: "my-secret-key",
		ServerUrl:     "subscription.rhsm.redhat.com",
		BaseUrl:       "http://cdn.redhat.com/",
	}

	CheckPkgSetInclude(t, os.GetPackageSetChain(manifest.DISTRO_NULL), []string{"subscription-manager"})
}

func TestSubscriptionManagerInsightsPackages(t *testing.T) {
	os := manifest.NewTestOS()
	os.Subscription = &subscription.ImageOptions{
		Organization:  "2040324",
		ActivationKey: "my-secret-key",
		ServerUrl:     "subscription.rhsm.redhat.com",
		BaseUrl:       "http://cdn.redhat.com/",
		Insights:      true,
	}
	CheckPkgSetInclude(t, os.GetPackageSetChain(manifest.DISTRO_NULL), []string{"subscription-manager", "insights-client"})
}

func TestRhcInsightsPackages(t *testing.T) {
	os := manifest.NewTestOS()
	os.Subscription = &subscription.ImageOptions{
		Organization:  "2040324",
		ActivationKey: "my-secret-key",
		ServerUrl:     "subscription.rhsm.redhat.com",
		BaseUrl:       "http://cdn.redhat.com/",
		Insights:      false,
		Rhc:           true,
	}
	CheckPkgSetInclude(t, os.GetPackageSetChain(manifest.DISTRO_NULL), []string{"rhc", "subscription-manager", "insights-client"})
}

func TestBootupdStage(t *testing.T) {
	os := manifest.NewTestOS()
	os.OSTreeRef = "some/ref"
	os.Bootupd = true
	pipeline := os.Serialize()
	st := manifest.FindStage("org.osbuild.bootupd.gen-metadata", pipeline.Stages)
	require.NotNil(t, st)
}

func TestTomlLibUsedNoneByDefault(t *testing.T) {
	os := manifest.NewTestOS()
	buildPkgs := os.GetBuildPackages(manifest.DISTRO_FEDORA)
	for _, pkg := range []string{"python3-pytoml", "python3-toml", "python3-tomli-w"} {
		assert.NotContains(t, buildPkgs, pkg)
	}
}

func TestTomlLibUsedForContainer(t *testing.T) {
	os := manifest.NewTestOS()
	os.OSCustomizations.Containers = []container.SourceSpec{
		{Source: "some-source"},
	}
	os.OSCustomizations.ContainersStorage = common.ToPtr("foo")

	testTomlPkgsFor(t, os)
}

func TestTomlLibUsedForBootcConfig(t *testing.T) {
	os := manifest.NewTestOS()
	os.BootcConfig = &bootc.Config{Filename: "something"}

	testTomlPkgsFor(t, os)
}

func testTomlPkgsFor(t *testing.T, os *manifest.OS) {
	for _, tc := range []struct {
		distro          manifest.Distro
		expectedTomlPkg string
	}{
		{manifest.DISTRO_EL8, "python3-pytoml"},
		{manifest.DISTRO_EL9, "python3-toml"},
		{manifest.DISTRO_EL10, "python3-tomli-w"},
		{manifest.DISTRO_FEDORA, "python3-tomli-w"},
	} {
		buildPkgs := os.GetBuildPackages(tc.distro)
		assert.Contains(t, buildPkgs, tc.expectedTomlPkg)
	}
}

func TestMachineIdUninitializedIncludesMachineIdStage(t *testing.T) {
	os := manifest.NewTestOS()

	os.MachineIdUninitialized = true

	pipeline := os.Serialize()
	st := manifest.FindStage("org.osbuild.machine-id", pipeline.Stages)
	require.NotNil(t, st)
}

func TestMachineIdUninitializedDoesNotIncludeMachineIdStage(t *testing.T) {
	os := manifest.NewTestOS()

	pipeline := os.Serialize()
	st := manifest.FindStage("org.osbuild.machine-id", pipeline.Stages)
	require.Nil(t, st)
}

func TestModularityIncludesConfigStage(t *testing.T) {
	os := manifest.NewTestOS()

	testModuleConfigPath := filepath.Join(t.TempDir(), "module-config")
	testFailsafeConfigPath := filepath.Join(t.TempDir(), "failsafe-config")

	inputs := manifest.Inputs{
		Depsolved: dnfjson.DepsolveResult{
			Packages: []rpmmd.PackageSpec{
				{Name: "pkg1", Checksum: "sha1:c02524e2bd19490f2a7167958f792262754c5f46"},
			},
			Modules: []rpmmd.ModuleSpec{
				{
					ModuleConfigFile: rpmmd.ModuleConfigFile{
						Path: testModuleConfigPath,
					},
					FailsafeFile: rpmmd.ModuleFailsafeFile{
						Path: testFailsafeConfigPath,
					},
				},
			}},
	}
	pipeline := os.SerializeWith(inputs)
	st := manifest.FindStage("org.osbuild.dnf.module-config", pipeline.Stages)
	require.NotNil(t, st)
}

func TestModularityDoesNotIncludeConfigStage(t *testing.T) {
	os := manifest.NewTestOS()

	pipeline := os.Serialize()
	st := manifest.FindStage("org.osbuild.dnf.module-config", pipeline.Stages)
	require.Nil(t, st)
}

func checkStagesForFSTab(t *testing.T, stages []*osbuild.Stage) {
	fstab := manifest.FindStage("org.osbuild.fstab", stages)
	require.NotNil(t, fstab)

	// The plain OS pipeline doesn't have any systemd.unit.create stages by
	// default. This test will break and will need to be adjusted if this ever
	// changes (if a systemd.unit.create stage is added to the pipeline by
	// default).
	systemdStages := findStages("org.osbuild.systemd.unit.create", stages)
	require.Nil(t, systemdStages)
}

func checkStagesForMountUnits(t *testing.T, stages []*osbuild.Stage, expectedUnits []string) {
	fstab := manifest.FindStage("org.osbuild.fstab", stages)
	require.Nil(t, fstab)

	// The plain OS pipeline doesn't have any systemd.unit.create stages by
	// default. This test will break and will need to be adjusted if this ever
	// changes (if a systemd.unit.create stage is added to the pipeline by
	// default).
	systemdStages := findStages("org.osbuild.systemd.unit.create", stages)
	require.Len(t, systemdStages, len(expectedUnits))

	var mountUnitFilenames []string
	for _, stage := range systemdStages {
		options := stage.Options.(*osbuild.SystemdUnitCreateStageOptions)
		mountUnitFilenames = append(mountUnitFilenames, options.Filename)
	}
	require.ElementsMatch(t, mountUnitFilenames, expectedUnits)

	// creating mount units also adds a systemd stage to enable them
	enable := manifest.FindStage("org.osbuild.systemd", stages)
	require.NotNil(t, enable)
	enableOptions := enable.Options.(*osbuild.SystemdStageOptions)
	require.ElementsMatch(t, enableOptions.EnabledServices, expectedUnits)

}

func TestOSPipelineFStabStage(t *testing.T) {
	os := manifest.NewTestOS()

	os.PartitionTable = testdisk.MakeFakePartitionTable("/") // PT specifics don't matter
	os.MountUnits = false                                    // set it explicitly just to be sure

	checkStagesForFSTab(t, os.Serialize().Stages)
}

func TestOSPipelineMountUnitStages(t *testing.T) {
	os := manifest.NewTestOS()

	expectedUnits := []string{"-.mount", "home.mount"}
	os.PartitionTable = testdisk.MakeFakePartitionTable("/", "/home")
	os.MountUnits = true

	checkStagesForMountUnits(t, os.Serialize().Stages, expectedUnits)
}

func TestLanguageIncludesLocaleStage(t *testing.T) {
	os := manifest.NewTestOS()

	os.Language = "en_US.UTF-8"

	pipeline := os.Serialize()
	st := manifest.FindStage("org.osbuild.locale", pipeline.Stages)
	require.NotNil(t, st)
}

func TestLanguageDoesNotIncludeLocaleStage(t *testing.T) {
	os := manifest.NewTestOS()

	pipeline := os.Serialize()
	st := manifest.FindStage("org.osbuild.locale", pipeline.Stages)
	require.Nil(t, st)
}

func TestTimezoneIncludesTimezoneStage(t *testing.T) {
	os := manifest.NewTestOS()

	os.Timezone = "Etc/UTC"

	pipeline := os.Serialize()
	st := manifest.FindStage("org.osbuild.timezone", pipeline.Stages)
	require.NotNil(t, st)
}

func TestTimezoneDoesNotIncludeTimezoneStage(t *testing.T) {
	os := manifest.NewTestOS()

	pipeline := os.Serialize()
	st := manifest.FindStage("org.osbuild.timezone", pipeline.Stages)
	require.Nil(t, st)
}

func TestHostnameIncludesHostnameStage(t *testing.T) {
	os := manifest.NewTestOS()

	os.Hostname = "funky.name"

	pipeline := os.Serialize()
	st := manifest.FindStage("org.osbuild.hostname", pipeline.Stages)
	require.NotNil(t, st)
}

func TestHostnameDoesNotIncludeHostnameStage(t *testing.T) {
	os := manifest.NewTestOS()

	pipeline := os.Serialize()
	st := manifest.FindStage("org.osbuild.hostname", pipeline.Stages)
	require.Nil(t, st)
}
