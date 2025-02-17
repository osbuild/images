package platform_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/platform"
)

func TestPlatformRiscv64Arch(t *testing.T) {
	platform := &platform.RISCV64{}

	assert.Equal(t, arch.ARCH_RISCV64, platform.GetArch())
}
