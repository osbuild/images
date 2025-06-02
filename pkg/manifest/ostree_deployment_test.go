package manifest_test

import (
	"testing"

	"github.com/osbuild/images/internal/testdisk"
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/ostree"
	"github.com/osbuild/images/pkg/platform"
	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/osbuild/images/pkg/runner"
	"github.com/stretchr/testify/require"
)

// NewTestOSTreeDeployment returns a minimally populated OSTreeDeployment for
// use in testing
func NewTestOSTreeDeployment() *manifest.OSTreeDeployment {
	repos := []rpmmd.RepoConfig{}
	m := manifest.New()
	runner := &runner.Fedora{Version: 38}
	build := manifest.NewBuild(&m, runner, repos, nil)
	build.Checkpoint()

	// create an x86_64 platform with bios boot
	platform := &platform.X86{
		BIOS: true,
	}
	commit := &ostree.SourceSpec{}
	os := manifest.NewOSTreeCommitDeployment(build, commit, "fedora", platform)
	return os
}

func TestOSTreeDeploymentPipelineFStabStage(t *testing.T) {
	pipeline := NewTestOSTreeDeployment()

	pipeline.PartitionTable = testdisk.MakeFakePartitionTable("/") // PT specifics don't matter
	pipeline.GenerateMounts = blueprint.GenerateFstab

	checkStagesForFSTab(t, pipeline.Serialize().Stages)
}

func TestOSTreeDeploymentPipelineMountUnitStages(t *testing.T) {
	pipeline := NewTestOSTreeDeployment()

	expectedUnits := []string{"-.mount", "home.mount"}
	pipeline.PartitionTable = testdisk.MakeFakePartitionTable("/", "/home")
	pipeline.GenerateMounts = blueprint.GenerateUnits

	checkStagesForMountUnits(t, pipeline.Serialize().Stages, expectedUnits)
}

func TestOSTreeDeploymentPipelineNoMounts(t *testing.T) {
	pipeline := NewTestOSTreeDeployment()

	pipeline.PartitionTable = testdisk.MakeFakePartitionTable("/", "/home")
	pipeline.GenerateMounts = blueprint.GenerateNone

	checkStagesForNoMounts(t, pipeline.Serialize().Stages)
}

func TestAddInlineOSTreeDeployment(t *testing.T) {
	deployment := NewTestOSTreeDeployment()

	require := require.New(t)

	// add some files to the Files list which are included near the end of the
	// pipeline
	deployment.Files = createTestFilesForPipeline()

	// enabling FIPS adds files before the Files defined above
	deployment.FIPS = true

	expectedPaths := []string{
		"tree:///etc/system-fips", // from FIPS = true
		"tree:///etc/test/one",    // directly from the OS customizations
		"tree:///etc/test/two",
	}

	// the OSTreeDeployment pipeline *requires* a partition table
	deployment.PartitionTable = testdisk.MakeFakeBtrfsPartitionTable("/")
	pipeline := deployment.Serialize()

	destinationPaths := collectCopyDestinationPaths(pipeline.Stages)

	// The order is significant. Do not use ElementsMatch() or similar.
	require.Equal(expectedPaths, destinationPaths)

	expectedContents := []string{
		"test 1",
		"test 2",
		"# FIPS module installation complete\n",
	}

	fileContents := deployment.GetInline()
	// These are used to define the 'sources' part of the manifest, so the
	// order doesn't matter
	require.ElementsMatch(expectedContents, fileContents)
}
