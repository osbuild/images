package rhel9

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/pkg/distro"
)

func TestDistroFactory(t *testing.T) {
	type testCase struct {
		strID    string
		expected distro.Distro
	}

	testCases := []testCase{
		{
			strID:    "rhel-90",
			expected: newDistro("rhel", 0),
		},
		{
			strID:    "rhel-9.0",
			expected: newDistro("rhel", 0),
		},
		{
			strID:    "rhel-93",
			expected: newDistro("rhel", 3),
		},
		{
			strID:    "rhel-9.3",
			expected: newDistro("rhel", 3),
		},
		{
			strID:    "rhel-910", // this is intentionally not supported for el9
			expected: nil,
		},
		{
			strID:    "rhel-9.10",
			expected: newDistro("rhel", 10),
		},
		{
			strID:    "centos-9",
			expected: newDistro("centos", -1),
		},
		{
			strID:    "centos-9.0",
			expected: nil,
		},
		{
			strID:    "rhel-9",
			expected: nil,
		},
		{
			strID:    "rhel-8.0",
			expected: nil,
		},
		{
			strID:    "rhel-80",
			expected: nil,
		},
		{
			strID:    "rhel-8.4",
			expected: nil,
		},
		{
			strID:    "rhel-84",
			expected: nil,
		},
		{
			strID:    "rhel-8.10",
			expected: nil,
		},
		{
			strID:    "rhel-810",
			expected: nil,
		},
		{
			strID:    "rhel-8",
			expected: nil,
		},
		{
			strID:    "rhel-8.4.1",
			expected: nil,
		},
		{
			strID:    "rhel-7",
			expected: nil,
		},
		{
			strID:    "rhel-79",
			expected: nil,
		},
		{
			strID:    "rhel-7.9",
			expected: nil,
		},
		{
			strID:    "fedora-9",
			expected: nil,
		},
		{
			strID:    "fedora-37",
			expected: nil,
		},
		{
			strID:    "fedora-38.1",
			expected: nil,
		},
		{
			strID:    "fedora",
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.strID, func(t *testing.T) {
			d := DistroFactory(tc.strID)
			if tc.expected == nil {
				assert.Nil(t, d)
			} else {
				assert.NotNil(t, d)
				assert.Equal(t, tc.expected.Name(), d.Name())
			}
		})
	}
}
