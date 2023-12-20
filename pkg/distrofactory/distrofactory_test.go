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
