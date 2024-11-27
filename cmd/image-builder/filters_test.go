package main_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/cmd/image-builder"
)

func TestGetOneImageHappy(t *testing.T) {
	t.Setenv("IMAGE_BUILDER_EXTRA_REPOS_PATH", "../../test/data")

	res, err := main.GetOneImage("centos-9", "qcow2", "x86_64")
	require.NoError(t, err)
	assert.Equal(t, "centos-9", res.Distro.Name())
	assert.Equal(t, "x86_64", res.Arch.Name())
	assert.Equal(t, "qcow2", res.ImgType.Name())
}

func TestGetOneImageSad(t *testing.T) {
	t.Setenv("IMAGE_BUILDER_EXTRA_REPOS_PATH", "../../test/data")

	_, err := main.GetOneImage("no-distro-meeh", "qcow2", "x86_64")
	require.EqualError(t, err, `cannot find image for: distro:"no-distro-meeh" type:"qcow2" arch:"x86_64"`)
}
