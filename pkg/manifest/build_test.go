package manifest

import (
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
	mf := New(DISTRO_FEDORA)
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

func TestNewBuildFromContainerSpecs(t *testing.T) {
	containers := []container.SourceSpec{
		{
			Name:   "Build container",
			Source: "ghcr.io/ondrejbudai/booc:fedora",
		},
	}
	mf := New(DISTRO_FEDORA)
	runner := &runner.Fedora{Version: 39}

	buildIf := NewBuildFromContainer(&mf, runner, containers, nil)
	require.NotNil(t, buildIf)
	build := buildIf.(*BuildrootFromContainer)

	fakeContainerSpecs := []container.Spec{
		{
			ImageID: "id-0",
			Source:  "registry.example.org/reg/img",
		},
	}
	// containerSpecs is "nil" until serializeStart populates it
	require.Nil(t, build.getContainerSpecs())
	build.serializeStart(nil, fakeContainerSpecs, nil, nil)
	assert.Equal(t, build.getContainerSpecs(), fakeContainerSpecs)

	osbuildPipeline := build.serialize()
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
