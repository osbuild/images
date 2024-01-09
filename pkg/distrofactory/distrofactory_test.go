package distrofactory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDistroDefaultList(t *testing.T) {
	type testCase struct {
		strID              string
		expectedDistroName string
	}

	testCases := []testCase{
		{
			strID:              "rhel-7.9",
			expectedDistroName: "rhel-7.9",
		},
		{
			strID:              "rhel-89",
			expectedDistroName: "rhel-8.9",
		},
		{
			strID:              "rhel-8.9",
			expectedDistroName: "rhel-8.9",
		},
		{
			strID:              "rhel-810",
			expectedDistroName: "rhel-8.10",
		},
		{
			strID:              "rhel-8.10",
			expectedDistroName: "rhel-8.10",
		},
		{
			strID:              "rhel-91",
			expectedDistroName: "rhel-9.1",
		},
		{
			strID:              "rhel-9.1",
			expectedDistroName: "rhel-9.1",
		},
		{
			strID:              "fedora-38",
			expectedDistroName: "fedora-38",
		},
	}

	df := NewDefault()

	for _, tc := range testCases {
		t.Run(tc.strID, func(t *testing.T) {
			d := df.GetDistro(tc.strID)
			assert.NotNil(t, d)
			assert.Equal(t, tc.expectedDistroName, d.Name())
		})
	}

}

func TestGetDistroDefaultListWithAliases(t *testing.T) {
	type testCase struct {
		aliases            map[string]string
		strID              string
		expectedDistroName string
		fail               bool
		errorMsg           string
	}

	testCases := []testCase{
		{
			aliases: map[string]string{
				"rhel-9": "rhel-9.1",
			},
			strID:              "rhel-9",
			expectedDistroName: "rhel-9.1",
		},
		{
			aliases: map[string]string{
				"best_distro-123": "rhel-9.1",
			},
			strID:              "best_distro-123",
			expectedDistroName: "rhel-9.1",
		},
		{
			aliases: map[string]string{
				"rhel-9.3": "rhel-9.1",
				"rhel-9.2": "rhel-9.1",
			},
			fail:     true,
			errorMsg: `invalid aliases: ["alias 'rhel-9.2' masks an existing distro" "alias 'rhel-9.3' masks an existing distro"]`,
		},
		{
			aliases: map[string]string{
				"rhel-12": "rhel-12.12",
				"rhel-13": "rhel-13.13",
			},
			fail:     true,
			errorMsg: `invalid aliases: ["alias 'rhel-12' targets a non-existing distro 'rhel-12.12'" "alias 'rhel-13' targets a non-existing distro 'rhel-13.13'"]`,
		},
	}

	df := NewDefault()
	for _, tc := range testCases {
		t.Run(tc.strID, func(t *testing.T) {
			err := df.RegisterAliases(tc.aliases)

			if tc.fail {
				assert.Error(t, err)
				assert.Equal(t, tc.errorMsg, err.Error())
				return
			}

			assert.NoError(t, err)
			d := df.GetDistro(tc.strID)
			assert.NotNil(t, d)
			assert.Equal(t, tc.expectedDistroName, d.Name())
		})
	}

}
