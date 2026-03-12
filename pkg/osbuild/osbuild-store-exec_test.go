package osbuild_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/pkg/osbuild"
)

func TestRunOSBuildStore(t *testing.T) {
	cmd := makeFakeOSBuild(t, `
echo -n "arguments: $@"
`)

	restore := osbuild.MockOSBuildStoreCmd(cmd)
	defer restore()

	out, err := osbuild.RunOSBuildStore(nil, "src", "tgt")
	assert.NoError(t, err)
	assert.Equal(t, "arguments: export-sources --source-store src --target-store tgt -", out)
}
