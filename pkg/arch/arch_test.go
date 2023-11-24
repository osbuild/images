package arch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCurrentArchAMD64(t *testing.T) {
	origRuntimeGOARCH := runtimeGOARCH
	defer func() { runtimeGOARCH = origRuntimeGOARCH }()
	runtimeGOARCH = "amd64"
	assert.Equal(t, "x86_64", Current().String())
	assert.True(t, IsX86_64())
}

func TestCurrentArchARM64(t *testing.T) {
	origRuntimeGOARCH := runtimeGOARCH
	defer func() { runtimeGOARCH = origRuntimeGOARCH }()
	runtimeGOARCH = "arm64"
	assert.Equal(t, "aarch64", Current().String())
	assert.True(t, IsAarch64())
}

func TestCurrentArchPPC64LE(t *testing.T) {
	origRuntimeGOARCH := runtimeGOARCH
	defer func() { runtimeGOARCH = origRuntimeGOARCH }()
	runtimeGOARCH = "ppc64le"
	assert.Equal(t, "ppc64le", Current().String())
	assert.True(t, IsPPC())
}

func TestCurrentArchS390X(t *testing.T) {
	origRuntimeGOARCH := runtimeGOARCH
	defer func() { runtimeGOARCH = origRuntimeGOARCH }()
	runtimeGOARCH = "s390x"
	assert.Equal(t, "s390x", Current().String())
	assert.True(t, IsS390x())
}

func TestCurrentArchUnsupported(t *testing.T) {
	origRuntimeGOARCH := runtimeGOARCH
	defer func() { runtimeGOARCH = origRuntimeGOARCH }()
	runtimeGOARCH = "UKNOWN"
	assert.PanicsWithValue(t, "unsupported architecture", func() { Current() })
}
