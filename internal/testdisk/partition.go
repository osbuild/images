package testdisk

import (
	"github.com/osbuild/images/pkg/datasizes"
	"github.com/osbuild/images/pkg/disk"
)

const (
	KiB = datasizes.KiB
	MiB = datasizes.MiB
	GiB = datasizes.GiB
)

const FakePartitionSize = uint64(789) * MiB

// TODO: Tidy up and unify TestPartitionTables with the fake partition table
// generators below (MakeFake*). Maybe use NewCustomPartitionTable() to
// generate test partition tables instead.

var TestPartitionTables = map[string]disk.PartitionTable{
	"plain": {
		UUID: "D209C89E-EA5E-4FBD-B161-B461CCE297E0",
		Type: disk.PT_GPT,
		Partitions: []disk.Partition{
			{
				Size:     1 * MiB,
				Bootable: true,
				Type:     disk.BIOSBootPartitionGUID,
				UUID:     disk.BIOSBootPartitionUUID,
			},
			{
				Size: 200 * MiB,
				Type: disk.EFISystemPartitionGUID,
				UUID: disk.EFISystemPartitionUUID,
				Payload: &disk.Filesystem{
					Type:         "vfat",
					UUID:         disk.EFIFilesystemUUID,
					Mountpoint:   "/boot/efi",
					Label:        "EFI-SYSTEM",
					FSTabOptions: "defaults,uid=0,gid=0,umask=077,shortname=winnt",
					FSTabFreq:    0,
					FSTabPassNo:  2,
				},
			},
			{
				Size: 500 * MiB,
				Type: disk.FilesystemDataGUID,
				UUID: disk.FilesystemDataUUID,
				Payload: &disk.Filesystem{
					Type:         "xfs",
					Mountpoint:   "/boot",
					Label:        "boot",
					FSTabOptions: "defaults",
					FSTabFreq:    0,
					FSTabPassNo:  0,
				},
			},
			{
				Type: disk.FilesystemDataGUID,
				UUID: disk.RootPartitionUUID,
				Payload: &disk.Filesystem{
					Type:         "xfs",
					Label:        "root",
					Mountpoint:   "/",
					FSTabOptions: "defaults",
					FSTabFreq:    0,
					FSTabPassNo:  0,
				},
			},
		},
	},

	"plain-swap": {
		UUID: "D209C89E-EA5E-4FBD-B161-B461CCE297E0",
		Type: disk.PT_GPT,
		Partitions: []disk.Partition{
			{
				Size:     1 * MiB,
				Bootable: true,
				Type:     disk.BIOSBootPartitionGUID,
				UUID:     disk.BIOSBootPartitionUUID,
			},
			{
				Size: 200 * MiB,
				Type: disk.EFISystemPartitionGUID,
				UUID: disk.EFISystemPartitionUUID,
				Payload: &disk.Filesystem{
					Type:         "vfat",
					UUID:         disk.EFIFilesystemUUID,
					Mountpoint:   "/boot/efi",
					Label:        "EFI-SYSTEM",
					FSTabOptions: "defaults,uid=0,gid=0,umask=077,shortname=winnt",
					FSTabFreq:    0,
					FSTabPassNo:  2,
				},
			},
			{
				Size: 500 * MiB,
				Type: disk.FilesystemDataGUID,
				UUID: disk.FilesystemDataUUID,
				Payload: &disk.Filesystem{
					Type:         "xfs",
					Mountpoint:   "/boot",
					Label:        "boot",
					FSTabOptions: "defaults",
					FSTabFreq:    0,
					FSTabPassNo:  0,
				},
			},
			{
				Size: 512 * MiB,
				Type: disk.SwapPartitionGUID,
				Payload: &disk.Swap{
					Label:        "swap",
					FSTabOptions: "defaults",
				},
			},
			{
				Type: disk.FilesystemDataGUID,
				UUID: disk.RootPartitionUUID,
				Payload: &disk.Filesystem{
					Type:         "xfs",
					Label:        "root",
					Mountpoint:   "/",
					FSTabOptions: "defaults",
					FSTabFreq:    0,
					FSTabPassNo:  0,
				},
			},
		},
	},

	"plain-noboot": {
		UUID: "D209C89E-EA5E-4FBD-B161-B461CCE297E0",
		Type: disk.PT_GPT,
		Partitions: []disk.Partition{
			{
				Size:     1 * MiB,
				Bootable: true,
				Type:     disk.BIOSBootPartitionGUID,
				UUID:     disk.BIOSBootPartitionUUID,
			},
			{
				Size: 200 * MiB,
				Type: disk.EFISystemPartitionGUID,
				UUID: disk.EFISystemPartitionUUID,
				Payload: &disk.Filesystem{
					Type:         "vfat",
					UUID:         disk.EFIFilesystemUUID,
					Mountpoint:   "/boot/efi",
					Label:        "EFI-SYSTEM",
					FSTabOptions: "defaults,uid=0,gid=0,umask=077,shortname=winnt",
					FSTabFreq:    0,
					FSTabPassNo:  2,
				},
			},
			{
				Type: disk.FilesystemDataGUID,
				UUID: disk.RootPartitionUUID,
				Payload: &disk.Filesystem{
					Type:         "xfs",
					Label:        "root",
					Mountpoint:   "/",
					FSTabOptions: "defaults",
					FSTabFreq:    0,
					FSTabPassNo:  0,
				},
			},
		},
	},

	"luks": {
		UUID: "D209C89E-EA5E-4FBD-B161-B461CCE297E0",
		Type: disk.PT_GPT,
		Partitions: []disk.Partition{
			{
				Size:     1 * MiB,
				Bootable: true,
				Type:     disk.BIOSBootPartitionGUID,
				UUID:     disk.BIOSBootPartitionUUID,
			},
			{
				Size: 200 * MiB,
				Type: disk.EFISystemPartitionGUID,
				UUID: disk.EFISystemPartitionUUID,
				Payload: &disk.Filesystem{
					Type:         "vfat",
					UUID:         disk.EFIFilesystemUUID,
					Mountpoint:   "/boot/efi",
					Label:        "EFI-SYSTEM",
					FSTabOptions: "defaults,uid=0,gid=0,umask=077,shortname=winnt",
					FSTabFreq:    0,
					FSTabPassNo:  2,
				},
			},
			{
				Size: 500 * MiB,
				Type: disk.FilesystemDataGUID,
				UUID: disk.FilesystemDataUUID,
				Payload: &disk.Filesystem{
					Type:         "xfs",
					Mountpoint:   "/boot",
					Label:        "boot",
					FSTabOptions: "defaults",
					FSTabFreq:    0,
					FSTabPassNo:  0,
				},
			},
			{
				Type: disk.FilesystemDataGUID,
				UUID: disk.RootPartitionUUID,
				Payload: &disk.LUKSContainer{
					UUID:  "",
					Label: "crypt_root",
					Payload: &disk.Filesystem{
						Type:         "xfs",
						Label:        "root",
						Mountpoint:   "/",
						FSTabOptions: "defaults",
						FSTabFreq:    0,
						FSTabPassNo:  0,
					},
				},
			},
		},
	},
	"luks+lvm": {
		UUID: "D209C89E-EA5E-4FBD-B161-B461CCE297E0",
		Type: disk.PT_GPT,
		Partitions: []disk.Partition{
			{
				Size:     1 * MiB,
				Bootable: true,
				Type:     disk.BIOSBootPartitionGUID,
				UUID:     disk.BIOSBootPartitionUUID,
			},
			{
				Size: 200 * MiB,
				Type: disk.EFISystemPartitionGUID,
				UUID: disk.EFISystemPartitionUUID,
				Payload: &disk.Filesystem{
					Type:         "vfat",
					UUID:         disk.EFIFilesystemUUID,
					Mountpoint:   "/boot/efi",
					Label:        "EFI-SYSTEM",
					FSTabOptions: "defaults,uid=0,gid=0,umask=077,shortname=winnt",
					FSTabFreq:    0,
					FSTabPassNo:  2,
				},
			},
			{
				Size: 500 * MiB,
				Type: disk.FilesystemDataGUID,
				UUID: disk.FilesystemDataUUID,
				Payload: &disk.Filesystem{
					Type:         "xfs",
					Mountpoint:   "/boot",
					Label:        "boot",
					FSTabOptions: "defaults",
					FSTabFreq:    0,
					FSTabPassNo:  0,
				},
			},
			{
				Type: disk.FilesystemDataGUID,
				UUID: disk.RootPartitionUUID,
				Size: 5 * GiB,
				Payload: &disk.LUKSContainer{
					UUID: "",
					Payload: &disk.LVMVolumeGroup{
						Name:        "",
						Description: "",
						LogicalVolumes: []disk.LVMLogicalVolume{
							{
								Size: 2 * GiB,
								Payload: &disk.Filesystem{
									Type:         "xfs",
									Label:        "root",
									Mountpoint:   "/",
									FSTabOptions: "defaults",
									FSTabFreq:    0,
									FSTabPassNo:  0,
								},
							},
							{
								Size: 2 * GiB,
								Payload: &disk.Filesystem{
									Type:         "xfs",
									Label:        "root",
									Mountpoint:   "/home",
									FSTabOptions: "defaults",
									FSTabFreq:    0,
									FSTabPassNo:  0,
								},
							},
						},
					},
				},
			},
		},
	},
	"btrfs": {
		UUID: "D209C89E-EA5E-4FBD-B161-B461CCE297E0",
		Type: disk.PT_GPT,
		Partitions: []disk.Partition{
			{
				Size:     1 * MiB,
				Bootable: true,
				Type:     disk.BIOSBootPartitionGUID,
				UUID:     disk.BIOSBootPartitionUUID,
			},
			{
				Size: 200 * MiB,
				Type: disk.EFISystemPartitionGUID,
				UUID: disk.EFISystemPartitionUUID,
				Payload: &disk.Filesystem{
					Type:         "vfat",
					UUID:         disk.EFIFilesystemUUID,
					Mountpoint:   "/boot/efi",
					Label:        "EFI-SYSTEM",
					FSTabOptions: "defaults,uid=0,gid=0,umask=077,shortname=winnt",
					FSTabFreq:    0,
					FSTabPassNo:  2,
				},
			},
			{
				Size: 500 * MiB,
				Type: disk.FilesystemDataGUID,
				UUID: disk.FilesystemDataUUID,
				Payload: &disk.Filesystem{
					Type:         "xfs",
					Mountpoint:   "/boot",
					Label:        "boot",
					FSTabOptions: "defaults",
					FSTabFreq:    0,
					FSTabPassNo:  0,
				},
			},
			{
				Type: disk.FilesystemDataGUID,
				UUID: disk.RootPartitionUUID,
				Size: 10 * GiB,
				Payload: &disk.Btrfs{
					UUID:       "",
					Label:      "",
					Mountpoint: "",
					Subvolumes: []disk.BtrfsSubvolume{
						{
							Name:       "root",
							Size:       0,
							Mountpoint: "/",
							GroupID:    0,
						},
						{
							Name:       "var",
							Size:       5 * GiB,
							Mountpoint: "/var",
							GroupID:    0,
						},
					},
				},
			},
		},
	},
}

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
		Type:       disk.PT_GPT,
		Partitions: partitions,
	}
}

// MakeFakeBtrfsPartitionTable is similar to MakeFakePartitionTable but
// creates a btrfs-based partition table.
func MakeFakeBtrfsPartitionTable(mntPoints ...string) *disk.PartitionTable {
	var subvolumes []disk.BtrfsSubvolume
	pt := &disk.PartitionTable{
		Type:       disk.PT_GPT,
		Size:       10 * GiB,
		Partitions: []disk.Partition{},
	}
	size := uint64(0)
	for _, mntPoint := range mntPoints {
		switch mntPoint {
		case "/boot":
			pt.Partitions = append(pt.Partitions, disk.Partition{
				Start: size,
				Size:  1 * GiB,
				Payload: &disk.Filesystem{
					Type:       "ext4",
					Mountpoint: mntPoint,
				},
			})
			size += 1 * GiB
		case "/boot/efi":
			pt.Partitions = append(pt.Partitions, disk.Partition{
				Start: size,
				Size:  100 * MiB,
				Payload: &disk.Filesystem{
					Type:       "vfat",
					Mountpoint: mntPoint,
					UUID:       disk.EFIFilesystemUUID,
				},
			})
			size += 100 * MiB
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
		Size:  9 * GiB,
		Payload: &disk.Btrfs{
			UUID:       disk.RootPartitionUUID,
			Subvolumes: subvolumes,
		},
	})

	size += 9 * GiB
	pt.Size = size

	return pt
}

// MakeFakeLVMPartitionTable is similar to MakeFakePartitionTable but
// creates a lvm-based partition table.
func MakeFakeLVMPartitionTable(mntPoints ...string) *disk.PartitionTable {
	var lvs []disk.LVMLogicalVolume
	pt := &disk.PartitionTable{
		Type:       disk.PT_GPT,
		Size:       10 * GiB,
		Partitions: []disk.Partition{},
	}
	size := uint64(0)
	for _, mntPoint := range mntPoints {
		switch mntPoint {
		case "/boot":
			pt.Partitions = append(pt.Partitions, disk.Partition{
				Start: size,
				Size:  1 * GiB,
				Payload: &disk.Filesystem{
					Type:       "ext4",
					Mountpoint: mntPoint,
				},
			})
			size += 1 * GiB
		case "/boot/efi":
			pt.Partitions = append(pt.Partitions, disk.Partition{
				Start: size,
				Size:  100 * MiB,
				Payload: &disk.Filesystem{
					Type:       "vfat",
					Mountpoint: mntPoint,
					UUID:       disk.EFIFilesystemUUID,
				},
			})
			size += 100 * MiB
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
		Size:  9 * GiB,
		Payload: &disk.LVMVolumeGroup{
			LogicalVolumes: lvs,
		},
	})

	size += 9 * GiB
	pt.Size = size

	return pt
}
