package fedora

import (
	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/distro"
)

var defaultBasePartitionTables = distro.BasePartitionTableMap{
	arch.ARCH_X86_64.String(): disk.PartitionTable{
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
				Size: 500 * common.MebiByte,
				Type: disk.FilesystemDataGUID,
				Payload: &disk.Filesystem{
					Type:         "ext4",
					Mountpoint:   "/boot",
					Label:        "boot",
					FSTabOptions: "defaults",
					FSTabFreq:    0,
					FSTabPassNo:  0,
				},
			},
			{
				Size: 2 * common.GibiByte,
				Type: disk.FilesystemDataGUID,
				Payload: &disk.Filesystem{
					Type:         "ext4",
					Label:        "root",
					Mountpoint:   "/",
					FSTabOptions: "defaults",
					FSTabFreq:    0,
					FSTabPassNo:  0,
				},
			},
		},
	},
	arch.ARCH_AARCH64.String(): disk.PartitionTable{
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
				Size: 500 * common.MebiByte,
				Type: disk.FilesystemDataGUID,
				Payload: &disk.Filesystem{
					Type:         "ext4",
					Mountpoint:   "/boot",
					Label:        "boot",
					FSTabOptions: "defaults",
					FSTabFreq:    0,
					FSTabPassNo:  0,
				},
			},
			{
				Size: 2 * common.GibiByte,
				Type: disk.FilesystemDataGUID,
				Payload: &disk.Filesystem{
					Type:         "ext4",
					Label:        "root",
					Mountpoint:   "/",
					FSTabOptions: "defaults",
					FSTabFreq:    0,
					FSTabPassNo:  0,
				},
			},
		},
	},
	arch.ARCH_PPC64LE.String(): disk.PartitionTable{
		Type: "dos",
		Partitions: []disk.Partition{
			{
				Size:     4 * common.MebiByte,
				Type:     "41",
				Bootable: true,
			},
			{
				Size: 500 * common.MebiByte,
				Payload: &disk.Filesystem{
					Type:         "ext4",
					Mountpoint:   "/boot",
					Label:        "boot",
					FSTabOptions: "defaults",
					FSTabFreq:    0,
					FSTabPassNo:  0,
				},
			},
			{
				Size: 2 * common.GibiByte,
				Payload: &disk.Filesystem{
					Type:         "ext4",
					Mountpoint:   "/",
					FSTabOptions: "defaults",
					FSTabFreq:    0,
					FSTabPassNo:  0,
				},
			},
		},
	},

	arch.ARCH_S390X.String(): disk.PartitionTable{
		Type: "dos",
		Partitions: []disk.Partition{
			{
				Size: 500 * common.MebiByte,
				Payload: &disk.Filesystem{
					Type:         "ext4",
					Mountpoint:   "/boot",
					Label:        "boot",
					FSTabOptions: "defaults",
					FSTabFreq:    0,
					FSTabPassNo:  0,
				},
			},
			{
				Size:     2 * common.GibiByte,
				Bootable: true,
				Payload: &disk.Filesystem{
					Type:         "ext4",
					Mountpoint:   "/",
					FSTabOptions: "defaults",
					FSTabFreq:    0,
					FSTabPassNo:  0,
				},
			},
		},
	},
}

var minimalrawPartitionTables = distro.BasePartitionTableMap{
	arch.ARCH_X86_64.String(): disk.PartitionTable{
		Type:        "gpt",
		StartOffset: 8 * common.MebiByte,
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
				Size: 1 * common.GibiByte,
				Type: disk.XBootLDRPartitionGUID,
				Payload: &disk.Filesystem{
					Type:         "ext4",
					Mountpoint:   "/boot",
					Label:        "boot",
					FSTabOptions: "defaults",
					FSTabFreq:    0,
					FSTabPassNo:  0,
				},
			},
			{
				Size: 2 * common.GibiByte,
				Type: disk.FilesystemDataGUID,
				Payload: &disk.Filesystem{
					Type:         "ext4",
					Label:        "root",
					Mountpoint:   "/",
					FSTabOptions: "defaults",
					FSTabFreq:    0,
					FSTabPassNo:  0,
				},
			},
		},
	},
	arch.ARCH_AARCH64.String(): disk.PartitionTable{
		Type:        "dos",
		StartOffset: 8 * common.MebiByte,
		Partitions: []disk.Partition{
			{
				Size:     200 * common.MebiByte,
				Type:     "06",
				Bootable: true,
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
				Size: 1 * common.GibiByte,
				Type: "83",
				Payload: &disk.Filesystem{
					Type:         "ext4",
					Mountpoint:   "/boot",
					Label:        "boot",
					FSTabOptions: "defaults",
					FSTabFreq:    0,
					FSTabPassNo:  0,
				},
			},
			{
				Size: 2 * common.GibiByte,
				Type: "83",
				Payload: &disk.Filesystem{
					Type:         "ext4",
					Label:        "root",
					Mountpoint:   "/",
					FSTabOptions: "defaults",
					FSTabFreq:    0,
					FSTabPassNo:  0,
				},
			},
		},
	},
}

var iotBasePartitionTables = distro.BasePartitionTableMap{
	arch.ARCH_X86_64.String(): disk.PartitionTable{
		Type:        "gpt",
		StartOffset: 8 * common.MebiByte,
		Partitions: []disk.Partition{
			{
				Size: 501 * common.MebiByte,
				Type: disk.EFISystemPartitionGUID,
				Payload: &disk.Filesystem{
					Type:         "vfat",
					Mountpoint:   "/boot/efi",
					Label:        "EFI-SYSTEM",
					FSTabOptions: "umask=0077,shortname=winnt",
					FSTabFreq:    0,
					FSTabPassNo:  2,
				},
			},
			{
				Size: 1 * common.GibiByte,
				Type: disk.FilesystemDataGUID,
				Payload: &disk.Filesystem{
					Type:         "ext4",
					Mountpoint:   "/boot",
					Label:        "boot",
					FSTabOptions: "defaults",
					FSTabFreq:    1,
					FSTabPassNo:  2,
				},
			},
			{
				Size: 2569 * common.MebiByte,
				Type: disk.FilesystemDataGUID,
				Payload: &disk.Filesystem{
					Type:         "ext4",
					Label:        "root",
					Mountpoint:   "/",
					FSTabOptions: "defaults",
					FSTabFreq:    1,
					FSTabPassNo:  1,
				},
			},
		},
	},
	arch.ARCH_AARCH64.String(): disk.PartitionTable{
		Type:        "dos",
		StartOffset: 8 * common.MebiByte,
		Partitions: []disk.Partition{
			{
				Size:     501 * common.MebiByte,
				Type:     "06",
				Bootable: true,
				Payload: &disk.Filesystem{
					Type:         "vfat",
					Mountpoint:   "/boot/efi",
					Label:        "EFI-SYSTEM",
					FSTabOptions: "umask=0077,shortname=winnt",
					FSTabFreq:    0,
					FSTabPassNo:  2,
				},
			},
			{
				Size: 1 * common.GibiByte,
				Type: "83",
				Payload: &disk.Filesystem{
					Type:         "ext4",
					Mountpoint:   "/boot",
					Label:        "boot",
					FSTabOptions: "defaults",
					FSTabFreq:    1,
					FSTabPassNo:  2,
				},
			},
			{
				Size: 2569 * common.MebiByte,
				Type: "83",
				Payload: &disk.Filesystem{
					Type:         "ext4",
					Label:        "root",
					Mountpoint:   "/",
					FSTabOptions: "defaults",
					FSTabFreq:    1,
					FSTabPassNo:  1,
				},
			},
		},
	},
}

var iotSimplifiedInstallerPartitionTables = distro.BasePartitionTableMap{
	arch.ARCH_X86_64.String(): disk.PartitionTable{
		Type: "gpt",
		Partitions: []disk.Partition{
			{
				Size: 501 * common.MebiByte,
				Type: disk.EFISystemPartitionGUID,
				Payload: &disk.Filesystem{
					Type:         "vfat",
					Mountpoint:   "/boot/efi",
					Label:        "EFI-SYSTEM",
					FSTabOptions: "umask=0077,shortname=winnt",
					FSTabFreq:    0,
					FSTabPassNo:  2,
				},
			},
			{
				Size: 1 * common.GibiByte,
				Type: disk.XBootLDRPartitionGUID,
				Payload: &disk.Filesystem{
					Type:         "ext4",
					Mountpoint:   "/boot",
					Label:        "boot",
					FSTabOptions: "defaults",
					FSTabFreq:    1,
					FSTabPassNo:  1,
				},
			},
			{
				Type: disk.FilesystemDataGUID,
				Payload: &disk.LUKSContainer{
					Label:      "crypt_root",
					Cipher:     "cipher_null",
					Passphrase: "osbuild",
					PBKDF: disk.Argon2id{
						Memory:      32,
						Iterations:  4,
						Parallelism: 1,
					},
					Clevis: &disk.ClevisBind{
						Pin:              "null",
						Policy:           "{}",
						RemovePassphrase: true,
					},
					Payload: &disk.LVMVolumeGroup{
						Name:        "rootvg",
						Description: "built with lvm2 and osbuild",
						LogicalVolumes: []disk.LVMLogicalVolume{
							{
								Size: 8 * common.GibiByte,
								Name: "rootlv",
								Payload: &disk.Filesystem{
									Type:         "ext4",
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
			},
		},
	},
	arch.ARCH_AARCH64.String(): disk.PartitionTable{
		Type: "gpt",
		Partitions: []disk.Partition{
			{
				Size: 501 * common.MebiByte,
				Type: disk.EFISystemPartitionGUID,
				Payload: &disk.Filesystem{
					Type:         "vfat",
					Mountpoint:   "/boot/efi",
					Label:        "EFI-SYSTEM",
					FSTabOptions: "umask=0077,shortname=winnt",
					FSTabFreq:    0,
					FSTabPassNo:  2,
				},
			},
			{
				Size: 1 * common.GibiByte,
				Type: disk.XBootLDRPartitionGUID,
				Payload: &disk.Filesystem{
					Type:         "ext4",
					Mountpoint:   "/boot",
					Label:        "boot",
					FSTabOptions: "defaults",
					FSTabFreq:    1,
					FSTabPassNo:  1,
				},
			},
			{
				Type: disk.FilesystemDataGUID,
				Payload: &disk.LUKSContainer{
					Label:      "crypt_root",
					Cipher:     "cipher_null",
					Passphrase: "osbuild",
					PBKDF: disk.Argon2id{
						Memory:      32,
						Iterations:  4,
						Parallelism: 1,
					},
					Clevis: &disk.ClevisBind{
						Pin:              "null",
						Policy:           "{}",
						RemovePassphrase: true,
					},
					Payload: &disk.LVMVolumeGroup{
						Name:        "rootvg",
						Description: "built with lvm2 and osbuild",
						LogicalVolumes: []disk.LVMLogicalVolume{
							{
								Size: 8 * common.GibiByte,
								Name: "rootlv",
								Payload: &disk.Filesystem{
									Type:         "ext4",
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
			},
		},
	},
}
