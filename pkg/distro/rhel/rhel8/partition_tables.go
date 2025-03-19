package rhel8

import (
	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/datasizes"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/distro/defs"
	"github.com/osbuild/images/pkg/distro/rhel"
)

func defaultBasePartitionTables(t *rhel.ImageType) (disk.PartitionTable, bool) {
	partitionTable, err := defs.PartitionTable(t)
	if err != nil {
		// XXX: have a check to differenciate ErrNoEnt and else
		return disk.PartitionTable{}, false
	}
	if partitionTable == nil {
		return disk.PartitionTable{}, false
	}

	return *partitionTable, true
}

func edgeBasePartitionTables(t *rhel.ImageType) (disk.PartitionTable, bool) {
	return defaultBasePartitionTables(t)
}

func ec2PartitionTables(t *rhel.ImageType) (disk.PartitionTable, bool) {
	// x86_64 - without /boot
	// aarch  - <= 8.9 - 512MiB, 8.10 and centos: 1 GiB
	var aarch64BootSize uint64
	switch {
	case common.VersionLessThan(t.Arch().Distro().OsVersion(), "8.10") && t.IsRHEL():
		aarch64BootSize = 512 * datasizes.MebiByte
	default:
		aarch64BootSize = 1 * datasizes.GibiByte
	}

	x86PartitionTable := disk.PartitionTable{
		UUID: "D209C89E-EA5E-4FBD-B161-B461CCE297E0",
		Type: disk.PT_GPT,
		Partitions: []disk.Partition{
			{
				Size:     1 * datasizes.MebiByte,
				Bootable: true,
				Type:     disk.BIOSBootPartitionGUID,
				UUID:     disk.BIOSBootPartitionUUID,
			},
			{
				Size: 200 * datasizes.MebiByte,
				Type: disk.EFISystemPartitionGUID,
				UUID: disk.EFISystemPartitionUUID,
				Payload: &disk.Filesystem{
					Type:         "vfat",
					UUID:         disk.EFIFilesystemUUID,
					Mountpoint:   "/boot/efi",
					FSTabOptions: "defaults,uid=0,gid=0,umask=077,shortname=winnt",
					FSTabFreq:    0,
					FSTabPassNo:  2,
				},
			},
			{
				Size: 2 * datasizes.GibiByte,
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
	}
	// RHEL EC2 x86_64 images prior to 8.9 support only BIOS boot
	if common.VersionLessThan(t.Arch().Distro().OsVersion(), "8.9") && t.IsRHEL() {
		x86PartitionTable = disk.PartitionTable{
			UUID: "D209C89E-EA5E-4FBD-B161-B461CCE297E0",
			Type: disk.PT_GPT,
			Partitions: []disk.Partition{
				{
					Size:     1 * datasizes.MebiByte,
					Bootable: true,
					Type:     disk.BIOSBootPartitionGUID,
					UUID:     disk.BIOSBootPartitionUUID,
				},
				{
					Size: 2 * datasizes.GibiByte,
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
		}
	}

	switch t.Arch().Name() {
	case arch.ARCH_X86_64.String():
		return x86PartitionTable, true

	case arch.ARCH_AARCH64.String():
		return disk.PartitionTable{
			UUID: "D209C89E-EA5E-4FBD-B161-B461CCE297E0",
			Type: disk.PT_GPT,
			Partitions: []disk.Partition{
				{
					Size: 200 * datasizes.MebiByte,
					Type: disk.EFISystemPartitionGUID,
					UUID: disk.EFISystemPartitionUUID,
					Payload: &disk.Filesystem{
						Type:         "vfat",
						UUID:         disk.EFIFilesystemUUID,
						Mountpoint:   "/boot/efi",
						FSTabOptions: "defaults,uid=0,gid=0,umask=077,shortname=winnt",
						FSTabFreq:    0,
						FSTabPassNo:  2,
					},
				},
				{
					Size: aarch64BootSize,
					Type: disk.FilesystemDataGUID,
					UUID: disk.DataPartitionUUID,
					Payload: &disk.Filesystem{
						Type:         "xfs",
						Mountpoint:   "/boot",
						FSTabOptions: "defaults",
						FSTabFreq:    0,
						FSTabPassNo:  0,
					},
				},
				{
					Size: 2 * datasizes.GibiByte,
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
		}, true

	default:
		return disk.PartitionTable{}, false
	}
}
