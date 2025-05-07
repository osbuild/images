package arch

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/internal/common"
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

func TestCurrentArchRiscv64(t *testing.T) {
	origRuntimeGOARCH := runtimeGOARCH
	defer func() { runtimeGOARCH = origRuntimeGOARCH }()
	runtimeGOARCH = "riscv64"
	assert.Equal(t, "riscv64", Current().String())
	assert.True(t, IsRISCV64())
}

func TestCurrentArchUnsupported(t *testing.T) {
	origRuntimeGOARCH := runtimeGOARCH
	defer func() { runtimeGOARCH = origRuntimeGOARCH }()
	runtimeGOARCH = "UKNOWN"
	assert.PanicsWithError(t, "unsupported architecture", func() { Current() })
}

func TestFromStringUnsupported(t *testing.T) {
	_, err := FromString("UNKNOWN")
	assert.EqualError(t, err, "unsupported architecture")
}

func TestFromString(t *testing.T) {
	assert.Equal(t, ARCH_AARCH64, common.Must(FromString("arm64")))
	assert.Equal(t, ARCH_AARCH64, common.Must(FromString("aarch64")))
	assert.Equal(t, ARCH_X86_64, common.Must(FromString("amd64")))
	assert.Equal(t, ARCH_X86_64, common.Must(FromString("x86_64")))
	assert.Equal(t, ARCH_S390X, common.Must(FromString("s390x")))
	assert.Equal(t, ARCH_PPC64LE, common.Must(FromString("ppc64le")))
	assert.Equal(t, ARCH_RISCV64, common.Must(FromString("riscv64")))
}
