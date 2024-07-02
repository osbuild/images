package otkdisk_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/internal/otkdisk"
)

func TestPartTypeValidationHappy(t *testing.T) {
	assert.NoError(t, otkdisk.PartTypeDOS.Validate())
	assert.NoError(t, otkdisk.PartTypeGPT.Validate())
}

func TestPartTypeValidationSad(t *testing.T) {
	assert.EqualError(t, otkdisk.PartTypeUnset.Validate(), `unsupported partition type ""`)
	assert.EqualError(t, otkdisk.PartType("foo").Validate(), `unsupported partition type "foo"`)
}
