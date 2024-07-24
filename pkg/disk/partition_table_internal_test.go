package disk

import (
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

// validatePTSize checks that each Partition is large enough to contain every
// sizeable under it.
func validatePTSize(pt *PartitionTable) error {
	ptTotal := uint64(0)
	for _, partition := range pt.Partitions {
		if err := validateEntitySize(&partition, partition.GetSize()); err != nil {
			return err
		}
		ptTotal += partition.GetSize()
	}

	if pt.GetSize() < ptTotal {
		return fmt.Errorf("PartitionTable size %d is smaller than the sum of its partitions %d", pt.GetSize(), ptTotal)
	}
	return nil
}

// validateEntitySize checks that every sizeable under a given Entity can be
// contained in the given size.
func validateEntitySize(ent Entity, size uint64) error {
	if ent.IsContainer() {
		containerTotal := uint64(0)
		cont := ent.(Container)
		for idx := uint(0); idx < cont.GetItemCount(); idx++ {
			child := cont.GetChild(idx)
			var childSize uint64
			if sizeable, convOk := child.(Sizeable); convOk {
				childSize = sizeable.GetSize()
				containerTotal += childSize
			} else {
				// child is not sizeable: use the parent size
				childSize = size
			}
			if err := validateEntitySize(child, childSize); err != nil {
				return err
			}
		}

		if size < containerTotal {
			return fmt.Errorf("Entity size %d is smaller than the sum of its children %d", size, containerTotal)
		}
	}
	// non-containers need no checking
	return nil
}
