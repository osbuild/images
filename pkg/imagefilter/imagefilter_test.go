package imagefilter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/pkg/distrofactory"
	"github.com/osbuild/images/pkg/imagefilter"
	testrepos "github.com/osbuild/images/test/data/repositories"
)

func TestImageFilterSmoke(t *testing.T) {
	fac := distrofactory.NewDefault()
	repos, err := testrepos.New()
	require.NoError(t, err)

	imgFilter, err := imagefilter.New(fac, repos)
	require.NoError(t, err)
	res, err := imgFilter.Filter("*")
	require.NoError(t, err)
	assert.True(t, len(res) > 0)
}

func TestImageFilterFilter(t *testing.T) {
	fac := distrofactory.NewDefault()
	repos, err := testrepos.New()
	require.NoError(t, err)

	imgFilter, err := imagefilter.New(fac, repos)
	require.NoError(t, err)

	for _, tc := range []struct {
		searchExpr   []string
		expectsMatch bool
	}{
		// no prefix is a "fuzzy" filter and will check distro/arch/imgType
		{[]string{"foo"}, false},
		{[]string{"rhel-9.1"}, true},
		{[]string{"rhel*"}, true},
		{[]string{"x86_64"}, true},
		{[]string{"qcow2"}, true},
		// distro: prefix
		{[]string{"distro:foo"}, false},
		{[]string{"distro:centos-9"}, true},
		{[]string{"distro:centos*"}, true},
		{[]string{"distro:centos"}, false},
		// arch: prefix
		{[]string{"arch:foo"}, false},
		{[]string{"arch:x86_64"}, true},
		{[]string{"arch:x86*"}, true},
		{[]string{"arch:x86"}, false},
		// type: prefix
		{[]string{"type:foo"}, false},
		{[]string{"type:qcow2"}, true},
		{[]string{"type:qcow?"}, true},
		{[]string{"type:qcow"}, false},
		// bootmode: prefix
		{[]string{"bootmode:foo"}, false},
		{[]string{"bootmode:hybrid"}, true},
		// multiple filters are AND
		{[]string{"distro:centos-9", "type:foo"}, false},
		{[]string{"distro:centos-9", "type:qcow2"}, true},
		{[]string{"distro:centos-9", "arch:foo", "type:qcow2"}, false},
	} {

		t.Run(tc.searchExpr[0], func(t *testing.T) {
			t.Parallel()
			matches, err := imgFilter.Filter(tc.searchExpr...)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectsMatch, len(matches) > 0, tc)
		})
	}
}
