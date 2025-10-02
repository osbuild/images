package osbuild

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDNFConfigStageOptions(t *testing.T) {
	variables := []DNFVariable{
		{
			Name:  "release",
			Value: "8.4",
		},
	}

	dnfconfig := &DNFConfig{
		Main: &DNFConfigMain{
			IPResolve: "4",
		},
	}

	expectedOptions := &DNFConfigStageOptions{
		Variables: variables,
		Config:    dnfconfig,
	}
	actualOptions := NewDNFConfigStageOptions(variables, dnfconfig)
	assert.Equal(t, expectedOptions, actualOptions)
}

func TestNewDNFConfigStage(t *testing.T) {
	expectedStage := &Stage{
		Type:    "org.osbuild.dnf.config",
		Options: &DNFConfigStageOptions{},
	}
	actualStage := NewDNFConfigStage(&DNFConfigStageOptions{})
	assert.Equal(t, expectedStage, actualStage)
}

func TestJSONDNFConfigStage(t *testing.T) {
	expectedOptions := DNFConfigStageOptions{
		Variables: []DNFVariable{
			{
				Name:  "release",
				Value: "8.4",
			},
		},
		Config: &DNFConfig{
			Main: &DNFConfigMain{
				IPResolve: "4",
			},
		},
	}

	inputString := `{"variables":[{"name":"release","value":"8.4"}],"config":{"main":{"ip_resolve":"4"}}}`
	var inputOptions DNFConfigStageOptions
	err := json.Unmarshal([]byte(inputString), &inputOptions)
	assert.NoError(t, err, "failed to parse JSON dnf config")
	assert.True(t, reflect.DeepEqual(expectedOptions, inputOptions))

	inputBytes, err := json.Marshal(expectedOptions)
	assert.NoError(t, err, "failed to marshal YUM config into JSON")
	assert.Equal(t, inputString, string(inputBytes))
}

func TestDNFConfigValidate(t *testing.T) {
	variables := []DNFVariable{
		{
			Name:  "release",
			Value: "8.4",
		},
	}

	tests := []struct {
		options DNFConfigStageOptions
		valid   bool
	}{
		{
			DNFConfigStageOptions{},
			true,
		},
		{
			DNFConfigStageOptions{
				Variables: variables,
				Config: &DNFConfig{
					Main: nil,
				},
			},
			true,
		},
		{
			DNFConfigStageOptions{
				Variables: variables,
				Config: &DNFConfig{
					Main: &DNFConfigMain{},
				},
			},
			true,
		},
		{
			DNFConfigStageOptions{
				Variables: variables,
				Config: &DNFConfig{
					Main: &DNFConfigMain{
						IPResolve: "4",
					},
				},
			},
			true,
		},
		{
			DNFConfigStageOptions{
				Variables: variables,
				Config: &DNFConfig{
					Main: &DNFConfigMain{
						IPResolve: "urgh",
					},
				},
			},
			false,
		},
	}
	for idx := range tests {
		test := tests[idx]
		if test.valid {
			require.NotPanics(t, func() { NewDNFConfigStage(&test.options) })
		} else {
			require.Panics(t, func() { NewDNFConfigStage(&test.options) })
		}
	}
}

func TestDNFConfigUpdateVar(t *testing.T) {
	type testCase struct {
		options     DNFConfigStageOptions
		newVarName  string
		newVarValue string
		expected    DNFConfigStageOptions
	}

	testCases := map[string]testCase{
		"append-to-empty": {
			options:     DNFConfigStageOptions{},
			newVarName:  "new",
			newVarValue: "new-value",
			expected: DNFConfigStageOptions{
				Variables: []DNFVariable{
					{
						Name:  "new",
						Value: "new-value",
					},
				},
			},
		},
		"append-to-non-empty": {
			options: DNFConfigStageOptions{
				Variables: []DNFVariable{
					{
						Name:  "existing",
						Value: "existing-value",
					},
				},
			},
			newVarName:  "new2",
			newVarValue: "new2-value",
			expected: DNFConfigStageOptions{
				Variables: []DNFVariable{
					{
						Name:  "existing",
						Value: "existing-value",
					},
					{
						Name:  "new2",
						Value: "new2-value",
					},
				},
			},
		},
		"update-value": {
			options: DNFConfigStageOptions{
				Variables: []DNFVariable{
					{
						Name:  "existing",
						Value: "existing-value",
					},
					{
						Name:  "tobeupdated",
						Value: "old-value",
					},
					{
						Name:  "three",
						Value: "third-value",
					},
				},
			},
			newVarName:  "tobeupdated",
			newVarValue: "updated-value",
			expected: DNFConfigStageOptions{
				Variables: []DNFVariable{
					{
						Name:  "existing",
						Value: "existing-value",
					},
					{
						Name:  "tobeupdated",
						Value: "updated-value",
					},
					{
						Name:  "three",
						Value: "third-value",
					},
				},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			input := &tc.options
			input.UpdateVar(tc.newVarName, tc.newVarValue)
			assert.Equal(t, tc.expected, *input)
		})
	}

	assert.PanicsWithError(t, "UpdateVar() call on nil DNFConfigStageOptions", func() {
		var dnf *DNFConfigStageOptions
		dnf.UpdateVar("", "")
	})
}
