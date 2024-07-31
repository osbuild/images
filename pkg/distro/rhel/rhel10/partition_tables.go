package rhel10

import (
	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/distro/rhel"
)

func defaultBasePartitionTables(t *rhel.ImageType) (disk.PartitionTable, bool) {
	switch t.Arch().Name() {
	case arch.ARCH_X86_64.String():
		return disk.PartitionTable{
			Type: "gpt",
			Partitions: []disk.Partition{
				{
					Size:     1 * common.MebiByte,
					Bootable: true,
					Type:     disk.BIOSBootPartitionGUID,
				},
				{
					Size: 200 * common.MebiByte,
					Type: disk.EFISystemPartitionGUID,
					Payload: &disk.Filesystem{
						Type:         "vfat",
						Mountpoint:   "/boot/efi",
						Label:        "EFI-SYSTEM",
						FSTabOptions: "defaults,uid=0,gid=0,umask=077,shortname=winnt",
						FSTabFreq:    0,
						FSTabPassNo:  2,
					},
				},
				{
					Size: 2 * common.GibiByte,
					Type: disk.FilesystemDataGUID,
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
		}, true
	case arch.ARCH_AARCH64.String():
		return disk.PartitionTable{
			Type: "gpt",
			Partitions: []disk.Partition{
				{
					Size: 200 * common.MebiByte,
					Type: disk.EFISystemPartitionGUID,
					Payload: &disk.Filesystem{
						Type:         "vfat",
						Mountpoint:   "/boot/efi",
						Label:        "EFI-SYSTEM",
						FSTabOptions: "defaults,uid=0,gid=0,umask=077,shortname=winnt",
						FSTabFreq:    0,
						FSTabPassNo:  2,
					},
				},
				{
					Size: 2 * common.GibiByte,
					Type: disk.FilesystemDataGUID,
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
		}, true
	case arch.ARCH_PPC64LE.String():
		return disk.PartitionTable{
			Type: "dos",
			Partitions: []disk.Partition{
				{
					Size:     4 * common.MebiByte,
					Type:     "41",
					Bootable: true,
				},
				{
					Size: 2 * common.GibiByte,
					Payload: &disk.Filesystem{
						Type:         "xfs",
						Mountpoint:   "/",
						FSTabOptions: "defaults",
						FSTabFreq:    0,
						FSTabPassNo:  0,
					},
				},
			},
		}, true

	case arch.ARCH_S390X.String():
		return disk.PartitionTable{
			Type: "dos",
			Partitions: []disk.Partition{
				{
					Size:     2 * common.GibiByte,
					Bootable: true,
					Payload: &disk.Filesystem{
						Type:         "xfs",
						Mountpoint:   "/",
						FSTabOptions: "defaults",
						FSTabFreq:    0,
						FSTabPassNo:  0,
					},
				},
			},
		}, true

	default:
		return disk.PartitionTable{}, false
	}
}
