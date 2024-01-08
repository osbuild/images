package image_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/image"
)

func TestBootcDiskImageNew(t *testing.T) {
	containerSource := container.SourceSpec{
		Source: "source-spec",
		Name:   "name",
	}

	img := image.NewBootcDiskImage(containerSource)
	require.NotNil(t, img)
	assert.Equal(t, img.OSTreeDiskImage.Base.Name(), "bootc-raw-image")
}
