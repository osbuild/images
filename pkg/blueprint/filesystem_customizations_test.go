package blueprint

import (
	"encoding/json"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/pkg/pathpolicy"
)

// Happy tests for unmarshallers are in blueprint_test.go
func TestFilesystemCustomizationUnmarshalTOMLUnhappy(t *testing.T) {
	cases := []struct {
		name  string
		input string
		err   string
	}{
		{
			name: "mountpoint not string",
			input: `mountpoint = 42
			minsize = 42`,
			err: "toml: line 0: TOML unmarshal: mountpoint must be string, got 42 of type int64",
		},
		{
			name: "misize nor string nor int",
			input: `mountpoint="/"
			minsize = true`,
			err: "toml: line 0: TOML unmarshal: minsize must be integer or string, got true of type bool",
		},
		{
			name: "misize not parseable",
			input: `mountpoint="/"
			minsize = "20 KG"`,
			err: "toml: line 0: TOML unmarshal: minsize is not valid filesystem size (unknown data size units in string: 20 KG)",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var fsc FilesystemCustomization
			err := toml.Unmarshal([]byte(c.input), &fsc)
			assert.EqualError(t, err, c.err)
		})
	}
}

func TestFilesystemCustomizationUnmarshalJSONUnhappy(t *testing.T) {
	cases := []struct {
		name  string
		input string
		err   string
	}{
		{
			name:  "mountpoint not string",
			input: `{"mountpoint": 42, "minsize": 42}`,
			err:   "JSON unmarshal: mountpoint must be string, got 42 of type float64",
		},
		{
			name:  "misize nor string nor int",
			input: `{"mountpoint":"/", "minsize": true}`,
			err:   "JSON unmarshal: minsize must be float64 number or string, got true of type bool",
		},
		{
			name:  "misize not parseable",
			input: `{ "mountpoint": "/", "minsize": "20 KG"}`,
			err:   "JSON unmarshal: minsize is not valid filesystem size (unknown data size units in string: 20 KG)",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var fsc FilesystemCustomization
			err := json.Unmarshal([]byte(c.input), &fsc)
			assert.EqualError(t, err, c.err)
		})
	}
}

func TestCheckMountpointsPolicy(t *testing.T) {
	policy := pathpolicy.NewPathPolicies(map[string]pathpolicy.PathPolicy{
		"/": {Exact: true},
	})

	mps := []FilesystemCustomization{
		{Mountpoint: "/foo"},
		{Mountpoint: "/boot/"},
	}

	expectedErr := `The following errors occurred while setting up custom mountpoints:
path "/foo" is not allowed
path "/boot/" must be canonical`
	err := CheckMountpointsPolicy(mps, policy)
	assert.EqualError(t, err, expectedErr)
}
