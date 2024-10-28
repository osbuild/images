package main_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/cmd/image-builder"
)

func TestListImagesSmoke(t *testing.T) {
	t.Setenv("IMAGE_BUILDER_EXTRA_REPOS_PATH", "../../test/data")

	restore := main.MockOsArgs([]string{"list-images"})
	defer restore()

	var fakeStdout bytes.Buffer
	restore = main.MockOsStdout(&fakeStdout)
	defer restore()

	err := main.Run()
	assert.NoError(t, err)
	// output is sorted
	assert.Regexp(t, `(?ms)rhel-8.9.*rhel-8.10`, fakeStdout.String())
}
