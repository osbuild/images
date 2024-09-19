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
				UUID: "0194fdc2-fa2f-4cc0-81d3-ff12045b73c8",
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
							UUID:         "6e4ff95f-f662-45ee-a82a-bdf44a2d0b75",
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
				UUID: "0194fdc2-fa2f-4cc0-81d3-ff12045b73c8",
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
						UUID:     "", // partitions on dos PTs don't have UUIDs
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "data",
							Mountpoint:   "/data",
							UUID:         "6e4ff95f-f662-45ee-a82a-bdf44a2d0b75",
							FSTabOptions: "defaults",
							FSTabFreq:    0,
							FSTabPassNo:  0,
						},
					},
					{
						Start:    222 * datasizes.MiB,
						Size:     0,
						Type:     disk.FilesystemDataGUID,
						UUID:     "", // partitions on dos PTs don't have UUIDs
						Bootable: false,
						Payload: &disk.Filesystem{
							Type:         "xfs",
							Label:        "root",
							Mountpoint:   "/",
							UUID:         "fb180daf-48a7-4ee0-b10d-394651850fd4",
							FSTabOptions: "defaults",
							FSTabFreq:    0,
							FSTabPassNo:  0,
						},
					},
				},
			},
		},
		"plain-legacy": {
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
				BootMode:           platform.BOOT_LEGACY,
				PartitionTableType: disk.PT_DOS,
			},
			expected: &disk.PartitionTable{
				Type: "dos",
				Size: 22 * datasizes.MiB,
				UUID: "0194fdc2-fa2f-4cc0-81d3-ff12045b73c8",
				Partitions: []disk.Partition{
					{
						Start:    1 * datasizes.MiB, // header
						Size:     1 * datasizes.MiB,
						Bootable: true,
						Type:     disk.BIOSBootPartitionGUID,
						UUID:     disk.BIOSBootPartitionUUID,
					},
					{
						Start:    2 * datasizes.MiB,
						Size:     20 * datasizes.MiB,
						Type:     disk.FilesystemDataGUID,
						Bootable: false,
						UUID:     "", // partitions on dos PTs don't have UUIDs
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "data",
							Mountpoint:   "/data",
							UUID:         "6e4ff95f-f662-45ee-a82a-bdf44a2d0b75",
							FSTabOptions: "defaults",
							FSTabFreq:    0,
							FSTabPassNo:  0,
						},
					},
					{
						Start:    22 * datasizes.MiB,
						Size:     0,
						Type:     disk.FilesystemDataGUID,
						UUID:     "", // partitions on dos PTs don't have UUIDs
						Bootable: false,
						Payload: &disk.Filesystem{
							Type:         "xfs",
							Label:        "root",
							Mountpoint:   "/",
							UUID:         "fb180daf-48a7-4ee0-b10d-394651850fd4",
							FSTabOptions: "defaults",
							FSTabFreq:    0,
							FSTabPassNo:  0,
						},
					},
				},
			},
		},
		"plain-uefi": {
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
				BootMode:           platform.BOOT_UEFI,
				PartitionTableType: disk.PT_DOS,
			},
			expected: &disk.PartitionTable{
				Type: "dos",
				Size: 221 * datasizes.MiB,
				UUID: "0194fdc2-fa2f-4cc0-81d3-ff12045b73c8",
				Partitions: []disk.Partition{
					{
						Start: 1 * datasizes.MiB,
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
						Start:    201 * datasizes.MiB,
						Size:     20 * datasizes.MiB,
						Type:     disk.FilesystemDataGUID,
						Bootable: false,
						UUID:     "", // partitions on dos PTs don't have UUIDs
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "data",
							Mountpoint:   "/data",
							UUID:         "6e4ff95f-f662-45ee-a82a-bdf44a2d0b75",
							FSTabOptions: "defaults",
							FSTabFreq:    0,
							FSTabPassNo:  0,
						},
					},
					{
						Start:    221 * datasizes.MiB,
						Size:     0,
						Type:     disk.FilesystemDataGUID,
						UUID:     "", // partitions on dos PTs don't have UUIDs
						Bootable: false,
						Payload: &disk.Filesystem{
							Type:         "xfs",
							Label:        "root",
							Mountpoint:   "/",
							UUID:         "fb180daf-48a7-4ee0-b10d-394651850fd4",
							FSTabOptions: "defaults",
							FSTabFreq:    0,
							FSTabPassNo:  0,
						},
					},
				},
			},
		},
		"plain-reqsizes": {
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
				RequiredMinSizes:   map[string]uint64{"/": 1 * datasizes.GiB, "/usr": 2 * datasizes.GiB}, // the default for our distro definitions
				PartitionTableType: disk.PT_DOS,
			},
			expected: &disk.PartitionTable{
				Type: "dos",
				Size: 222*datasizes.MiB + 3*datasizes.GiB,
				UUID: "0194fdc2-fa2f-4cc0-81d3-ff12045b73c8",
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
						UUID:     "", // partitions on dos PTs don't have UUIDs
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "data",
							Mountpoint:   "/data",
							UUID:         "6e4ff95f-f662-45ee-a82a-bdf44a2d0b75",
							FSTabOptions: "defaults",
							FSTabFreq:    0,
							FSTabPassNo:  0,
						},
					},
					{
						Start:    222 * datasizes.MiB,
						Size:     3 * datasizes.GiB,
						Type:     disk.FilesystemDataGUID,
						UUID:     "", // partitions on dos PTs don't have UUIDs
						Bootable: false,
						Payload: &disk.Filesystem{
							Type:         "xfs",
							Label:        "root",
							Mountpoint:   "/",
							UUID:         "fb180daf-48a7-4ee0-b10d-394651850fd4",
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
				RequiredMinSizes:   map[string]uint64{"/": 3 * datasizes.GiB},
			},
			expected: &disk.PartitionTable{
				Type: "gpt",
				Size: 222*datasizes.MiB + 3*datasizes.GiB + datasizes.MiB, // start + size of last partition + footer

				UUID: "0194fdc2-fa2f-4cc0-81d3-ff12045b73c8",
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
						Size:     3*datasizes.GiB + datasizes.MiB - (disk.DefaultSectorSize + (128 * 128)), // grows by 1 grain size (1 MiB) minus the unaligned size of the header to fit the gpt footer
						Type:     disk.FilesystemDataGUID,
						UUID:     "a178892e-e285-4ce1-9114-55780875d64e",
						Bootable: false,
						Payload: &disk.Filesystem{
							Type:         "xfs",
							Label:        "root",
							Mountpoint:   "/",
							FSTabOptions: "defaults",
							UUID:         "6e4ff95f-f662-45ee-a82a-bdf44a2d0b75",
							FSTabFreq:    0,
							FSTabPassNo:  0,
						},
					},
					{
						Start:    202 * datasizes.MiB,
						Size:     20 * datasizes.MiB,
						Type:     disk.FilesystemDataGUID,
						UUID:     "e2d3d0d0-de6b-48f9-b44c-e85ff044c6b1",
						Bootable: false,
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "home",
							Mountpoint:   "/home",
							FSTabOptions: "defaults",
							UUID:         "fb180daf-48a7-4ee0-b10d-394651850fd4",
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
								{ // unnamed + untyped logical volume
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
				UUID: "0194fdc2-fa2f-4cc0-81d3-ff12045b73c8",
				Size: 714*datasizes.MiB + 168*datasizes.MiB + datasizes.MiB, // start + size of last partition (VG) + footer
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
						UUID:     "f83b8e88-3bbf-457a-ab99-c5b252c7429c",
						Bootable: false,
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "boot",
							Mountpoint:   "/boot",
							FSTabOptions: "defaults",
							UUID:         "6e4ff95f-f662-45ee-a82a-bdf44a2d0b75",
							FSTabFreq:    0,
							FSTabPassNo:  0,
						},
					},
					{
						Start:    714 * datasizes.MiB,
						Size:     168*datasizes.MiB + datasizes.MiB - (disk.DefaultSectorSize + (128 * 128)), // the sum of the LVs (rounded to the next 4 MiB extent) grows by 1 grain size (1 MiB) minus the unaligned size of the header to fit the gpt footer
						Type:     disk.LVMPartitionGUID,
						UUID:     "32f3a8ae-b79e-4856-b659-c18f0dcecc77",
						Bootable: false,
						Payload: &disk.LVMVolumeGroup{
							Name:        "testvg",
							Description: "created via lvm2 and osbuild",
							LogicalVolumes: []disk.LVMLogicalVolume{
								{
									Name: "varloglv",
									Size: 12 * datasizes.MiB, // rounded up to next extent (4 MiB)
									Payload: &disk.Filesystem{
										Label:        "var-log",
										Type:         "xfs",
										Mountpoint:   "/var/log",
										FSTabOptions: "defaults",
										UUID:         "fb180daf-48a7-4ee0-b10d-394651850fd4",
									},
								},
								{
									Name: "rootlv",
									Size: 52 * datasizes.MiB, // rounded up to the next extent (4 MiB)
									Payload: &disk.Filesystem{
										Label:        "root",
										Type:         "xfs",
										Mountpoint:   "/",
										FSTabOptions: "defaults",
										UUID:         "a178892e-e285-4ce1-9114-55780875d64e",
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
										UUID:         "e2d3d0d0-de6b-48f9-b44c-e85ff044c6b1",
									},
								},
							},
						},
					},
				},
			},
		},
		"lvm-multivg": {
			// two volume groups, both unnamed, and no root lv defined
			// NOTE: this is currently not supported by customizations but the
			// PR creation function can handle it
			customizations: &blueprint.PartitioningCustomization{
				LVM: &blueprint.LVMCustomization{
					VolumeGroups: []blueprint.VGCustomization{
						{
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
							},
						},
						{
							LogicalVolumes: []blueprint.LVCustomization{
								{ // unnamed + untyped logical volume
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
				DefaultFSType:    disk.FS_EXT4,
				BootMode:         platform.BOOT_HYBRID,
				RequiredMinSizes: map[string]uint64{"/": 3 * datasizes.GiB},
			},
			expected: &disk.PartitionTable{
				Type: "gpt", // default when unspecified
				UUID: "0194fdc2-fa2f-4cc0-81d3-ff12045b73c8",
				Size: 818*datasizes.MiB + 16*datasizes.MiB + 3*datasizes.GiB + datasizes.MiB, // start + size of last partition (VG) + footer
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
						UUID:     "f83b8e88-3bbf-457a-ab99-c5b252c7429c",
						Bootable: false,
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "boot",
							Mountpoint:   "/boot",
							FSTabOptions: "defaults",
							UUID:         "6e4ff95f-f662-45ee-a82a-bdf44a2d0b75",
							FSTabFreq:    0,
							FSTabPassNo:  0,
						},
					},
					{
						Start:    818 * datasizes.MiB,                                                                         // the root vg is moved to the end of the partition table by relayout()
						Size:     3*datasizes.GiB + 16*datasizes.MiB + datasizes.MiB - (disk.DefaultSectorSize + (128 * 128)), // the sum of the LVs (rounded to the next 4 MiB extent) grows by 1 grain size (1 MiB) minus the unaligned size of the header to fit the gpt footer
						Type:     disk.LVMPartitionGUID,
						UUID:     "32f3a8ae-b79e-4856-b659-c18f0dcecc77",
						Bootable: false,
						Payload: &disk.LVMVolumeGroup{
							Name:        "vg00",
							Description: "created via lvm2 and osbuild",
							LogicalVolumes: []disk.LVMLogicalVolume{
								{
									Name: "varloglv",
									Size: 12 * datasizes.MiB, // rounded up to next extent (4 MiB)
									Payload: &disk.Filesystem{
										Label:        "var-log",
										Type:         "xfs",
										Mountpoint:   "/var/log",
										FSTabOptions: "defaults",
										UUID:         "fb180daf-48a7-4ee0-b10d-394651850fd4",
									},
								},
								{
									Name: "rootlv",
									Size: 3 * datasizes.GiB,
									Payload: &disk.Filesystem{
										Label:        "root",
										Type:         "ext4", // the defaultType
										Mountpoint:   "/",
										FSTabOptions: "defaults",
										UUID:         "a178892e-e285-4ce1-9114-55780875d64e",
									},
								},
							},
						},
					},
					{
						Start:    714 * datasizes.MiB,
						Size:     104 * datasizes.MiB, // the sum of the LVs (rounded to the next 4 MiB extent) grows by 1 grain size (1 MiB) minus the unaligned size of the header to fit the gpt footer
						Type:     disk.LVMPartitionGUID,
						UUID:     "c75e7a81-bfde-475f-a7cf-e242cf3cc354",
						Bootable: false,
						Payload: &disk.LVMVolumeGroup{
							Name:        "vg01",
							Description: "created via lvm2 and osbuild",
							LogicalVolumes: []disk.LVMLogicalVolume{
								{
									Name: "datalv",
									Size: 100 * datasizes.MiB,
									Payload: &disk.Filesystem{
										Label:        "data",
										Type:         "ext4", // the defaultType
										Mountpoint:   "/data",
										FSTabOptions: "defaults",
										UUID:         "e2d3d0d0-de6b-48f9-b44c-e85ff044c6b1",
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
				Size: 714*datasizes.MiB + 230*datasizes.MiB + datasizes.MiB, // start + size of last partition + footer
				UUID: "0194fdc2-fa2f-4cc0-81d3-ff12045b73c8",
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
						UUID:     "a178892e-e285-4ce1-9114-55780875d64e",
						Bootable: false,
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "boot",
							Mountpoint:   "/boot",
							UUID:         "6e4ff95f-f662-45ee-a82a-bdf44a2d0b75",
							FSTabOptions: "defaults",
							FSTabFreq:    0,
							FSTabPassNo:  0,
						},
					},
					{
						Start:    714 * datasizes.MiB,
						Size:     231*datasizes.MiB - (disk.DefaultSectorSize + (128 * 128)), // grows by 1 grain size (1 MiB) minus the unaligned size of the header to fit the gpt footer
						Type:     disk.FilesystemDataGUID,
						UUID:     "e2d3d0d0-de6b-48f9-b44c-e85ff044c6b1",
						Bootable: false,
						Payload: &disk.Btrfs{
							UUID: "fb180daf-48a7-4ee0-b10d-394651850fd4",
							Subvolumes: []disk.BtrfsSubvolume{
								{
									Name:       "subvol/root",
									Mountpoint: "/",
									UUID:       "fb180daf-48a7-4ee0-b10d-394651850fd4", // same as volume UUID
								},
								{
									Name:       "subvol/home",
									Mountpoint: "/home",
									UUID:       "fb180daf-48a7-4ee0-b10d-394651850fd4", // same as volume UUID
								},
								{
									Name:       "subvol/varlog",
									Mountpoint: "/var/log",
									UUID:       "fb180daf-48a7-4ee0-b10d-394651850fd4", // same as volume UUID
								},
							},
						},
					},
				},
			},
		},
		"autorootbtrfs": {
			customizations: &blueprint.PartitioningCustomization{
				Btrfs: &blueprint.BtrfsCustomization{
					Volumes: []blueprint.BtrfsVolumeCustomization{
						{},
					},
				},
			},
			options: nil,
			expected: &disk.PartitionTable{
				Type: "gpt",
				Size: 514 * datasizes.MiB,
				UUID: "0194fdc2-fa2f-4cc0-81d3-ff12045b73c8",
				Partitions: []disk.Partition{
					{
						Start:    1 * datasizes.MiB,
						Size:     512 * datasizes.MiB,
						Type:     disk.XBootLDRPartitionGUID,
						UUID:     "a178892e-e285-4ce1-9114-55780875d64e",
						Bootable: false,
						Payload: &disk.Filesystem{
							Type:         "xfs",
							Label:        "boot",
							Mountpoint:   "/boot",
							UUID:         "6e4ff95f-f662-45ee-a82a-bdf44a2d0b75",
							FSTabOptions: "defaults",
							FSTabFreq:    0,
							FSTabPassNo:  0,
						},
					},
					{
						Start:    513 * datasizes.MiB,
						Size:     1*datasizes.MiB - (disk.DefaultSectorSize + (128 * 128)), // grows by 1 grain size (1 MiB) minus the unaligned size of the header to fit the gpt footer
						Type:     disk.FilesystemDataGUID,
						UUID:     "e2d3d0d0-de6b-48f9-b44c-e85ff044c6b1",
						Bootable: false,
						Payload: &disk.Btrfs{
							UUID: "fb180daf-48a7-4ee0-b10d-394651850fd4",
							Subvolumes: []disk.BtrfsSubvolume{
								{
									Name:       "root",
									Mountpoint: "/",
									UUID:       "fb180daf-48a7-4ee0-b10d-394651850fd4", // same as volume UUID
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

			// Initialise rng for each test separately, otherwise test run
			// order will affect results
			/* #nosec G404 */
			rnd := rand.New(rand.NewSource(0))
			pt, err := disk.NewCustomPartitionTable(tc.customizations, tc.options, rnd)

			assert.NoError(err)
			assert.Equal(tc.expected, pt)
		})
	}

}

func TestNewCustomPartitionTableErrors(t *testing.T) {
	type testCase struct {
		customizations *blueprint.PartitioningCustomization
		options        *disk.CustomPartitionTableOptions
		errmsg         string
	}

	testCases := map[string]testCase{
		"autoroot-notype": {
			customizations: nil,
			options:        nil,
			errmsg:         "error creating root partition: no default filesystem type",
		},
		"autorootlv-notype": {
			customizations: &blueprint.PartitioningCustomization{
				LVM: &blueprint.LVMCustomization{
					VolumeGroups: []blueprint.VGCustomization{
						{
							Name: "vg-without-root",
						},
					},
				},
			},
			options: nil,
			errmsg:  "error creating root logical volume: no default filesystem type",
		},
		"notype-nodefault": {
			customizations: &blueprint.PartitioningCustomization{
				Plain: &blueprint.PlainFilesystemCustomization{
					Filesystems: []blueprint.FilesystemCustomization{
						{
							Mountpoint: "/",
						},
					},
				},
			},
			options: nil,
			errmsg:  `error generating partition table: error creating partition with mountpoint "/": no filesystem type defined and no default set`,
		},
		"lvm-notype-nodefault": {
			customizations: &blueprint.PartitioningCustomization{
				LVM: &blueprint.LVMCustomization{
					VolumeGroups: []blueprint.VGCustomization{
						{
							Name: "rootvg",
							LogicalVolumes: []blueprint.LVCustomization{
								{
									Name: "rootlv",
									FilesystemCustomization: blueprint.FilesystemCustomization{
										Mountpoint: "/",
									},
								},
							},
						},
					},
				},
			},
			options: nil,
			errmsg:  `error generating partition table: error creating logical volume "rootlv" (/): no filesystem type defined and no default set`,
		},
	}

	// we don't care about the rng for error tests
	/* #nosec G404 */
	rnd := rand.New(rand.NewSource(0))

	for name := range testCases {
		tc := testCases[name]
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			_, err := disk.NewCustomPartitionTable(tc.customizations, tc.options, rnd)
			assert.EqualError(err, tc.errmsg)
		})
	}

}
