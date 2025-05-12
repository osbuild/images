package environment_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"github.com/osbuild/images/internal/environment"
	"github.com/osbuild/images/pkg/rpmmd"
)

func TestEnvironmentConf(t *testing.T) {
	inputYAML := `
packages:
 - pkg1
 - pkg2
repos:
 - baseurls: ["http://example.com"]
services:
 - srv1
`
	var env environment.EnvironmentConf
	err := yaml.Unmarshal([]byte(inputYAML), &env)
	assert.NoError(t, err)
	expected := environment.EnvironmentConf{
		Packages: []string{"pkg1", "pkg2"},
		Repos: []rpmmd.RepoConfig{
			{
				BaseURLs: []string{"http://example.com"},
			},
		},
		Services: []string{"srv1"},
	}
	assert.Equal(t, expected, env)
}
