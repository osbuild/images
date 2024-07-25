package disk

import (
	"fmt"
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

func TestValidateFunctions(t *testing.T) {
	type testCase struct {
		pt  *PartitionTable
		err error
	}

	testCases := map[string]testCase{
		"happy-simple": {
			pt: &PartitionTable{
				Size: 100,
				Partitions: []Partition{
					{
						Size: 10,
					},
					{
						Size: 20,
					},
				},
			},
			err: nil,
		},
		"happy-nested": {
			pt: &PartitionTable{
				Size: 100,
				Partitions: []Partition{
					{
						Size: 10,
					},
					{
						Size: 20,
						Payload: &LVMVolumeGroup{
							LogicalVolumes: []LVMLogicalVolume{
								{
									Size: 5,
								},
								{
									Size: 8,
								},
							},
						},
					},
				},
			},
			err: nil,
		},
		"happy-btrfs": {
			pt: &PartitionTable{
				Size: 100,
				Partitions: []Partition{
					{
						Size: 10,
					},
					{
						Size: 20,
						Payload: &Btrfs{
							Subvolumes: []BtrfsSubvolume{
								{
									Size: 4,
								},
								{
									Size: 2,
								},
							},
						},
					},
				},
			},
			err: nil,
		},
		"unhappy-simple": {
			pt: &PartitionTable{
				Size: 10,
				Partitions: []Partition{
					{
						Size: 10,
					},
					{
						Size: 20,
					},
				},
			},
			err: fmt.Errorf("PartitionTable size 10 is smaller than the sum of its partitions 30"),
		},
		"unhappy-nested": {
			pt: &PartitionTable{
				Size: 100,
				Partitions: []Partition{
					{
						Size: 10,
					},
					{
						Size: 20,
						Payload: &LVMVolumeGroup{
							LogicalVolumes: []LVMLogicalVolume{
								{
									Size: 15,
								},
								{
									Size: 8,
								},
							},
						},
					},
				},
			},
			err: fmt.Errorf("Entity size 20 is smaller than the sum of its children 23"),
		},
		"unhappy-nested-luks": {
			pt: &PartitionTable{
				Size: 100,
				Partitions: []Partition{
					{
						Size: 10,
					},
					{
						Size: 20,
						Payload: &LUKSContainer{
							Payload: &LVMVolumeGroup{
								LogicalVolumes: []LVMLogicalVolume{
									{
										Size: 15,
									},
									{
										Size: 8,
									},
								},
							},
						},
					},
				},
			},
			err: fmt.Errorf("Entity size 20 is smaller than the sum of its children 23"),
		},
		"unhappy-btrfs": {
			pt: &PartitionTable{
				Size: 100,
				Partitions: []Partition{
					{
						Size: 10,
					},
					{
						Size: 20,
						Payload: &Btrfs{
							Subvolumes: []BtrfsSubvolume{
								{
									Size: 10,
								},
								{
									Size: 10,
								},
								{
									Size: 1,
								},
							},
						},
					},
				},
			},
			err: fmt.Errorf("Entity size 20 is smaller than the sum of its children 21"),
		},
	}

	for name := range testCases {
		tc := testCases[name]
		t.Run(name, func(t *testing.T) {
			err := validatePTSize(tc.pt)
			assert.Equal(t, tc.err, err)
		})
	}
}

func TestRelayout(t *testing.T) {
	type testCase struct {
		pt       *PartitionTable
		size     uint64
		expected *PartitionTable
	}

	testCases := map[string]testCase{
		"simple-dos": {
			pt: &PartitionTable{
				Type: "dos",
				Size: 100 * MiB,
				Partitions: []Partition{
					{
						Size: 10 * MiB,
					},
					{
						Payload: &Filesystem{
							Mountpoint: "/",
						},
						Size: 20 * MiB,
					},
				},
			},
			size: 100 * MiB,
			expected: &PartitionTable{
				Type: "dos",
				Size: 100 * MiB,
				Partitions: []Partition{
					{
						Start: 1 * MiB, // 1 sector header aligned up to the default grain (1 MiB)
						Size:  10 * MiB,
					},
					{
						Payload: &Filesystem{
							Mountpoint: "/",
						},
						Start: 11 * MiB,
						Size:  89 * MiB, // Grows to fill the space
					},
				},
			},
		},
		"simple-gpt": {
			pt: &PartitionTable{
				Type: "gpt",
				Size: 100 * MiB,
				Partitions: []Partition{
					{
						Size: 10 * MiB,
					},
					{
						Payload: &Filesystem{
							Mountpoint: "/",
						},
						Size: 20 * MiB,
					},
				},
			},
			size: 100 * MiB,
			expected: &PartitionTable{
				Type: "gpt",
				Size: 100 * MiB,
				Partitions: []Partition{
					{
						Start: 1 * MiB, // header (1 sector + 128 B * 128 partitions) aligned up to the default grain (1 MiB)
						Size:  10 * MiB,
					},
					{
						Payload: &Filesystem{
							Mountpoint: "/",
						},
						Start: 11 * MiB,
						Size:  89*MiB - (DefaultSectorSize + (128 * 128)), // Grows to fill the space, but gpt adds a footer the same size as the header (unaligned)
					},
				},
			},
		},
		"simple-gpt-root-first": {
			pt: &PartitionTable{
				Type: "gpt",
				Size: 100 * MiB,
				Partitions: []Partition{
					{
						Size: 10 * MiB,
						Payload: &Filesystem{
							Mountpoint: "/",
						},
					},
					{
						Size: 20 * MiB,
					},
					{
						Size: 30 * MiB,
					},
				},
			},
			size: 100 * MiB,
			expected: &PartitionTable{
				Type: "gpt",
				Size: 100 * MiB,
				Partitions: []Partition{
					{
						Start: 51 * MiB,                                   // root gets moved to last position
						Size:  49*MiB - (DefaultSectorSize + (128 * 128)), // Grows to fill the space, but gpt adds a footer the same size as the header (unaligned)
						Payload: &Filesystem{
							Mountpoint: "/",
						},
					},
					{
						Start: 1 * MiB, // header (1 sector + 128 B * 128 partitions) aligned up to the default grain (1 MiB)
						Size:  20 * MiB,
					},
					{
						Start: 21 * MiB, // header (1 sector + 128 B * 128 partitions) aligned up to the default grain (1 MiB)
						Size:  30 * MiB,
					},
				},
			},
		},
		"lvm-dos": {
			pt: &PartitionTable{
				Type: "dos",
				Size: 100 * MiB,
				Partitions: []Partition{
					{
						Size: 20 * MiB,
					},
					{
						Size: 30 * MiB,
						Payload: &LVMVolumeGroup{
							LogicalVolumes: []LVMLogicalVolume{
								{
									Payload: &Filesystem{
										Mountpoint: "/",
									},
								},
							},
						},
					},
				},
			},
			size: 100 * MiB,
			expected: &PartitionTable{
				Type: "dos",
				Size: 100 * MiB,
				Partitions: []Partition{
					{
						Start: 1 * MiB, // 1 sector header aligned up to the default grain (1 MiB)
						Size:  20 * MiB,
					},
					{
						Start: 21 * MiB,
						Size:  79 * MiB, // Grows to fill the space
						Payload: &LVMVolumeGroup{
							LogicalVolumes: []LVMLogicalVolume{
								{
									Payload: &Filesystem{
										Mountpoint: "/",
									},
								},
							},
						},
					},
				},
			},
		},
		"lvm-gpt": {
			pt: &PartitionTable{
				Type: "gpt",
				Size: 100 * MiB,
				Partitions: []Partition{
					{
						Size: 20 * MiB,
					},
					{
						Size: 30 * MiB,
						Payload: &LVMVolumeGroup{
							LogicalVolumes: []LVMLogicalVolume{
								{
									Payload: &Filesystem{
										Mountpoint: "/",
									},
									Size: 10 * MiB,
								},
							},
						},
					},
				},
			},
			size: 100 * MiB,
			expected: &PartitionTable{
				Type: "gpt",
				Size: 100 * MiB,
				Partitions: []Partition{
					{
						Start: 1 * MiB, // 1 sector header aligned up to the default grain (1 MiB)
						Size:  20 * MiB,
					},
					{
						Start: 21 * MiB,
						Size:  79*MiB - (DefaultSectorSize + (128 * 128)), // Grows to fill the space, but gpt adds a footer the same size as the header (unaligned)
						Payload: &LVMVolumeGroup{
							LogicalVolumes: []LVMLogicalVolume{
								{
									Payload: &Filesystem{
										Mountpoint: "/",
									},
									Size: 10 * MiB, // We don't automatically grow the root LV
								},
							},
						},
					},
				},
			},
		},
		"lvm-gpt-multilv": {
			pt: &PartitionTable{
				Type: "gpt",
				Size: 100 * MiB,
				Partitions: []Partition{
					{
						Size: 20 * MiB,
					},
					{
						Size: 30 * MiB,
						Payload: &LVMVolumeGroup{
							LogicalVolumes: []LVMLogicalVolume{
								{
									Size: 20 * MiB,
								},
								{
									Payload: &Filesystem{
										Mountpoint: "/",
									},
									// TODO: fix bug where the VG partition is not resized to fit the sum of LVs
									Size: 10 * MiB,
								},
							},
						},
					},
				},
			},
			size: 100 * MiB,
			expected: &PartitionTable{
				Type: "gpt",
				Size: 100 * MiB,
				Partitions: []Partition{
					{
						Start: 1 * MiB, // 1 sector header aligned up to the default grain (1 MiB)
						Size:  20 * MiB,
					},
					{
						Start: 21 * MiB,
						Size:  79*MiB - (DefaultSectorSize + (128 * 128)), // Grows to fill the space, but gpt adds a footer the same size as the header (unaligned)
						Payload: &LVMVolumeGroup{
							LogicalVolumes: []LVMLogicalVolume{
								{
									Size: 20 * MiB,
								},
								{
									Payload: &Filesystem{
										Mountpoint: "/",
									},
									Size: 10 * MiB, // We don't automatically grow the root LV
								},
							},
						},
					},
				},
			},
		},
	}

	for name := range testCases {
		tc := testCases[name]
		t.Run(name, func(t *testing.T) {
			pt := tc.pt
			pt.relayout(tc.size)
			err := validatePTSize(pt)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, pt)
		})
	}
}
