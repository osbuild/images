package testdisk

import (
	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/disk"
)

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
	for _, mntPoint := range mntPoints {
		name := mntPoint
		if name == "/" {
			name = "root"
		}
		subvolumes = append(subvolumes, disk.BtrfsSubvolume{Mountpoint: mntPoint, Name: name, Compress: disk.DefaultBtrfsCompression})
	}

	return &disk.PartitionTable{
		Type: "gpt",
		Size: 10 * common.GiB,
		Partitions: []disk.Partition{
			{
				Size: 1 * common.GiB,
				Payload: &disk.Filesystem{
					Type:       "ext4",
					Mountpoint: "/boot",
				},
			},
			{
				Start: 1 * common.GiB,
				Size:  9 * common.GiB,
				Payload: &disk.Btrfs{
					UUID:       disk.RootPartitionUUID,
					Subvolumes: subvolumes,
				},
			},
		},
	}
}
