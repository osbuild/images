package workload_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"github.com/osbuild/images/internal/workload"
	"github.com/osbuild/images/pkg/rpmmd"
)

func TestWorkloadConfUnmarshalAndGetters(t *testing.T) {
	yamlInput := `
packages:
  - foo
  - bar
modules:
  - mod1
  - mod2
repos:
  - id: repo1
    name: Repo 1
    baseurls:
      - http://example.com/repo1
enabled_services:
  - svc1
  - svc2
disabled_services:
  - svc3
masked_services:
  - svc4
`
	var wl workload.WorkloadConf
	err := yaml.Unmarshal([]byte(yamlInput), &wl)
	assert.NoError(t, err)

	assert.Equal(t, []string{"foo", "bar"}, wl.GetPackages())
	assert.Equal(t, []string{"mod1", "mod2"}, wl.GetEnabledModules())
	assert.Equal(t, []rpmmd.RepoConfig{
		{Id: "repo1", Name: "Repo 1", BaseURLs: []string{"http://example.com/repo1"}},
	}, wl.GetRepos())
	assert.Equal(t, []string{"svc1", "svc2"}, wl.GetServices())
	assert.Equal(t, []string{"svc3"}, wl.GetDisabledServices())
	assert.Equal(t, []string{"svc4"}, wl.GetMaskedServices())
}
