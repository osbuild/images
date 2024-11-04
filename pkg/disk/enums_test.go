package disk_test

import (
	"testing"

	"github.com/osbuild/images/pkg/disk"
	"github.com/stretchr/testify/assert"
)

func TestEnumPartitionTableType(t *testing.T) {
	enumMap := map[string]disk.PartitionTableType{
		"":    disk.PT_NONE,
		"dos": disk.PT_DOS,
		"gpt": disk.PT_GPT,
	}

	assert := assert.New(t)
	for name, num := range enumMap {
		ptt, err := disk.NewPartitionTableType(name)
		expected := disk.PartitionTableType(num)

		assert.NoError(err)
		assert.Equal(expected, ptt)

		assert.Equal(name, ptt.String())
	}

	// error test: bad value
	badPtt := disk.PartitionTableType(3)
	assert.PanicsWithValue("unknown or unsupported partition table type with enum value 3", func() { _ = badPtt.String() })

	// error test: bad name
	_, err := disk.NewPartitionTableType("not-a-type")
	assert.EqualError(err, "unknown or unsupported partition table type name: not-a-type")
}

func TestEnumFSType(t *testing.T) {
	enumMap := map[string]disk.FSType{
		"":      disk.FS_NONE,
		"vfat":  disk.FS_VFAT,
		"ext4":  disk.FS_EXT4,
		"xfs":   disk.FS_XFS,
		"btrfs": disk.FS_BTRFS,
	}

	assert := assert.New(t)
	for name, num := range enumMap {
		fst, err := disk.NewFSType(name)
		expected := disk.FSType(num)

		assert.NoError(err)
		assert.Equal(expected, fst)

		assert.Equal(name, fst.String())
	}

	// error test: bad value
	badFst := disk.FSType(5)
	assert.PanicsWithValue("unknown or unsupported filesystem type with enum value 5", func() { _ = badFst.String() })

	// error test: bad name
	_, err := disk.NewFSType("not-a-type")
	assert.EqualError(err, "unknown or unsupported filesystem type name: not-a-type")
}
