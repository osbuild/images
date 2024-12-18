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
			stringID: "fedora-41",
			expected: &ID{
				Name:         "fedora",
				MajorVersion: 41,
				MinorVersion: -1,
			},
		},
		{
			stringID: "fedora-41.0",
			expected: &ID{
				Name:         "fedora",
				MajorVersion: 41,
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
			stringID: "centos-stream-8",
			expected: &ID{
				Name:         "centos-stream",
				MajorVersion: 8,
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
		{
			stringID: "centos-stream",
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

			ver, err := id.Version()
			assert.NoError(t, err)
			assert.Equal(t, ver.Segments()[0], tc.expected.MajorVersion)
			if tc.expected.MinorVersion > -1 {
				assert.Equal(t, ver.Segments()[1], tc.expected.MinorVersion)
			}
		})
	}

}
