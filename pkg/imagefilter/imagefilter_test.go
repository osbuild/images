package imagefilter_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/pkg/distrofactory"
	"github.com/osbuild/images/pkg/imagefilter"
	"github.com/osbuild/images/pkg/rpmmd"
	testrepos "github.com/osbuild/images/test/data/repositories"
)

func TestImageFilterSmoke(t *testing.T) {
	fac := distrofactory.NewDefault()
	repos, err := testrepos.New()
	require.NoError(t, err)

	imgFilter, err := imagefilter.New(fac, repos, nil)
	require.NoError(t, err)
	res, err := imgFilter.Filter("*")
	require.NoError(t, err)
	assert.True(t, len(res) > 0)
}

func TestImageFilterSpecificResult(t *testing.T) {
	fac := distrofactory.NewDefault()
	repos, err := testrepos.New()
	require.NoError(t, err)

	imgFilter, err := imagefilter.New(fac, repos, nil)
	require.NoError(t, err)

	res, err := imgFilter.Filter("distro:centos-9", "arch:x86_64", "type:qcow2")
	require.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, "centos-9", res[0].Distro.Name())
	assert.Equal(t, "x86_64", res[0].Arch.Name())
	assert.Equal(t, "qcow2", res[0].ImgType.Name())
	assert.True(t, len(res[0].Repos) > 0)
	assert.True(t, slices.IndexFunc(res[0].Repos, func(r rpmmd.RepoConfig) bool {
		return r.Name == "BaseOS"
	}) >= 0)
}

func TestImageFilterFilter(t *testing.T) {
	fac := distrofactory.NewDefault()
	repos, err := testrepos.New()
	require.NoError(t, err)

	for _, tc := range []struct {
		searchExpr   []string
		showHidden   bool
		expectsMatch bool
	}{
		// no prefix is a "fuzzy" filter and will check distro/arch/imgType
		{[]string{"foo"}, false, false},
		{[]string{"rhel-9.6"}, false, true},
		{[]string{"rhel*"}, false, true},
		{[]string{"x86_64"}, false, true},
		{[]string{"qcow2"}, false, true},
		// distro: prefix
		{[]string{"distro:foo"}, false, false},
		{[]string{"distro:centos-9"}, false, true},
		{[]string{"distro:centos*"}, false, true},
		{[]string{"distro:centos"}, false, false},
		// arch: prefix
		{[]string{"arch:foo"}, false, false},
		{[]string{"arch:x86_64"}, false, true},
		{[]string{"arch:x86*"}, false, true},
		{[]string{"arch:x86"}, false, false},
		// type: prefix
		{[]string{"type:foo"}, false, false},
		{[]string{"type:qcow2"}, false, true},
		{[]string{"type:qcow?"}, false, true},
		{[]string{"type:qcow"}, false, false},
		// bootmode: prefix
		{[]string{"bootmode:foo"}, false, false},
		{[]string{"bootmode:hybrid"}, false, true},
		// multiple filters are AND
		{[]string{"distro:centos-9", "type:foo"}, false, false},
		{[]string{"distro:centos-9", "type:qcow2"}, false, true},
		{[]string{"distro:centos-9", "arch:foo", "type:qcow2"}, false, false},
		// hidden image types
		{[]string{"type:iot-bootable-container", "distro:fedora-44"}, false, false},
		{[]string{"type:iot-bootable-container", "distro:fedora-44"}, true, true},
		{[]string{"type:ec2", "distro:rhel-10.0"}, false, false},
		{[]string{"type:ec2", "distro:rhel-10.0"}, true, true},
	} {
		t.Run(tc.searchExpr[0], func(t *testing.T) {
			t.Parallel()

			imgFilter, err := imagefilter.New(fac, repos, &imagefilter.FilterOptions{ShowHidden: tc.showHidden})
			require.NoError(t, err)

			matches, err := imgFilter.Filter(tc.searchExpr...)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectsMatch, len(matches) > 0, tc)
		})
	}
}
