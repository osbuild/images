package distroidparser

import (
	"testing"

	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/fedora"
	"github.com/stretchr/testify/require"
)

func TestDefaltParser(t *testing.T) {
	type testCase struct {
		idStr    string
		expected *distro.ID
		err      bool
	}

	testCases := []testCase{
		// Fedora
		{
			idStr:    "fedora-39",
			expected: &distro.ID{Name: "fedora", MajorVersion: 39, MinorVersion: -1},
		},
		{
			idStr:    "fedora-39.1",
			expected: &distro.ID{Name: "fedora", MajorVersion: 39, MinorVersion: 1},
		},
		{
			idStr: "fedora-39.1.1",
			err:   true,
		},
		// RHEL-7
		{
			idStr:    "rhel-7",
			expected: &distro.ID{Name: "rhel", MajorVersion: 7, MinorVersion: -1},
		},
		{
			idStr:    "rhel-79",
			expected: &distro.ID{Name: "rhel", MajorVersion: 79, MinorVersion: -1},
		},
		{
			idStr:    "rhel-7.9",
			expected: &distro.ID{Name: "rhel", MajorVersion: 7, MinorVersion: 9},
		},
		// RHEL-8
		{
			idStr:    "rhel-8",
			expected: &distro.ID{Name: "rhel", MajorVersion: 8, MinorVersion: -1},
		},
		{
			idStr:    "rhel-80",
			expected: &distro.ID{Name: "rhel", MajorVersion: 8, MinorVersion: 0},
		},
		{
			idStr:    "rhel-8.0",
			expected: &distro.ID{Name: "rhel", MajorVersion: 8, MinorVersion: 0},
		},
		{
			idStr:    "rhel-810",
			expected: &distro.ID{Name: "rhel", MajorVersion: 8, MinorVersion: 10},
		},
		{
			idStr:    "rhel-8.10",
			expected: &distro.ID{Name: "rhel", MajorVersion: 8, MinorVersion: 10},
		},
		{
			idStr:    "rhel-8100",
			expected: &distro.ID{Name: "rhel", MajorVersion: 8100, MinorVersion: -1},
		},
		{
			idStr: "rhel-8.1.1",
			err:   true,
		},
		// CentOS-8
		{
			idStr:    "centos-8",
			expected: &distro.ID{Name: "centos", MajorVersion: 8, MinorVersion: -1},
		},
		{
			idStr:    "centos-8.2",
			expected: &distro.ID{Name: "centos", MajorVersion: 8, MinorVersion: 2},
		},
		{
			idStr: "centos-8.2.2",
			err:   true,
		},
		// RHEL-9
		{
			idStr:    "rhel-9",
			expected: &distro.ID{Name: "rhel", MajorVersion: 9, MinorVersion: -1},
		},
		{
			idStr:    "rhel-90",
			expected: &distro.ID{Name: "rhel", MajorVersion: 9, MinorVersion: 0},
		},
		{
			idStr:    "rhel-9.0",
			expected: &distro.ID{Name: "rhel", MajorVersion: 9, MinorVersion: 0},
		},
		{
			idStr:    "rhel-910",
			expected: &distro.ID{Name: "rhel", MajorVersion: 910, MinorVersion: -1},
		},
		{
			idStr:    "rhel-9.10",
			expected: &distro.ID{Name: "rhel", MajorVersion: 9, MinorVersion: 10},
		},
		{
			idStr:    "rhel-9100",
			expected: &distro.ID{Name: "rhel", MajorVersion: 9100, MinorVersion: -1},
		},
		{
			idStr: "rhel-9.1.1",
			err:   true,
		},
		// CentOS-9
		{
			idStr:    "centos-9",
			expected: &distro.ID{Name: "centos", MajorVersion: 9, MinorVersion: -1},
		},
		{
			idStr:    "centos-9.2",
			expected: &distro.ID{Name: "centos", MajorVersion: 9, MinorVersion: 2},
		},
		{
			idStr: "centos-9.2.2",
			err:   true,
		},
		// Non-existing distro
		{
			idStr:    "tuxdistro-1",
			expected: &distro.ID{Name: "tuxdistro", MajorVersion: 1, MinorVersion: -1},
		},
		{
			idStr:    "tuxdistro-1.2",
			expected: &distro.ID{Name: "tuxdistro", MajorVersion: 1, MinorVersion: 2},
		},
		{
			idStr:    "tuxdistro-123.321",
			expected: &distro.ID{Name: "tuxdistro", MajorVersion: 123, MinorVersion: 321},
		},
		{
			idStr: "tuxdistro-1.2.3",
			err:   true,
		},
	}

	parser := NewDefaultParser()
	for _, tc := range testCases {
		t.Run(tc.idStr, func(t *testing.T) {
			id, err := parser.Parse(tc.idStr)

			if tc.err {
				require.Error(t, err)
				require.Nil(t, id)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expected, id)
		})
	}
}

func TestParserDoubleMatch(t *testing.T) {
	Parser := New(fedora.ParseID, fedora.ParseID)

	require.Panics(t, func() {
		_, _ = Parser.Parse("fedora-33")
	}, "Parser should panic when fedora-33 is matched by multiple parsers")
}
