package distro_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/pkg/distro"
)

func TestSortNames(t *testing.T) {
	for _, tc := range []struct {
		in     []string
		sorted []string
	}{
		{
			// distro names are sorted by first
			[]string{"foo-2", "bar-1", "foo-1"},
			[]string{"bar-1", "foo-1", "foo-2"},
		}, {
			// 1.4 is smaller than 1.10, sort.Strings will get this
			// wrong
			[]string{"foo-1.10", "foo-1.4"},
			[]string{"foo-1.4", "foo-1.10"},
		}, {
			// multiple "-" are ok
			[]string{"foo-bar-2", "bar-foo-1", "foo-bar-1"},
			[]string{"bar-foo-1", "foo-bar-1", "foo-bar-2"},
		}, {
			// missing "-" is ok
			[]string{"foo", "bar-1"},
			[]string{"bar-1", "foo"},
		}, {
			// foo-bar-1.4-beta is "lower" than foo-bar-1.4
			[]string{"foo-bar-1.4", "foo-bar-1.4-beta", "foo-bar-1.0"},
			[]string{"foo-bar-1.0", "foo-bar-1.4-beta", "foo-bar-1.4"},
		},
	} {
		err := distro.SortNames(tc.in)
		assert.NoError(t, err)
		assert.Equal(t, tc.sorted, tc.in)
	}
}

func TestSortNamesInvalidVersion(t *testing.T) {
	err := distro.SortNames([]string{"foo-1.x", "foo-2"})
	assert.EqualError(t, err, `Malformed version: 1.x`)
}
