package distro

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDistroIDParser(t *testing.T) {
	type testCase struct {
		stringID string
		expected *ID
		err      bool
	}

	testCases := []testCase{
		{
			stringID: "fedora-39",
			expected: &ID{
				Name:         "fedora",
				MajorVersion: 39,
				MinorVersion: -1,
			},
		},
		{
			stringID: "fedora-39.0",
			expected: &ID{
				Name:         "fedora",
				MajorVersion: 39,
				MinorVersion: 0,
			},
		},
		{
			stringID: "rhel-8.4",
			expected: &ID{
				Name:         "rhel",
				MajorVersion: 8,
				MinorVersion: 4,
			},
		},
		{
			stringID: "rhel-8",
			expected: &ID{
				Name:         "rhel",
				MajorVersion: 8,
				MinorVersion: -1,
			},
		},
		{
			stringID: "rhel-84",
			expected: &ID{
				Name:         "rhel",
				MajorVersion: 84,
				MinorVersion: -1,
			},
		},
		{
			stringID: "rhel-810",
			expected: &ID{
				Name:         "rhel",
				MajorVersion: 810,
				MinorVersion: -1,
			},
		},
		{
			stringID: "rhel-8.4.1",
			err:      true,
		},
		{
			stringID: "rhel",
			err:      true,
		},
		{
			stringID: "rhel-",
			err:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.stringID, func(t *testing.T) {
			id, err := ParseID(tc.stringID)

			if tc.err {
				assert.Error(t, err)
				assert.Nil(t, id)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, id)
			assert.Equal(t, tc.expected, id)
		})
	}

}
