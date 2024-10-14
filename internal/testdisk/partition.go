package testdisk

import (
	"github.com/osbuild/images/pkg/datasizes"
	"github.com/osbuild/images/pkg/disk"
)

const FakePartitionSize = uint64(789) * datasizes.MiB

// MakeFakePartitionTable is a helper to create partition table structs
// for tests. It uses sensible defaults for common scenarios.
func MakeFakePartitionTable(mntPoints ...string) *disk.PartitionTable {
	var partitions []disk.Partition
	for _, mntPoint := range mntPoints {
		payload := &disk.Filesystem{
			Type:       "ext4",
			Mountpoint: mntPoint,
		}
		switch mntPoint {
		case "/":
			payload.UUID = disk.RootPartitionUUID
		case "/boot/efi":
			payload.UUID = disk.EFIFilesystemUUID
			payload.Type = "vfat"
		default:
			payload.UUID = disk.FilesystemDataUUID
		}
		partitions = append(partitions, disk.Partition{
			Size:    FakePartitionSize,
			Payload: payload,
		})

	}
	return &disk.PartitionTable{
		Type:       "gpt",
		Partitions: partitions,
	}
}

// MakeFakeBtrfsPartitionTable is similar to MakeFakePartitionTable but
// creates a btrfs-based partition table.
func MakeFakeBtrfsPartitionTable(mntPoints ...string) *disk.PartitionTable {
	var subvolumes []disk.BtrfsSubvolume
	pt := &disk.PartitionTable{
		Type:       "gpt",
		Size:       10 * datasizes.GiB,
		Partitions: []disk.Partition{},
	}
	size := uint64(0)
	for _, mntPoint := range mntPoints {
		switch mntPoint {
		case "/boot":
			pt.Partitions = append(pt.Partitions, disk.Partition{
				Start: size,
				Size:  1 * datasizes.GiB,
				Payload: &disk.Filesystem{
					Type:       "ext4",
					Mountpoint: mntPoint,
				},
			})
			size += 1 * datasizes.GiB
		case "/boot/efi":
			pt.Partitions = append(pt.Partitions, disk.Partition{
				Start: size,
				Size:  100 * datasizes.MiB,
				Payload: &disk.Filesystem{
					Type:       "vfat",
					Mountpoint: mntPoint,
					UUID:       disk.EFIFilesystemUUID,
				},
			})
			size += 100 * datasizes.MiB
		default:
			name := mntPoint
			if name == "/" {
				name = "root"
			}
			subvolumes = append(
				subvolumes,
				disk.BtrfsSubvolume{
					Mountpoint: mntPoint,
					Name:       name,
					UUID:       disk.RootPartitionUUID,
					Compress:   disk.DefaultBtrfsCompression,
				},
			)
		}
	}

	pt.Partitions = append(pt.Partitions, disk.Partition{
		Start: size,
		Size:  9 * datasizes.GiB,
		Payload: &disk.Btrfs{
			UUID:       disk.RootPartitionUUID,
			Subvolumes: subvolumes,
		},
	})

	size += 9 * datasizes.GiB
	pt.Size = size

	return pt
}

// MakeFakeLVMPartitionTable is similar to MakeFakePartitionTable but
// creates a lvm-based partition table.
func MakeFakeLVMPartitionTable(mntPoints ...string) *disk.PartitionTable {
	var lvs []disk.LVMLogicalVolume
	pt := &disk.PartitionTable{
		Type:       "gpt",
		Size:       10 * datasizes.GiB,
		Partitions: []disk.Partition{},
	}
	size := uint64(0)
	for _, mntPoint := range mntPoints {
		switch mntPoint {
		case "/boot":
			pt.Partitions = append(pt.Partitions, disk.Partition{
				Start: size,
				Size:  1 * datasizes.GiB,
				Payload: &disk.Filesystem{
					Type:       "ext4",
					Mountpoint: mntPoint,
				},
			})
			size += 1 * datasizes.GiB
		case "/boot/efi":
			pt.Partitions = append(pt.Partitions, disk.Partition{
				Start: size,
				Size:  100 * datasizes.MiB,
				Payload: &disk.Filesystem{
					Type:       "vfat",
					Mountpoint: mntPoint,
					UUID:       disk.EFIFilesystemUUID,
				},
			})
			size += 100 * datasizes.MiB
		default:
			name := "lv-for-" + mntPoint
			if name == "/" {
				name = "lvroot"
			}
			lvs = append(
				lvs,
				disk.LVMLogicalVolume{
					Name: name,
					Payload: &disk.Filesystem{
						Type:       "xfs",
						Mountpoint: mntPoint,
					},
				},
			)
		}
	}

	pt.Partitions = append(pt.Partitions, disk.Partition{
		Start: size,
		Size:  9 * datasizes.GiB,
		Payload: &disk.LVMVolumeGroup{
			LogicalVolumes: lvs,
		},
	})

	size += 9 * datasizes.GiB
	pt.Size = size

	return pt
}
