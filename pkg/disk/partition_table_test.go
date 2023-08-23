package disk

import (
	"github.com/osbuild/images/internal/common"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPartitionTableFeatures(t *testing.T) {
	type testCase struct {
		partitionType    string
		expectedFeatures partitionTableFeatures
	}
	testCases := []testCase{
		{"plain", partitionTableFeatures{XFS: true, FAT: true}},
		{"luks", partitionTableFeatures{XFS: true, FAT: true, LUKS: true}},
		{"luks+lvm", partitionTableFeatures{XFS: true, FAT: true, LUKS: true, LVM: true}},
		{"btrfs", partitionTableFeatures{XFS: true, FAT: true, Btrfs: true}},
	}

	for _, tc := range testCases {
		pt := testPartitionTables[tc.partitionType]
		assert.Equal(t, tc.expectedFeatures, pt.features())

	}
}

func TestPartitionTable_GenerateUUIDs(t *testing.T) {
	pt := PartitionTable{
		Type: "gpt",
		Partitions: []Partition{
			{
				Size:     1 * common.MebiByte,
				Bootable: true,
				Type:     BIOSBootPartitionGUID,
				UUID:     BIOSBootPartitionUUID,
			},
			{
				Size: 2 * common.GibiByte,
				Type: FilesystemDataGUID,
				Payload: &Filesystem{
					Type:         "xfs",
					Label:        "root",
					Mountpoint:   "/",
					FSTabOptions: "defaults",
					FSTabFreq:    0,
					FSTabPassNo:  0,
				},
			},
			{
				Size: 10 * common.GibiByte,
				Payload: &Btrfs{
					Subvolumes: []BtrfsSubvolume{
						{
							Mountpoint: "/var",
						},
					},
				},
			},
		},
	}

	// Static seed for testing
	/* #nosec G404 */
	rnd := rand.New(rand.NewSource(0))

	pt.GenerateUUIDs(rnd)

	// Check that GenUUID doesn't change already defined UUIDs
	assert.Equal(t, BIOSBootPartitionUUID, pt.Partitions[0].UUID)

	// Check that GenUUID generates fresh UUIDs if not defined prior the call
	assert.Equal(t, "a178892e-e285-4ce1-9114-55780875d64e", pt.Partitions[1].UUID)
	assert.Equal(t, "6e4ff95f-f662-45ee-a82a-bdf44a2d0b75", pt.Partitions[1].Payload.(*Filesystem).UUID)

	// Check that GenUUID generates the same UUID for BTRFS and its subvolumes
	assert.Equal(t, "fb180daf-48a7-4ee0-b10d-394651850fd4", pt.Partitions[2].Payload.(*Btrfs).UUID)
	assert.Equal(t, "fb180daf-48a7-4ee0-b10d-394651850fd4", pt.Partitions[2].Payload.(*Btrfs).Subvolumes[0].UUID)
}
