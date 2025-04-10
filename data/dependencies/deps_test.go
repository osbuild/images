package dependencies_test

import (
	"testing"

	"github.com/osbuild/images/data/dependencies"
	"github.com/stretchr/testify/assert"
)

func TestMinimumOSBuildVersion(t *testing.T) {
	assert.Equal(t, "147", dependencies.MinimumOSBuildVersion())
}
