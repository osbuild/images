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

func TestFromStringUnsupported(t *testing.T) {
	assert.PanicsWithValue(t, "unsupported architecture", func() { FromString("UNKNOWN") })
}

func TestFromString(t *testing.T) {
	assert.Equal(t, ARCH_AARCH64, FromString("arm64"))
	assert.Equal(t, ARCH_AARCH64, FromString("aarch64"))
	assert.Equal(t, ARCH_X86_64, FromString("amd64"))
	assert.Equal(t, ARCH_X86_64, FromString("x86_64"))
	assert.Equal(t, ARCH_S390X, FromString("s390x"))
	assert.Equal(t, ARCH_PPC64LE, FromString("ppc64le"))
}
