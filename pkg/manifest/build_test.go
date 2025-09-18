package manifest

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/osbuild/images/pkg/runner"
)

func TestBuildContainerBuildableNo(t *testing.T) {
	repos := []rpmmd.RepoConfig{}
	mf := New()
	runner := &runner.Fedora{Version: 39}

	buildIf := NewBuild(&mf, runner, repos, nil)
	build := buildIf.(*BuildrootFromPackages)
	require.NotNil(t, build)

	for _, tc := range []struct {
		packageSpec           []rpmmd.PackageSpec
		containerBuildable    bool
		expectedSELinuxLabels map[string]string
	}{
		// no pkgs means no selinux labels (container build or not)
		{
			[]rpmmd.PackageSpec{},
			false,
			map[string]string{},
		},
		{
			[]rpmmd.PackageSpec{},
			true,
			map[string]string{},
		},
		{
			[]rpmmd.PackageSpec{{Name: "coreutils"}},
			false,
			map[string]string{
				"/usr/bin/cp": "system_u:object_r:install_exec_t:s0",
			},
		},
		{
			[]rpmmd.PackageSpec{{Name: "tar"}},
			false,
			map[string]string{
				"/usr/bin/tar": "system_u:object_r:install_exec_t:s0",
			},
		},
		{
			[]rpmmd.PackageSpec{{Name: "coreutils"}, {Name: "tar"}},
			false,
			map[string]string{
				"/usr/bin/cp":  "system_u:object_r:install_exec_t:s0",
				"/usr/bin/tar": "system_u:object_r:install_exec_t:s0",
			},
		},
		{
			[]rpmmd.PackageSpec{{Name: "coreutils"}},
			true,
			map[string]string{
				"/usr/bin/cp":     "system_u:object_r:install_exec_t:s0",
				"/usr/bin/mount":  "system_u:object_r:install_exec_t:s0",
				"/usr/bin/umount": "system_u:object_r:install_exec_t:s0",
			},
		},
		{
			[]rpmmd.PackageSpec{{Name: "coreutils"}, {Name: "tar"}},
			true,
			map[string]string{
				"/usr/bin/cp":     "system_u:object_r:install_exec_t:s0",
				"/usr/bin/mount":  "system_u:object_r:install_exec_t:s0",
				"/usr/bin/umount": "system_u:object_r:install_exec_t:s0",
				"/usr/bin/tar":    "system_u:object_r:install_exec_t:s0",
			},
		},
	} {
		build.packageSpecs = tc.packageSpec
		build.containerBuildable = tc.containerBuildable

		labels := build.getSELinuxLabels()
		require.Equal(t, labels, tc.expectedSELinuxLabels)
	}
}

var fakeContainerSpecs = []container.Spec{
	{
		ImageID: "id-0",
		Source:  "registry.example.org/reg/img",
	},
}

func TestNewBuildFromContainerSpecs(t *testing.T) {
	containers := []container.SourceSpec{
		{
			Name:   "Build container",
			Source: "ghcr.io/ondrejbudai/booc:fedora",
		},
	}
	mf := New()
	runner := &runner.Fedora{Version: 39}

	buildIf := NewBuildFromContainer(&mf, runner, containers, nil)
	require.NotNil(t, buildIf)
	build := buildIf.(*BuildrootFromContainer)

	// containerSpecs is "nil" until serializeStart populates it
	require.Nil(t, build.getContainerSpecs())
	err := build.serializeStart(Inputs{Containers: fakeContainerSpecs})
	assert.NoError(t, err)
	assert.Equal(t, build.getContainerSpecs(), fakeContainerSpecs)

	osbuildPipeline, err := build.serialize()
	require.NoError(t, err)
	require.Len(t, osbuildPipeline.Stages, 2)
	assert.Equal(t, osbuildPipeline.Stages[0].Type, "org.osbuild.container-deploy")
	// one container src input is added
	assert.Equal(t, len(osbuildPipeline.Stages[0].Inputs.(osbuild.ContainerDeployInputs).Images.References), 1)

	assert.Equal(t, osbuildPipeline.Stages[1].Type, "org.osbuild.selinux")
	assert.Equal(t, len(osbuildPipeline.Stages[1].Options.(*osbuild.SELinuxStageOptions).Labels), 1)
	assert.Equal(t, osbuildPipeline.Stages[1].Options.(*osbuild.SELinuxStageOptions).Labels["/usr/bin/ostree"], "system_u:object_r:install_exec_t:s0")

	// serializeEnd "cleans up"
	build.serializeEnd()
	require.Nil(t, build.getContainerSpecs())
}

func TestBuildFromContainerSpecsGetSelinuxLabelsNotBuildable(t *testing.T) {
	build := &BuildrootFromContainer{}

	assert.Equal(t, build.getSELinuxLabels(), map[string]string{
		"/usr/bin/ostree": "system_u:object_r:install_exec_t:s0",
	})
}

func TestBuildFromContainerSpecsGetSelinuxLabelsWithContainerBuildable(t *testing.T) {
	build := &BuildrootFromContainer{
		containerBuildable: true,
	}

	assert.Equal(t, build.getSELinuxLabels(), map[string]string{
		"/usr/bin/ostree": "system_u:object_r:install_exec_t:s0",
		"/usr/bin/mount":  "system_u:object_r:install_exec_t:s0",
		"/usr/bin/umount": "system_u:object_r:install_exec_t:s0",
	})
}

func TestNewBuildOptionDisableSELinux(t *testing.T) {
	for _, disableSELinux := range []bool{false, true} {
		mf := New()
		runner := &runner.Linux{}
		opts := &BuildOptions{
			DisableSELinux: disableSELinux,
		}
		buildIf := NewBuild(&mf, runner, nil, opts)
		require.NotNil(t, buildIf)
		build := buildIf.(*BuildrootFromPackages)

		build.packageSpecs = []rpmmd.PackageSpec{
			{Name: "foo", Checksum: "sha256:" + strings.Repeat("x", 32)},
		}
		osbuildPipeline, err := build.serialize()
		require.NoError(t, err)
		if disableSELinux {
			require.Len(t, osbuildPipeline.Stages, 1)
			assert.Equal(t, osbuildPipeline.Stages[0].Type, "org.osbuild.rpm")
		} else {
			require.Len(t, osbuildPipeline.Stages, 2)
			assert.Equal(t, osbuildPipeline.Stages[0].Type, "org.osbuild.rpm")
			assert.Equal(t, osbuildPipeline.Stages[1].Type, "org.osbuild.selinux")
		}
	}
}

func TestNewBuildOptionSELinuxPolicyBuildrootFromPackages(t *testing.T) {
	for _, tc := range []struct {
		policy              string
		expectedBuildPkg    string
		expectedFileContext string
	}{
		{"", "selinux-policy-targeted", "etc/selinux/targeted/contexts/files/file_contexts"},
		{"custom", "selinux-policy-custom", "etc/selinux/custom/contexts/files/file_contexts"},
	} {
		mf := New()
		runner := &runner.Linux{}
		opts := &BuildOptions{
			SELinuxPolicy: tc.policy,
		}
		buildIf := NewBuild(&mf, runner, nil, opts)
		require.NotNil(t, buildIf)
		build := buildIf.(*BuildrootFromPackages)
		build.packageSpecs = []rpmmd.PackageSpec{
			{Name: "foo", Checksum: "sha256:" + strings.Repeat("x", 32)},
		}
		osbuildPipeline, err := build.serialize()
		require.NoError(t, err)
		require.Len(t, osbuildPipeline.Stages, 2)
		assert.Equal(t, "org.osbuild.selinux", osbuildPipeline.Stages[1].Type)
		assert.Equal(t, tc.expectedFileContext, osbuildPipeline.Stages[1].Options.(*osbuild.SELinuxStageOptions).FileContexts)
		assert.Contains(t, build.getPackageSetChain(DISTRO_NULL)[0].Include, tc.expectedBuildPkg)
	}
}

func TestNewBuildOptionSELinuxPolicyBuildFromCnt(t *testing.T) {
	for _, tc := range []struct {
		policy              string
		expectedBuildPkg    string
		expectedFileContext string
	}{
		{"", "selinux-policy-targeted", "etc/selinux/targeted/contexts/files/file_contexts"},
		{"custom", "selinux-policy-custom", "etc/selinux/custom/contexts/files/file_contexts"},
	} {
		mf := New()
		runner := &runner.Linux{}
		opts := &BuildOptions{
			SELinuxPolicy: tc.policy,
		}
		buildIf := NewBuildFromContainer(&mf, runner, nil, opts)
		require.NotNil(t, buildIf)
		build := buildIf.(*BuildrootFromContainer)
		build.containerSpecs = fakeContainerSpecs
		osbuildPipeline, err := build.serialize()
		require.NoError(t, err)
		require.Len(t, osbuildPipeline.Stages, 2)
		assert.Equal(t, "org.osbuild.selinux", osbuildPipeline.Stages[1].Type)
		assert.Equal(t, tc.expectedFileContext, osbuildPipeline.Stages[1].Options.(*osbuild.SELinuxStageOptions).FileContexts)
	}
}

func TestNewBuildFromContainerOptionDisableSELinux(t *testing.T) {
	for _, disableSELinux := range []bool{false, true} {
		mf := New()
		runner := &runner.Linux{}
		opts := &BuildOptions{
			DisableSELinux: disableSELinux,
		}
		buildIf := NewBuildFromContainer(&mf, runner, nil, opts)
		require.NotNil(t, buildIf)
		build := buildIf.(*BuildrootFromContainer)

		build.containerSpecs = fakeContainerSpecs
		osbuildPipeline, err := build.serialize()
		require.NoError(t, err)
		if disableSELinux {
			require.Len(t, osbuildPipeline.Stages, 1)
			assert.Equal(t, osbuildPipeline.Stages[0].Type, "org.osbuild.container-deploy")
		} else {
			require.Len(t, osbuildPipeline.Stages, 2)
			assert.Equal(t, osbuildPipeline.Stages[0].Type, "org.osbuild.container-deploy")
			assert.Equal(t, osbuildPipeline.Stages[1].Type, "org.osbuild.selinux")
		}
	}
}

func TestNewBootstrap(t *testing.T) {
	containers := []container.SourceSpec{
		{
			Name:   "Bootstrap container",
			Source: "ghcr.io/ondrejbudai/cool:stuff",
		},
	}
	mf := New()

	bootstrapIf := NewBootstrap(&mf, containers)
	require.NotNil(t, bootstrapIf)
	bootstrap := bootstrapIf.(*BuildrootFromContainer)
	assert.Equal(t, "bootstrap-buildroot", bootstrap.Name())
	assert.Equal(t, true, bootstrap.disableSelinux)
	assert.Equal(t, containers, bootstrap.containers)
	assert.Len(t, mf.pipelines, 1)
	assert.Equal(t, bootstrapIf, mf.pipelines[0])
}

func TestBuildOptionBootstrapForNewBuild(t *testing.T) {
	mf := New()
	runner := &runner.Linux{}
	bootstrapIf := NewBootstrap(&mf, nil)

	opts := &BuildOptions{BootstrapPipeline: bootstrapIf}
	buildIf := NewBuild(&mf, runner, nil, opts)
	build := buildIf.(*BuildrootFromPackages)
	assert.Equal(t, bootstrapIf, build.build)

	mf = New()
	bcIf := NewBuildFromContainer(&mf, runner, nil, opts)
	bc := bcIf.(*BuildrootFromContainer)
	assert.Equal(t, bootstrapIf, bc.build)
}
