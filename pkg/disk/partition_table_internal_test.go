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
	if cont, ok := ent.(Container); ok {
		containerTotal := uint64(0)
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
		"btrfs": {
			pt: &PartitionTable{
				Type: "gpt",
				Size: 100 * MiB,
				Partitions: []Partition{
					{
						Size: 20 * MiB,
					},
					{
						Size: 30 * MiB,
						Payload: &Btrfs{
							Subvolumes: []BtrfsSubvolume{
								{
									Size: 20 * MiB,
								},
								{
									Mountpoint: "/",
									Size:       10 * MiB,
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
						Payload: &Btrfs{
							Subvolumes: []BtrfsSubvolume{
								{
									Size: 20 * MiB,
								},
								{
									Mountpoint: "/",
									Size:       10 * MiB, // We don't automatically grow the root subvolume
								},
							},
						},
					},
				},
			},
		},
		"simple-dos-grow-pt": {
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
						Size: 200 * MiB,
					},
				},
			},
			size: 100 * MiB,
			expected: &PartitionTable{
				Type: "dos",
				Size: 211 * MiB, // grows to fit partitions and header
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
						Size:  200 * MiB,
					},
				},
			},
		},
		"simple-gpt-growpt": {
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
						Size: 500 * MiB,
					},
				},
			},
			size: 42 * MiB,
			expected: &PartitionTable{
				Type: "gpt",
				Size: 512 * MiB, // grows to fit partitions, header, and footer
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
						Size:  501*MiB - (DefaultSectorSize + (128 * 128)), // grows by (1 MiB - footer) so that the partition doesn't shrink below the desired root size
					},
				},
			},
		},
		"lvm-gpt-grow": {
			pt: &PartitionTable{
				Type: "gpt",
				Size: 10 * MiB,
				Partitions: []Partition{
					{
						Size: 200 * MiB,
					},
					{
						Size: 500 * MiB,
						Payload: &LVMVolumeGroup{
							LogicalVolumes: []LVMLogicalVolume{
								{
									Size: 20 * MiB,
								},
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
				Size: 702 * MiB,
				Partitions: []Partition{
					{
						Start: 1 * MiB, // 1 sector header aligned up to the default grain (1 MiB)
						Size:  200 * MiB,
					},
					{
						Start: 201 * MiB,
						Size:  501*MiB - (DefaultSectorSize + (128 * 128)), // grows by (1 MiB - footer) so that the partition doesn't shrink below the desired root size
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
		"lvm-dos-grow-rootvg": {
			pt: &PartitionTable{
				Type: "dos",
				Size: 10 * MiB, // PT is smaller than the sum of Partitions
				Partitions: []Partition{
					{
						Size: 200 * MiB,
					},
					{
						Size: 10 * MiB, // VG partition is smaller than sum of LVs
						Payload: &LVMVolumeGroup{
							LogicalVolumes: []LVMLogicalVolume{
								{
									Size: 20 * MiB,
								},
								{
									Payload: &Filesystem{
										Mountpoint: "/",
									},
									Size: 100 * MiB,
								},
							},
						},
					},
				},
			},
			size: 99 * MiB,
			expected: &PartitionTable{
				Type: "dos",
				Size: 325 * MiB,
				Partitions: []Partition{
					{
						Start: 1 * MiB, // 1 sector header aligned up to the default grain (1 MiB)
						Size:  200 * MiB,
					},
					{
						Start: 201 * MiB,
						Size:  124 * MiB, // grows to fit logical volumes + 1 MiB metadata, rounded up to default extent size (4 MiB)
						Payload: &LVMVolumeGroup{
							LogicalVolumes: []LVMLogicalVolume{
								{
									Size: 20 * MiB,
								},
								{
									Payload: &Filesystem{
										Mountpoint: "/",
									},
									Size: 100 * MiB,
								},
							},
						},
					},
				},
			},
		},
		"lvm-gpt-grow-rootvg": {
			pt: &PartitionTable{
				Type: "gpt",
				Size: 10 * MiB,
				Partitions: []Partition{
					{
						Size: 200 * MiB,
					},
					{
						Size: 10 * MiB,
						Payload: &LVMVolumeGroup{
							LogicalVolumes: []LVMLogicalVolume{
								{
									Size: 20 * MiB,
								},
								{
									Payload: &Filesystem{
										Mountpoint: "/",
									},
									Size: 100 * MiB,
								},
							},
						},
					},
				},
			},
			size: 99 * MiB,
			expected: &PartitionTable{
				Type: "gpt",
				Size: 326 * MiB,
				Partitions: []Partition{
					{
						Start: 1 * MiB, // 1 sector header aligned up to the default grain (1 MiB)
						Size:  200 * MiB,
					},
					{
						Start: 201 * MiB,
						Size:  125*MiB - (DefaultSectorSize + (128 * 128)), // grows to fit logical volumes and metadata, rounded up to default extent size + (1 MiB - footer) so that the no partitions shrink below the desired sizes
						Payload: &LVMVolumeGroup{
							LogicalVolumes: []LVMLogicalVolume{
								{
									Size: 20 * MiB,
								},
								{
									Payload: &Filesystem{
										Mountpoint: "/",
									},
									Size: 100 * MiB,
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

func TestEnsureRootFilesystem(t *testing.T) {
	type testCase struct {
		pt       PartitionTable
		expected PartitionTable
		options  *CustomPartitionTableOptions
		errmsg   string
	}

	testCases := map[string]testCase{
		"empty-plain": {
			pt: PartitionTable{},
			options: &CustomPartitionTableOptions{
				DefaultFSType: FS_EXT4,
			},
			expected: PartitionTable{
				Partitions: []Partition{
					{
						Start:    0,
						Size:     0,
						Type:     FilesystemDataGUID,
						Bootable: false,
						UUID:     "",
						Payload: &Filesystem{
							Type:         "ext4",
							Label:        "root",
							Mountpoint:   "/",
							FSTabOptions: "defaults",
						},
					},
				},
			},
		},
		"simple-plain": {
			pt: PartitionTable{
				Partitions: []Partition{
					{
						Payload: &Filesystem{
							Type:         "ext4",
							Label:        "home",
							Mountpoint:   "/home",
							FSTabOptions: "defaults",
						},
					},
				},
			},
			options: &CustomPartitionTableOptions{
				DefaultFSType: FS_EXT4,
			},
			expected: PartitionTable{
				Partitions: []Partition{
					{
						Payload: &Filesystem{
							Type:         "ext4",
							Label:        "home",
							Mountpoint:   "/home",
							FSTabOptions: "defaults",
						},
					},
					{
						Start:    0,
						Size:     0,
						Type:     FilesystemDataGUID,
						Bootable: false,
						UUID:     "",
						Payload: &Filesystem{
							Type:         "ext4",
							Label:        "root",
							Mountpoint:   "/",
							FSTabOptions: "defaults",
						},
					},
				},
			},
		},
		"simple-lvm": {
			pt: PartitionTable{
				Partitions: []Partition{
					{
						Payload: &LVMVolumeGroup{
							Name: "testvg",
							LogicalVolumes: []LVMLogicalVolume{
								{
									Name: "varloglv",
									Payload: &Filesystem{
										Label:      "var-log",
										Type:       "xfs",
										Mountpoint: "/var/log",
									},
								},
								{
									Name: "datalv",
									Payload: &Filesystem{
										Label:        "data",
										Mountpoint:   "/data",
										FSTabOptions: "defaults",
										Type:         "ext4",
									},
								},
							},
						},
					},
				},
			},
			options: &CustomPartitionTableOptions{
				DefaultFSType: FS_EXT4,
			},
			expected: PartitionTable{
				Partitions: []Partition{
					{
						Payload: &LVMVolumeGroup{
							Name: "testvg",
							LogicalVolumes: []LVMLogicalVolume{
								{
									Name: "varloglv",
									Payload: &Filesystem{
										Label:      "var-log",
										Type:       "xfs",
										Mountpoint: "/var/log",
									},
								},
								{
									Name: "datalv",
									Payload: &Filesystem{
										Label:        "data",
										Type:         "ext4",
										Mountpoint:   "/data",
										FSTabOptions: "defaults",
									},
								},
								{
									Name: "rootlv",
									Payload: &Filesystem{
										Label:        "root",
										Type:         "ext4",
										Mountpoint:   "/",
										FSTabOptions: "defaults",
									},
								},
							},
						},
					},
				},
			},
		},
		"simple-btrfs": {
			pt: PartitionTable{
				Partitions: []Partition{
					{
						Payload: &Btrfs{
							Subvolumes: []BtrfsSubvolume{
								{
									Name:       "subvol/home",
									Mountpoint: "/home",
								},
							},
						},
					},
				},
			},
			options: &CustomPartitionTableOptions{
				// no default fs required
			},
			expected: PartitionTable{
				Partitions: []Partition{
					{
						Payload: &Btrfs{
							Subvolumes: []BtrfsSubvolume{
								{
									Name:       "subvol/home",
									Mountpoint: "/home",
								},
								{
									Name:       "root",
									Mountpoint: "/",
								},
							},
						},
					},
				},
			},
		},
		"noop-lvm": {
			pt: PartitionTable{
				Partitions: []Partition{
					{
						Payload: &LVMVolumeGroup{
							Name: "testvg",
							LogicalVolumes: []LVMLogicalVolume{
								{
									Name: "varloglv",
									Payload: &Filesystem{
										Label:      "var-log",
										Type:       "xfs",
										Mountpoint: "/var/log",
									},
								},
								{
									Name: "datalv",
									Payload: &Filesystem{
										Label:        "data",
										Type:         "ext4",
										Mountpoint:   "/data",
										FSTabOptions: "defaults",
									},
								},
								{
									Name: "rootlv",
									Payload: &Filesystem{
										Label:        "root",
										Type:         "ext4",
										Mountpoint:   "/",
										FSTabOptions: "defaults",
									},
								},
							},
						},
					},
				},
			},
			expected: PartitionTable{
				Partitions: []Partition{
					{
						Payload: &LVMVolumeGroup{
							Name: "testvg",
							LogicalVolumes: []LVMLogicalVolume{
								{
									Name: "varloglv",
									Payload: &Filesystem{
										Label:      "var-log",
										Type:       "xfs",
										Mountpoint: "/var/log",
									},
								},
								{
									Name: "datalv",
									Payload: &Filesystem{
										Label:        "data",
										Type:         "ext4",
										Mountpoint:   "/data",
										FSTabOptions: "defaults",
									},
								},
								{
									Name: "rootlv",
									Payload: &Filesystem{
										Label:        "root",
										Type:         "ext4",
										Mountpoint:   "/",
										FSTabOptions: "defaults",
									},
								},
							},
						},
					},
				},
			},
		},
		"noop-btrfs": {
			pt: PartitionTable{
				Partitions: []Partition{
					{
						Payload: &Btrfs{
							Subvolumes: []BtrfsSubvolume{
								{
									Name:       "subvol/home",
									Mountpoint: "/home",
								},
								{
									Name:       "root",
									Mountpoint: "/",
								},
							},
						},
					},
				},
			},
			expected: PartitionTable{
				Partitions: []Partition{
					{
						Payload: &Btrfs{
							Subvolumes: []BtrfsSubvolume{
								{
									Name:       "subvol/home",
									Mountpoint: "/home",
								},
								{
									Name:       "root",
									Mountpoint: "/",
								},
							},
						},
					},
				},
			},
		},
		"err-empty": {
			pt:      PartitionTable{},
			options: &CustomPartitionTableOptions{},
			errmsg:  "error creating root partition: no default filesystem type",
		},
		"err-plain": {
			pt: PartitionTable{
				Partitions: []Partition{
					{
						Payload: &Filesystem{
							Type:         "ext4",
							Label:        "home",
							Mountpoint:   "/home",
							FSTabOptions: "defaults",
						},
					},
				},
			},
			errmsg: "error creating root partition: no default filesystem type",
		},
		"err-lvm": {
			pt: PartitionTable{
				Partitions: []Partition{
					{
						Payload: &LVMVolumeGroup{
							Name: "testvg",
							LogicalVolumes: []LVMLogicalVolume{
								{
									Name: "varloglv",
									Payload: &Filesystem{
										Label:      "var-log",
										Type:       "xfs",
										Mountpoint: "/var/log",
									},
								},
								{
									Name: "datalv",
									Payload: &Filesystem{
										Label:        "data",
										Mountpoint:   "/data",
										FSTabOptions: "defaults",
										Type:         "ext4",
									},
								},
							},
						},
					},
				},
			},
			errmsg: "error creating root logical volume: no default filesystem type",
		},
	}

	for name := range testCases {
		tc := testCases[name]
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			pt := tc.pt
			err := ensureRootFilesystem(&pt, tc.options)
			if tc.errmsg == "" {
				assert.NoError(err)
				assert.Equal(tc.expected, pt)
			} else {
				assert.EqualError(err, tc.errmsg)
			}
		})
	}
}
