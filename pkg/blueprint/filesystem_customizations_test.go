package blueprint_test

import (
	"encoding/json"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/pathpolicy"
)

// ensure all fields that are supported are filled here
var allFieldsFsc = blueprint.FilesystemCustomization{
	Mountpoint: "/data",
	MinSize:    1234567890,
}

func TestFilesystemCustomizationMarshalUnmarshalTOML(t *testing.T) {
	b, err := toml.Marshal(allFieldsFsc)
	assert.NoError(t, err)

	var fsc blueprint.FilesystemCustomization
	err = toml.Unmarshal(b, &fsc)
	assert.NoError(t, err)
	assert.Equal(t, fsc, allFieldsFsc)
}

func TestFilesystemCustomizationMarshalUnmarshalJSON(t *testing.T) {
	b, err := json.Marshal(allFieldsFsc)
	assert.NoError(t, err)

	var fsc blueprint.FilesystemCustomization
	err = json.Unmarshal(b, &fsc)
	assert.NoError(t, err)
	assert.Equal(t, fsc, allFieldsFsc)
}

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
			err: `toml: line 1 (last key "mountpoint"): incompatible types: TOML value has type int64; destination has type string`,
		},
		{
			name: "minsize nor string nor int",
			input: `mountpoint="/"
			minsize = true`,
			err: `toml: line 2 (last key "minsize"): error decoding TOML size: failed to convert value "true" to number`,
		},
		{
			name: "minsize not parseable",
			input: `mountpoint="/"
			minsize = "20 KG"`,
			err: `toml: line 2 (last key "minsize"): error decoding TOML size: unknown data size units in string: 20 KG`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var fsc blueprint.FilesystemCustomization
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
			err:   `json: cannot unmarshal number into Go struct field filesystemCustomization.mountpoint of type string`,
		},
		{
			name:  "minsize nor string nor int",
			input: `{"mountpoint":"/", "minsize": true}`,
			err:   `JSON unmarshal: error decoding minsize value for mountpoint "/": error decoding JSON size: failed to convert value "true" to number`,
		},
		{
			name:  "minsize not parseable",
			input: `{ "mountpoint": "/", "minsize": "20 KG"}`,
			err:   `JSON unmarshal: error decoding minsize value for mountpoint "/": error decoding JSON size: unknown data size units in string: 20 KG`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var fsc blueprint.FilesystemCustomization
			err := json.Unmarshal([]byte(c.input), &fsc)
			assert.EqualError(t, err, c.err)
		})
	}
}

func TestFilesystemCustomizationUnmarshalTOMLNotAnObject(t *testing.T) {
	cases := []struct {
		name  string
		input string
		err   string
	}{
		{
			name: "filesystem is not an object",
			input: `
[customizations]
filesystem = ["hello"]`,
			err: "toml: line 3 (last key \"customizations.filesystem\"): type mismatch for blueprint.FilesystemCustomization: expected table but found string",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var bp blueprint.Blueprint
			err := toml.Unmarshal([]byte(c.input), &bp)
			assert.EqualError(t, err, c.err)
		})
	}
}

func TestCheckMountpointsPolicy(t *testing.T) {
	policy := pathpolicy.NewPathPolicies(map[string]pathpolicy.PathPolicy{
		"/": {Exact: true},
	})

	mps := []blueprint.FilesystemCustomization{
		{Mountpoint: "/foo"},
		{Mountpoint: "/boot/"},
	}

	expectedErr := `The following errors occurred while setting up custom mountpoints:
path "/foo" is not allowed
path "/boot/" must be canonical`
	err := blueprint.CheckMountpointsPolicy(mps, policy)
	assert.EqualError(t, err, expectedErr)
}

func TestUnmarshalTOMLNoUndecoded(t *testing.T) {
	testTOML := `
[[customizations.filesystem]]
mountpoint = "/foo"
minsize = 1000
`

	var bp blueprint.Blueprint
	meta, err := toml.Decode(testTOML, &bp)
	assert.NoError(t, err)
	assert.Equal(t, blueprint.Blueprint{
		Customizations: &blueprint.Customizations{
			Filesystem: []blueprint.FilesystemCustomization{
				{
					Mountpoint: "/foo",
					MinSize:    1000,
				},
			},
		},
	}, bp)
	assert.Equal(t, 0, len(meta.Undecoded()))
}
