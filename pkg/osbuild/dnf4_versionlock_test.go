package osbuild

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDNF4VersionlockStageValidate(t *testing.T) {
	type testCase struct {
		add    []string
		expErr string
	}

	testCases := map[string]testCase{
		"nil": {
			add:    nil,
			expErr: "org.osbuild.dnf4.versionlock: at least one package must be included in the 'add' list",
		},
		"zero": {
			add:    []string{},
			expErr: "org.osbuild.dnf4.versionlock: at least one package must be included in the 'add' list",
		},
		"one": {
			add: []string{"pkg-eins"},
		},
		"two": {
			add: []string{"pkg-eins", "pkg-zwei"},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			options := DNF4VersionlockOptions{
				Add: tc.add,
			}
			err := options.validate()
			if tc.expErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expErr)
			}
		})
	}
}
