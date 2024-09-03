package otkdisk_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/internal/otkdisk"
	"github.com/osbuild/images/pkg/disk"
)

func TestPartTypeValidationHappy(t *testing.T) {
	assert.NoError(t, otkdisk.PartTypeDOS.Validate())
	assert.NoError(t, otkdisk.PartTypeGPT.Validate())
}

func TestPartTypeValidationSad(t *testing.T) {
	assert.EqualError(t, otkdisk.PartTypeUnset.Validate(), `unsupported partition type ""`)
	assert.EqualError(t, otkdisk.PartType("foo").Validate(), `unsupported partition type "foo"`)
}

func TestDataValidates(t *testing.T) {
	// sad
	d := otkdisk.Data{}
	assert.EqualError(t, d.Validate(), "no partition table")
	// happy
	d.Const.Internal.PartitionTable = &disk.PartitionTable{}
	err := d.Validate()
	assert.NoError(t, err)
}
