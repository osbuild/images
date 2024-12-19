package spec_test

import (
	"embed"
	"os"
	"testing"

	"github.com/osbuild/images/pkg/spec"
	"github.com/stretchr/testify/assert"
)

//go:embed testdata
var testData embed.FS

func TestMergeConfig(t *testing.T) {
	merged, err := spec.MergeConfig(testData, "testdata/derived.yaml")
	assert.NoError(t, err)

	result, err := os.ReadFile("testdata/result.yaml")
	assert.NoError(t, err)

	assert.YAMLEq(t, string(result), string(merged))
}
