package disk_test

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/internal/testdisk"
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/datasizes"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/platform"
)

func TestPartitionTable_GetMountpointSize(t *testing.T) {
	pt := testdisk.MakeFakePartitionTable("/", "/app")

	size, err := pt.GetMountpointSize("/")
	assert.NoError(t, err)
	assert.Equal(t, testdisk.FakePartitionSize, size)

	size, err = pt.GetMountpointSize("/app")
	assert.NoError(t, err)
	assert.Equal(t, testdisk.FakePartitionSize, size)

	// non-existing
	_, err = pt.GetMountpointSize("/custom")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot find mountpoint /custom")
}

func TestPartitionTable_GenerateUUIDs(t *testing.T) {
	pt := disk.PartitionTable{
		Type: "gpt",
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
				Payload: &disk.Filesystem{
					// create mixed xfs root filesystem and a btrfs /var partition
					Type:         "xfs",
					Label:        "root",
					Mountpoint:   "/",
					FSTabOptions: "defaults",
					FSTabFreq:    0,
					FSTabPassNo:  0,
				},
			},
			{
				Size: 10 * datasizes.GibiByte,
				Payload: &disk.Btrfs{
					Subvolumes: []disk.BtrfsSubvolume{
						{
							Mountpoint: "/var",
						},
					},
				},
			},
		},
	}

	// Static seed for testing
	/* #nosec G404 */
	rnd := rand.New(rand.NewSource(0))

	pt.GenerateUUIDs(rnd)

	// Check that GenUUID doesn't change already defined UUIDs
	assert.Equal(t, disk.BIOSBootPartitionUUID, pt.Partitions[0].UUID)

	// Check that GenUUID generates fresh UUIDs if not defined prior the call
	assert.Equal(t, "a178892e-e285-4ce1-9114-55780875d64e", pt.Partitions[1].UUID)
	assert.Equal(t, "6e4ff95f-f662-45ee-a82a-bdf44a2d0b75", pt.Partitions[1].Payload.(*disk.Filesystem).UUID)

	// Check that GenUUID generates the same UUID for BTRFS and its subvolumes
	assert.Equal(t, "fb180daf-48a7-4ee0-b10d-394651850fd4", pt.Partitions[2].Payload.(*disk.Btrfs).UUID)
	assert.Equal(t, "fb180daf-48a7-4ee0-b10d-394651850fd4", pt.Partitions[2].Payload.(*disk.Btrfs).Subvolumes[0].UUID)
}

func TestPartitionTable_GenerateUUIDs_VFAT(t *testing.T) {
	pt := disk.PartitionTable{
		Type: "dos",
		Partitions: []disk.Partition{
			{
				Size: 2 * datasizes.GibiByte,
				Type: disk.FilesystemDataGUID,
				Payload: &disk.Filesystem{
					Type:       "vfat",
					Mountpoint: "/boot/efi",
				},
			},
		},
	}

	// Static seed for testing
	/* #nosec G404 */
	rnd := rand.New(rand.NewSource(0))

	pt.GenerateUUIDs(rnd)

	assert.Equal(t, "6e4ff95f", pt.Partitions[0].Payload.(*disk.Filesystem).UUID)
}

func TestNewCustomPartitionTable(t *testing.T) {
	// Static seed for testing
	/* #nosec G404 */
	rnd := rand.New(rand.NewSource(0))

	type testCase struct {
		customizations *blueprint.PartitioningCustomization
		options        *disk.CustomPartitionTableOptions
		expected       *disk.PartitionTable
	}

	testCases := map[string]testCase{
		"dos-hybrid": {
			customizations: nil,
			options: &disk.CustomPartitionTableOptions{
				DefaultFSType:      disk.FS_XFS,
				BootMode:           platform.BOOT_HYBRID,
				PartitionTableType: disk.PT_DOS,
			},
			expected: &disk.PartitionTable{
				Type: "dos",
				Size: 202 * datasizes.MiB,
				Partitions: []disk.Partition{
					{
						Start:    1 * datasizes.MiB, // header
						Bootable: true,
						Size:     1 * datasizes.MiB,
						Type:     disk.BIOSBootPartitionGUID,
						UUID:     disk.BIOSBootPartitionUUID,
					},
					{
						Start: 2 * datasizes.MiB,
						Size:  200 * datasizes.MiB,
						Type:  disk.EFISystemPartitionGUID,
						UUID:  disk.EFISystemPartitionUUID,
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
						Start:    202 * datasizes.MiB,
						Size:     0,
						Type:     disk.FilesystemDataGUID,
						Bootable: false,
						Payload: &disk.Filesystem{
							Type:         "xfs",
							Label:        "root",
							Mountpoint:   "/",
							FSTabOptions: "defaults",
						},
					},
				},
			},
		},
		"plain": {
			customizations: &blueprint.PartitioningCustomization{
				Plain: &blueprint.PlainFilesystemCustomization{
					Filesystems: []blueprint.FilesystemCustomization{
						{
							Mountpoint: "/data",
							MinSize:    20 * datasizes.MiB,
							Label:      "data",
							Type:       "ext4",
						},
					},
				},
			},
			options: &disk.CustomPartitionTableOptions{
				DefaultFSType:      disk.FS_XFS,
				BootMode:           platform.BOOT_HYBRID,
				PartitionTableType: disk.PT_DOS,
			},
			expected: &disk.PartitionTable{
				Type: "dos",
				Size: 222 * datasizes.MiB,
				Partitions: []disk.Partition{
					{
						Start:    1 * datasizes.MiB, // header
						Size:     1 * datasizes.MiB,
						Bootable: true,
						Type:     disk.BIOSBootPartitionGUID,
						UUID:     disk.BIOSBootPartitionUUID,
					},
					{
						Start: 2 * datasizes.MiB,
						Size:  200 * datasizes.MiB,
						Type:  disk.EFISystemPartitionGUID,
						UUID:  disk.EFISystemPartitionUUID,
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
						Start:    202 * datasizes.MiB,
						Size:     20 * datasizes.MiB,
						Type:     disk.FilesystemDataGUID,
						Bootable: false,
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "data",
							Mountpoint:   "/data",
							FSTabOptions: "defaults",
							FSTabFreq:    0,
							FSTabPassNo:  0,
						},
					},
					{
						Start:    222 * datasizes.MiB,
						Size:     0,
						Type:     disk.FilesystemDataGUID,
						Bootable: false,
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
		"plain+": {
			customizations: &blueprint.PartitioningCustomization{
				Plain: &blueprint.PlainFilesystemCustomization{
					Filesystems: []blueprint.FilesystemCustomization{
						{
							Mountpoint: "/",
							MinSize:    50 * datasizes.MiB,
							Label:      "root",
							Type:       "xfs",
						},
						{
							Mountpoint: "/home",
							MinSize:    20 * datasizes.MiB,
							Label:      "home",
							Type:       "ext4",
						},
					},
				},
			},
			options: &disk.CustomPartitionTableOptions{
				DefaultFSType:      disk.FS_EXT4,
				BootMode:           platform.BOOT_HYBRID,
				PartitionTableType: disk.PT_GPT,
			},
			expected: &disk.PartitionTable{
				Type: "gpt",
				Size: 273 * datasizes.MiB,
				Partitions: []disk.Partition{
					{
						Start:    1 * datasizes.MiB, // header
						Size:     1 * datasizes.MiB,
						Bootable: true,
						Type:     disk.BIOSBootPartitionGUID,
						UUID:     disk.BIOSBootPartitionUUID,
					},
					{
						Start: 2 * datasizes.MiB,
						Size:  200 * datasizes.MiB,
						Type:  disk.EFISystemPartitionGUID,
						UUID:  disk.EFISystemPartitionUUID,
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
					// root is aligned to the end but not reindexed
					{
						Start:    222 * datasizes.MiB,
						Size:     51*datasizes.MiB - (disk.DefaultSectorSize + (128 * 128)), // grows by 1 grain size (1 MiB) minus the unaligned size of the header to fit the gpt footer
						Type:     disk.FilesystemDataGUID,
						Bootable: false,
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
						Start:    202 * datasizes.MiB,
						Size:     20 * datasizes.MiB,
						Type:     disk.FilesystemDataGUID,
						Bootable: false,
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "home",
							Mountpoint:   "/home",
							FSTabOptions: "defaults",
							FSTabFreq:    0,
							FSTabPassNo:  0,
						},
					},
				},
			},
		},
		"lvm": {
			customizations: &blueprint.PartitioningCustomization{
				LVM: &blueprint.LVMCustomization{
					VolumeGroups: []blueprint.VGCustomization{
						{
							Name:    "testvg",
							MinSize: 100 * datasizes.MiB,
							LogicalVolumes: []blueprint.LVCustomization{
								{
									Name: "varloglv",
									FilesystemCustomization: blueprint.FilesystemCustomization{
										Mountpoint: "/var/log",
										MinSize:    10 * datasizes.MiB,
										Label:      "var-log",
										Type:       "xfs",
									},
								},
								{
									Name: "rootlv",
									FilesystemCustomization: blueprint.FilesystemCustomization{
										Mountpoint: "/",
										MinSize:    50 * datasizes.MiB,
										Label:      "root",
										Type:       "xfs",
									},
								},
								{
									Name: "datalv", // untyped logical volume
									FilesystemCustomization: blueprint.FilesystemCustomization{
										Mountpoint: "/data",
										MinSize:    100 * datasizes.MiB,
										Label:      "data",
									},
								},
							},
						},
					},
				},
			},
			options: &disk.CustomPartitionTableOptions{
				DefaultFSType: disk.FS_EXT4,
				BootMode:      platform.BOOT_HYBRID,
			},
			expected: &disk.PartitionTable{
				Type: "gpt", // default when unspecified
				Size: 879 * datasizes.MiB,
				Partitions: []disk.Partition{
					{
						Start:    1 * datasizes.MiB, // header
						Size:     1 * datasizes.MiB,
						Bootable: true,
						Type:     disk.BIOSBootPartitionGUID,
						UUID:     disk.BIOSBootPartitionUUID,
					},
					{
						Start: 2 * datasizes.MiB,
						Size:  200 * datasizes.MiB,
						Type:  disk.EFISystemPartitionGUID,
						UUID:  disk.EFISystemPartitionUUID,
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
						Start:    202 * datasizes.MiB,
						Size:     512 * datasizes.MiB,
						Type:     disk.XBootLDRPartitionGUID,
						Bootable: false,
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "boot",
							Mountpoint:   "/boot",
							FSTabOptions: "defaults",
							FSTabFreq:    0,
							FSTabPassNo:  0,
						},
					},
					{
						Start:    714 * datasizes.MiB,
						Size:     165*datasizes.MiB - (disk.DefaultSectorSize + (128 * 128)), // includes 4 MiB LVM header (after rounding to the next extent) - gpt footer the same size as the header (unaligned)
						Type:     disk.LVMPartitionGUID,
						Bootable: false,
						Payload: &disk.LVMVolumeGroup{
							Name:        "testvg",
							Description: "created via lvm2 and osbuild",
							LogicalVolumes: []disk.LVMLogicalVolume{
								{
									Name: "varloglv",
									Size: 10 * datasizes.MiB,
									Payload: &disk.Filesystem{
										Label:        "var-log",
										Type:         "xfs",
										Mountpoint:   "/var/log",
										FSTabOptions: "defaults",
									},
								},
								{
									Name: "rootlv",
									Size: 50 * datasizes.MiB,
									Payload: &disk.Filesystem{
										Label:        "root",
										Type:         "xfs",
										Mountpoint:   "/",
										FSTabOptions: "defaults",
									},
								},
								{
									Name: "datalv",
									Size: 100 * datasizes.MiB,
									Payload: &disk.Filesystem{
										Label:        "data",
										Type:         "ext4", // the defaultType
										Mountpoint:   "/data",
										FSTabOptions: "defaults",
									},
								},
							},
						},
					},
				},
			},
		},
		"btrfs": {
			customizations: &blueprint.PartitioningCustomization{
				Btrfs: &blueprint.BtrfsCustomization{
					Volumes: []blueprint.BtrfsVolumeCustomization{
						{
							MinSize: 230 * datasizes.MiB,
							Subvolumes: []blueprint.BtrfsSubvolumeCustomization{
								{
									Name:       "subvol/root",
									Mountpoint: "/",
								},
								{
									Name:       "subvol/home",
									Mountpoint: "/home",
								},
								{
									Name:       "subvol/varlog",
									Mountpoint: "/var/log",
								},
							},
						},
					},
				},
			},
			options: &disk.CustomPartitionTableOptions{
				DefaultFSType:      disk.FS_EXT4,
				BootMode:           platform.BOOT_HYBRID,
				PartitionTableType: disk.PT_GPT,
			},
			expected: &disk.PartitionTable{
				Type: "gpt",
				Size: 945 * datasizes.MiB,
				Partitions: []disk.Partition{
					{
						Start:    1 * datasizes.MiB, // header
						Size:     1 * datasizes.MiB,
						Bootable: true,
						Type:     disk.BIOSBootPartitionGUID,
						UUID:     disk.BIOSBootPartitionUUID,
					},
					{
						Start: 2 * datasizes.MiB, // header
						Size:  200 * datasizes.MiB,
						Type:  disk.EFISystemPartitionGUID,
						UUID:  disk.EFISystemPartitionUUID,
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
						Start:    202 * datasizes.MiB,
						Size:     512 * datasizes.MiB,
						Type:     disk.XBootLDRPartitionGUID,
						Bootable: false,
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "boot",
							Mountpoint:   "/boot",
							FSTabOptions: "defaults",
							FSTabFreq:    0,
							FSTabPassNo:  0,
						},
					},
					{
						Start:    714 * datasizes.MiB,
						Size:     231*datasizes.MiB - (disk.DefaultSectorSize + (128 * 128)), // grows by 1 grain size (1 MiB) minus the unaligned size of the header to fit the gpt footer
						Type:     disk.FilesystemDataGUID,
						Bootable: false,
						Payload: &disk.Btrfs{
							Subvolumes: []disk.BtrfsSubvolume{
								{
									Name:       "subvol/root",
									Mountpoint: "/",
								},
								{
									Name:       "subvol/home",
									Mountpoint: "/home",
								},
								{
									Name:       "subvol/varlog",
									Mountpoint: "/var/log",
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
			assert := assert.New(t)

			pt, err := disk.NewCustomPartitionTable(tc.customizations, tc.options, rnd)

			assert.NoError(err)
			assert.Equal(tc.expected, pt)
		})
	}

}
