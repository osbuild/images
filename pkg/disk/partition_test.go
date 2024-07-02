package disk_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/internal/testdisk"
	"github.com/osbuild/images/pkg/disk"
)

func TestMarshalUnmarshalSimple(t *testing.T) {
	fakePt := testdisk.MakeFakePartitionTable("/", "/boot", "/boot/efi")

	js, err := json.Marshal(fakePt)
	assert.NoError(t, err)

	var ptFromJS disk.PartitionTable
	err = json.Unmarshal(js, &ptFromJS)
	assert.NoError(t, err)
	assert.Equal(t, fakePt, &ptFromJS)
}

func TestMarshalUnmarshalSad(t *testing.T) {
	var part disk.Partition
	err := json.Unmarshal([]byte(`{"randon": "json"}`), &part)
	assert.ErrorContains(t, err, `cannot build partition from "{`)
}

func TestMarshalUnmarshalPartitionHappy(t *testing.T) {
	part := &disk.Partition{}

	for _, ent := range []disk.Entity{
		&disk.Filesystem{Type: "ext2"},
		&disk.LUKSContainer{Passphrase: "secret"},
		&disk.Btrfs{Label: "foo"},
		&disk.LVMVolumeGroup{Name: "bar"},
	} {
		part.Payload = ent
		js, err := json.Marshal(part)
		assert.NoError(t, err)

		var partFromJS disk.Partition
		err = json.Unmarshal(js, &partFromJS)
		assert.NoError(t, err)
		assert.Equal(t, part, &partFromJS)
	}
}

func TestUnmarshalNullPayload(t *testing.T) {
	part := &disk.Partition{}
	part.Payload = nil

	js, err := json.Marshal(part)
	assert.NoError(t, err)

	var partFromJS disk.Partition
	err = json.Unmarshal(js, &partFromJS)
	assert.NoError(t, err)
	assert.Equal(t, part, &partFromJS)
}
