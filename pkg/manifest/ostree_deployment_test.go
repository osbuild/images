package manifest_test

import (
	"testing"

	"github.com/osbuild/images/internal/testdisk"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/ostree"
	"github.com/osbuild/images/pkg/platform"
	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/osbuild/images/pkg/runner"
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
	pipeline.MountUnits = false                                    // set it explicitly just to be sure

	checkStagesForFSTab(t, pipeline.Serialize().Stages)
}

func TestOSTreeDeploymentPipelineMountUnitStages(t *testing.T) {
	pipeline := NewTestOSTreeDeployment()

	expectedUnits := []string{"-.mount", "home.mount"}
	pipeline.PartitionTable = testdisk.MakeFakePartitionTable("/", "/home")
	pipeline.MountUnits = true

	checkStagesForMountUnits(t, pipeline.Serialize().Stages, expectedUnits)
}
