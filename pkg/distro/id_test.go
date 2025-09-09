package distro

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDistroIDParser(t *testing.T) {
	type testCase struct {
		stringID        string
		expected        *ID
		expectedErr     string
		expectedErrType error
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
			stringID:    "rhel-8.4.1",
			expectedErr: `error when parsing distro name "rhel-8.4.1": too many dots in the version (2)`,
		},
		{
			stringID:    "rhel",
			expectedErr: `error when parsing distro name "rhel": A dash is expected to separate distro name and version`,
		},
		{
			stringID: "rhel-",
			expectedErr: `error when parsing distro name "rhel-": parsing major version failed, inner error:
strconv.Atoi: parsing "": invalid syntax`,
			expectedErrType: strconv.ErrSyntax,
		},
		{
			stringID: "centos-stream",
			expectedErr: `error when parsing distro name "centos-stream": parsing major version failed, inner error:
strconv.Atoi: parsing "stream": invalid syntax`,
			expectedErrType: strconv.ErrSyntax,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.stringID, func(t *testing.T) {
			id, err := ParseID(tc.stringID)

			if tc.expectedErr != "" {
				assert.ErrorContains(t, err, tc.expectedErr)
				assert.Nil(t, id)
				if tc.expectedErrType != nil {
					assert.ErrorAs(t, err, &tc.expectedErrType)
				}
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
