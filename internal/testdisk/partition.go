package testdisk

import (
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
