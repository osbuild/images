package image_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/pkg/image"
	"github.com/osbuild/images/pkg/manifest"
)

func TestArchiveValidatesTarCompressor(t *testing.T) {
	mf := manifest.New()
	assert.NotNil(t, mf)

	img := image.NewArchive()
	assert.NotNil(t, img)
	img.Compression = "invalid"

	_, err := img.InstantiateManifest(&mf, nil, nil, nil)
	assert.EqualError(t, err, `unsupported compression value "invalid"`)
}
