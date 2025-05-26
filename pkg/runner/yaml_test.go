package runner_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"github.com/osbuild/images/pkg/runner"
)

func TestRunnerFromYaml(t *testing.T) {
	inputYAML := `
name: org.osbuild.fedora42
build_packages: ["glibc", "systemd"]
`
	var rc runner.RunnerConf
	err := yaml.Unmarshal([]byte(inputYAML), &rc)
	assert.NoError(t, err)
	assert.Equal(t, rc.String(), "org.osbuild.fedora42")
	assert.Equal(t, rc.GetBuildPackages(), []string{"glibc", "systemd"})
}
