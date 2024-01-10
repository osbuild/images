package manifest

import (
	"testing"

	"github.com/stretchr/testify/require"

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
