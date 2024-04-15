package manifest_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/internal/testdisk"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/customizations/users"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/runner"
)

func hasPipeline(haystack []manifest.Pipeline, needle manifest.Pipeline) bool {
	for _, p := range haystack {
		if p == needle {
			return true
		}
	}
	return false
}

func TestNewRawBootcImage(t *testing.T) {
	mani := manifest.New()
	runner := &runner.Linux{}
	buildIf := manifest.NewBuildFromContainer(&mani, runner, nil, nil)
	build := buildIf.(*manifest.BuildrootFromContainer)

	rawBootcPipeline := manifest.NewRawBootcImage(build, nil, nil)
	require.NotNil(t, rawBootcPipeline)

	assert.True(t, hasPipeline(build.Dependents(), rawBootcPipeline))

	// disk.img is hardcoded for filename
	assert.Equal(t, "disk.img", rawBootcPipeline.Filename())
}

func TestRawBootcImageSerialize(t *testing.T) {
	mani := manifest.New()
	runner := &runner.Linux{}
	build := manifest.NewBuildFromContainer(&mani, runner, nil, nil)

	rawBootcPipeline := manifest.NewRawBootcImage(build, nil, nil)
	rawBootcPipeline.PartitionTable = testdisk.MakeFakePartitionTable("/", "/boot", "/boot/efi")
	rawBootcPipeline.Users = []users.User{{Name: "root", Key: common.ToPtr("some-ssh-key")}}
	rawBootcPipeline.KernelOptionsAppend = []string{"karg1", "karg2"}

	rawBootcPipeline.SerializeStart(nil, []container.Spec{{Source: "foo"}}, nil, nil)
	imagePipeline := rawBootcPipeline.Serialize()
	assert.Equal(t, "image", imagePipeline.Name)

	bootcInst := manifest.FindStage("org.osbuild.bootc.install-to-filesystem", imagePipeline.Stages)
	require.NotNil(t, bootcInst)
	opts := bootcInst.Options.(*osbuild.BootcInstallToFilesystemOptions)
	// Note that the root account is customized via the "users" stage
	// (mostly for uniformity)
	assert.Equal(t, len(opts.RootSSHAuthorizedKeys), 0)
	assert.Equal(t, []string{"karg1", "karg2"}, opts.Kargs)
}

func TestRawBootcImageSerializeMountsValidated(t *testing.T) {
	mani := manifest.New()
	runner := &runner.Linux{}
	build := manifest.NewBuildFromContainer(&mani, runner, nil, nil)

	rawBootcPipeline := manifest.NewRawBootcImage(build, nil, nil)
	// note that we create a partition table without /boot here
	rawBootcPipeline.PartitionTable = testdisk.MakeFakePartitionTable("/", "/missing-boot")
	rawBootcPipeline.SerializeStart(nil, []container.Spec{{Source: "foo"}}, nil, nil)
	assert.PanicsWithError(t, `required mounts for bootupd stage [/boot /boot/efi] missing`, func() {
		rawBootcPipeline.Serialize()
	})
}

func findMountIdx(mounts []osbuild.Mount, mntType string) int {
	for i, mnt := range mounts {
		if mnt.Type == mntType {
			return i
		}
	}
	return -1
}

func makeFakeRawBootcPipeline() *manifest.RawBootcImage {
	mani := manifest.New()
	runner := &runner.Linux{}
	build := manifest.NewBuildFromContainer(&mani, runner, nil, nil)
	rawBootcPipeline := manifest.NewRawBootcImage(build, nil, nil)
	rawBootcPipeline.PartitionTable = testdisk.MakeFakePartitionTable("/", "/boot", "/boot/efi")
	rawBootcPipeline.SerializeStart(nil, []container.Spec{{Source: "foo"}}, nil, nil)

	return rawBootcPipeline
}

func TestRawBootcImageSerializeCreateUsersOptions(t *testing.T) {
	rawBootcPipeline := makeFakeRawBootcPipeline()

	for _, tc := range []struct {
		users              []users.User
		expectedUsersStage bool
	}{
		{nil, false},
		{[]users.User{{Name: "root"}}, true},
		{[]users.User{{Name: "foo"}}, true},
		{[]users.User{{Name: "root"}, {Name: "foo"}}, true},
	} {
		rawBootcPipeline.Users = tc.users

		pipeline := rawBootcPipeline.Serialize()
		usersStage := manifest.FindStage("org.osbuild.users", pipeline.Stages)
		if tc.expectedUsersStage {
			// ensure options got passed
			require.NotNil(t, usersStage)
			userOptions := usersStage.Options.(*osbuild.UsersStageOptions)
			for _, user := range tc.users {
				assert.NotNil(t, userOptions.Users[user.Name])
			}
		} else {
			require.Nil(t, usersStage)
		}
	}
}

func TestRawBootcImageSerializeCreateGroupOptions(t *testing.T) {
	rawBootcPipeline := makeFakeRawBootcPipeline()

	for _, tc := range []struct {
		groups              []users.Group
		expectedGroupsStage bool
	}{
		{nil, false},
		{[]users.Group{{Name: "root"}}, true},
		{[]users.Group{{Name: "foo"}}, true},
		{[]users.Group{{Name: "root"}, {Name: "foo"}}, true},
	} {
		rawBootcPipeline.Groups = tc.groups

		pipeline := rawBootcPipeline.Serialize()
		groupsStage := manifest.FindStage("org.osbuild.groups", pipeline.Stages)
		if tc.expectedGroupsStage {
			// ensure options got passed
			require.NotNil(t, groupsStage)
			groupOptions := groupsStage.Options.(*osbuild.GroupsStageOptions)
			for _, group := range tc.groups {
				assert.NotNil(t, groupOptions.Groups[group.Name])
			}
		} else {
			require.Nil(t, groupsStage)
		}
	}
}

func assertBootcDeploymentAndBindMount(t *testing.T, stage *osbuild.Stage) {
	// check for bind mount to deployment is there so
	// that the customization actually works
	deploymentMntIdx := findMountIdx(stage.Mounts, "org.osbuild.ostree.deployment")
	assert.True(t, deploymentMntIdx >= 0)
	bindMntIdx := findMountIdx(stage.Mounts, "org.osbuild.bind")
	assert.True(t, bindMntIdx >= 0)
	// order is important, bind must happen *after* deploy
	assert.True(t, bindMntIdx > deploymentMntIdx)
}

func TestRawBootcImageSerializeCustomizationGenCorrectStages(t *testing.T) {
	rawBootcPipeline := makeFakeRawBootcPipeline()

	for _, tc := range []struct {
		users   []users.User
		groups  []users.Group
		SELinux string

		expectedStages []string
	}{
		{nil, nil, "", nil},
		{[]users.User{{Name: "foo"}}, nil, "", []string{"org.osbuild.mkdir", "org.osbuild.users"}},
		{[]users.User{{Name: "foo"}}, nil, "targeted", []string{"org.osbuild.mkdir", "org.osbuild.users", "org.osbuild.selinux"}},
		{[]users.User{{Name: "foo"}}, []users.Group{{Name: "bar"}}, "targeted", []string{"org.osbuild.mkdir", "org.osbuild.users", "org.osbuild.users", "org.osbuild.selinux"}},
	} {
		rawBootcPipeline.Users = tc.users
		rawBootcPipeline.SELinux = tc.SELinux

		pipeline := rawBootcPipeline.Serialize()
		for _, expectedStage := range tc.expectedStages {
			stage := manifest.FindStage(expectedStage, pipeline.Stages)
			assert.NotNil(t, stage)
			assertBootcDeploymentAndBindMount(t, stage)
		}
	}
}
