package manifest_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/internal/assertx"
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

	rawBootcPipeline.SerializeStart(nil, []container.Spec{{Source: "foo"}}, nil)
	imagePipeline := rawBootcPipeline.Serialize()
	assert.Equal(t, "image", imagePipeline.Name)

	bootcInst := manifest.FindStage("org.osbuild.bootc.install-to-filesystem", imagePipeline.Stages)
	require.NotNil(t, bootcInst)
	opts := bootcInst.Options.(*osbuild.BootcInstallToFilesystemOptions)
	assert.Equal(t, []string{"some-ssh-key"}, opts.RootSSHAuthorizedKeys)
}

func TestRawBootcImageSerializeMountsValidated(t *testing.T) {
	mani := manifest.New()
	runner := &runner.Linux{}
	build := manifest.NewBuildFromContainer(&mani, runner, nil, nil)

	rawBootcPipeline := manifest.NewRawBootcImage(build, nil, nil)
	// note that we create a partition table without /boot here
	rawBootcPipeline.PartitionTable = testdisk.MakeFakePartitionTable("/", "/missing-boot")
	rawBootcPipeline.SerializeStart(nil, []container.Spec{{Source: "foo"}}, nil)
	assert.PanicsWithError(t, `required mounts for bootupd stage [/boot /boot/efi] missing`, func() {
		rawBootcPipeline.Serialize()
	})
}

func TestRawBootcImageSerializeValidatesUsers(t *testing.T) {
	mani := manifest.New()
	runner := &runner.Linux{}
	build := manifest.NewBuildFromContainer(&mani, runner, nil, nil)

	rawBootcPipeline := manifest.NewRawBootcImage(build, nil, nil)
	rawBootcPipeline.PartitionTable = testdisk.MakeFakePartitionTable("/", "/boot", "/boot/efi")
	rawBootcPipeline.SerializeStart(nil, []container.Spec{{Source: "foo"}}, nil)

	for _, tc := range []struct {
		users       []users.User
		expectedErr string
	}{
		// good
		{nil, ""},
		{[]users.User{{Name: "root"}}, ""},
		{[]users.User{{Name: "root", Key: common.ToPtr("some-key")}}, ""},
		// bad
		{[]users.User{{Name: "foo"}},
			"raw bootc image only supports the root user, got.*"},
		{[]users.User{{Name: "root"}, {Name: "foo"}},
			"raw bootc image only supports a single root key for user customization, got.*"},
	} {
		rawBootcPipeline.Users = tc.users

		if tc.expectedErr == "" {
			rawBootcPipeline.Serialize()
		} else {
			expectedErr := regexp.MustCompile(tc.expectedErr)
			assertx.PanicsWithErrorRegexp(t, expectedErr, func() {
				rawBootcPipeline.Serialize()
			})
		}
	}
}
