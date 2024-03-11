package platform_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/pkg/platform"
)

func TestImageFormatString(t *testing.T) {
	assert.Equal(t, "unset", platform.FORMAT_UNSET.String())
	assert.Equal(t, "ova", platform.FORMAT_OVA.String())
}

func TestImageFormatStringUnknown(t *testing.T) {
	assert.PanicsWithError(t, "unknown image format 999", func() {
		_ = platform.ImageFormat(999).String()
	})
}
