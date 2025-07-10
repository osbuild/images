package osbuild

import (
	"testing"

	"github.com/osbuild/images/pkg/rpmmd"
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

func TestGenVersionlockStageOptions(t *testing.T) {
	packages := []rpmmd.PackageSpec{
		{
			Name:     "test-kernel",
			Epoch:    0,
			Version:  "13.3",
			Release:  "7.el9",
			Arch:     "x86_64",
			Checksum: "sha256:7777777777777777777777777777777777777777777777777777777777777777",
		},
		{
			Name:     "uki-direct",
			Epoch:    0,
			Version:  "25.11",
			Release:  "1.el9",
			Arch:     "noarch",
			Checksum: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		},
		{
			Name:     "shim-x64",
			Epoch:    0,
			Version:  "15.8",
			Release:  "3",
			Arch:     "x86_64",
			Checksum: "sha256:aae94b3b8451ef28b02594d9abca5979e153c14f4db25283b011403fa92254fd",
		},
		{
			Name:     "pkg42",
			Epoch:    7,
			Version:  "42.13",
			Release:  "9",
			Arch:     "x86_64",
			Checksum: "sha256:4242424242424242424242424242424242424242424242424242424242424242",
		},
	}

	type testCase struct {
		packageNames []string
		expOut       []string
		expErr       string
	}

	testCases := map[string]testCase{
		"shim": {
			packageNames: []string{
				"shim-x64",
			},
			expOut: []string{"shim-x64-0:15.8-3"},
		},
		"42": {
			packageNames: []string{
				"pkg42",
			},
			expOut: []string{"pkg42-7:42.13-9"},
		},
		"all": {
			packageNames: []string{
				"test-kernel",
				"uki-direct",
				"shim-x64",
				"pkg42",
			},
			expOut: []string{
				"test-kernel-0:13.3-7.el9",
				"uki-direct-0:25.11-1.el9",
				"shim-x64-0:15.8-3",
				"pkg42-7:42.13-9",
			},
		},
		"not-found": {
			packageNames: []string{
				"not-a-package",
			},
			expErr: `org.osbuild.dnf4.versionlock: package "not-a-package" not found in package list`,
		},
		"mixed": {
			packageNames: []string{
				"pkg42",
				"not-a-package",
			},
			expErr: `org.osbuild.dnf4.versionlock: package "not-a-package" not found in package list`,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			options, err := GenDNF4VersionlockStageOptions(tc.packageNames, packages)
			if tc.expErr != "" {
				assert.EqualError(err, tc.expErr)
			} else {
				assert.NoError(err)
				expOptions := &DNF4VersionlockOptions{Add: tc.expOut}
				assert.Equal(expOptions, options)
			}
		})
	}
}
