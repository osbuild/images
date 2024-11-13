package disk_test

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/internal/testdisk"
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
		Type: disk.PT_DOS,
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

func TestEnsureRootFilesystem(t *testing.T) {
	type testCase struct {
		pt            disk.PartitionTable
		expected      disk.PartitionTable
		defaultFsType disk.FSType
	}

	testCases := map[string]testCase{
		"empty-plain-gpt": {
			pt:            disk.PartitionTable{Type: disk.PT_GPT},
			defaultFsType: disk.FS_EXT4,
			expected: disk.PartitionTable{
				Type: disk.PT_GPT,
				Partitions: []disk.Partition{
					{
						Start:    0,
						Size:     0,
						Type:     disk.FilesystemDataGUID,
						Bootable: false,
						UUID:     "",
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "root",
							Mountpoint:   "/",
							FSTabOptions: "defaults",
						},
					},
				},
			},
		},
		"empty-plain-dos": {
			pt:            disk.PartitionTable{Type: disk.PT_DOS},
			defaultFsType: disk.FS_EXT4,
			expected: disk.PartitionTable{
				Type: disk.PT_DOS,
				Partitions: []disk.Partition{
					{
						Start:    0,
						Size:     0,
						Type:     disk.DosLinuxTypeID,
						Bootable: false,
						UUID:     "",
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "root",
							Mountpoint:   "/",
							FSTabOptions: "defaults",
						},
					},
				},
			},
		},
		"simple-plain-gpt": {
			pt: disk.PartitionTable{
				Type: disk.PT_GPT,
				Partitions: []disk.Partition{
					{
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "home",
							Mountpoint:   "/home",
							FSTabOptions: "defaults",
						},
					},
				},
			},
			defaultFsType: disk.FS_EXT4,
			expected: disk.PartitionTable{
				Type: disk.PT_GPT,
				Partitions: []disk.Partition{
					{
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "home",
							Mountpoint:   "/home",
							FSTabOptions: "defaults",
						},
					},
					{
						Start:    0,
						Size:     0,
						Type:     disk.FilesystemDataGUID,
						Bootable: false,
						UUID:     "",
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "root",
							Mountpoint:   "/",
							FSTabOptions: "defaults",
						},
					},
				},
			},
		},
		"simple-plain-dos": {
			pt: disk.PartitionTable{
				Type: disk.PT_DOS,
				Partitions: []disk.Partition{
					{
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "home",
							Mountpoint:   "/home",
							FSTabOptions: "defaults",
						},
					},
				},
			},
			defaultFsType: disk.FS_EXT4,
			expected: disk.PartitionTable{
				Type: disk.PT_DOS,
				Partitions: []disk.Partition{
					{
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "home",
							Mountpoint:   "/home",
							FSTabOptions: "defaults",
						},
					},
					{
						Start:    0,
						Size:     0,
						Type:     disk.DosLinuxTypeID,
						Bootable: false,
						UUID:     "",
						Payload: &disk.Filesystem{
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
			pt: disk.PartitionTable{
				Partitions: []disk.Partition{
					{
						Payload: &disk.LVMVolumeGroup{
							Name: "testvg",
							LogicalVolumes: []disk.LVMLogicalVolume{
								{
									Name: "varloglv",
									Payload: &disk.Filesystem{
										Label:      "var-log",
										Type:       "xfs",
										Mountpoint: "/var/log",
									},
								},
								{
									Name: "datalv",
									Payload: &disk.Filesystem{
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
			defaultFsType: disk.FS_EXT4,
			expected: disk.PartitionTable{
				Partitions: []disk.Partition{
					{
						Payload: &disk.LVMVolumeGroup{
							Name: "testvg",
							LogicalVolumes: []disk.LVMLogicalVolume{
								{
									Name: "varloglv",
									Payload: &disk.Filesystem{
										Label:      "var-log",
										Type:       "xfs",
										Mountpoint: "/var/log",
									},
								},
								{
									Name: "datalv",
									Payload: &disk.Filesystem{
										Label:        "data",
										Type:         "ext4",
										Mountpoint:   "/data",
										FSTabOptions: "defaults",
									},
								},
								{
									Name: "rootlv",
									Payload: &disk.Filesystem{
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
			pt: disk.PartitionTable{
				Partitions: []disk.Partition{
					{
						Payload: &disk.Btrfs{
							Subvolumes: []disk.BtrfsSubvolume{
								{
									Name:       "subvol/home",
									Mountpoint: "/home",
								},
							},
						},
					},
				},
			},
			// no default fs required
			expected: disk.PartitionTable{
				Partitions: []disk.Partition{
					{
						Payload: &disk.Btrfs{
							Subvolumes: []disk.BtrfsSubvolume{
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
			pt: disk.PartitionTable{
				Partitions: []disk.Partition{
					{
						Payload: &disk.LVMVolumeGroup{
							Name: "testvg",
							LogicalVolumes: []disk.LVMLogicalVolume{
								{
									Name: "varloglv",
									Payload: &disk.Filesystem{
										Label:      "var-log",
										Type:       "xfs",
										Mountpoint: "/var/log",
									},
								},
								{
									Name: "datalv",
									Payload: &disk.Filesystem{
										Label:        "data",
										Type:         "ext4",
										Mountpoint:   "/data",
										FSTabOptions: "defaults",
									},
								},
								{
									Name: "rootlv",
									Payload: &disk.Filesystem{
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
			expected: disk.PartitionTable{
				Partitions: []disk.Partition{
					{
						Payload: &disk.LVMVolumeGroup{
							Name: "testvg",
							LogicalVolumes: []disk.LVMLogicalVolume{
								{
									Name: "varloglv",
									Payload: &disk.Filesystem{
										Label:      "var-log",
										Type:       "xfs",
										Mountpoint: "/var/log",
									},
								},
								{
									Name: "datalv",
									Payload: &disk.Filesystem{
										Label:        "data",
										Type:         "ext4",
										Mountpoint:   "/data",
										FSTabOptions: "defaults",
									},
								},
								{
									Name: "rootlv",
									Payload: &disk.Filesystem{
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
			pt: disk.PartitionTable{
				Partitions: []disk.Partition{
					{
						Payload: &disk.Btrfs{
							Subvolumes: []disk.BtrfsSubvolume{
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
			expected: disk.PartitionTable{
				Partitions: []disk.Partition{
					{
						Payload: &disk.Btrfs{
							Subvolumes: []disk.BtrfsSubvolume{
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
		"plain-collision": {
			pt: disk.PartitionTable{
				Type: disk.PT_GPT,
				Partitions: []disk.Partition{
					{
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "root",
							Mountpoint:   "/root",
							FSTabOptions: "defaults",
						},
					},
				},
			},
			defaultFsType: disk.FS_EXT4,
			expected: disk.PartitionTable{
				Type: disk.PT_GPT,
				Partitions: []disk.Partition{
					{
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "root",
							Mountpoint:   "/root",
							FSTabOptions: "defaults",
						},
					},
					{
						Start:    0,
						Size:     0,
						Type:     disk.FilesystemDataGUID,
						Bootable: false,
						UUID:     "",
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "root00",
							Mountpoint:   "/",
							FSTabOptions: "defaults",
						},
					},
				},
			},
		},
		"lvm-collision": {
			pt: disk.PartitionTable{
				Type: disk.PT_GPT,
				Partitions: []disk.Partition{
					{
						Payload: &disk.LVMVolumeGroup{
							Name: "testvg",
							LogicalVolumes: []disk.LVMLogicalVolume{
								{
									Name: "varloglv",
									Payload: &disk.Filesystem{
										Label:      "var-log",
										Type:       "xfs",
										Mountpoint: "/var/log",
									},
								},
								{
									Name: "datalv",
									Payload: &disk.Filesystem{
										Label:        "data",
										Type:         "ext4",
										Mountpoint:   "/data",
										FSTabOptions: "defaults",
									},
								},
								{
									Name: "rootlv",
									Payload: &disk.Filesystem{
										Label:        "root",
										Type:         "ext4",
										Mountpoint:   "/root",
										FSTabOptions: "defaults",
									},
								},
							},
						},
					},
				},
			},
			defaultFsType: disk.FS_XFS,
			expected: disk.PartitionTable{
				Type: disk.PT_GPT,
				Partitions: []disk.Partition{
					{
						Payload: &disk.LVMVolumeGroup{
							Name: "testvg",
							LogicalVolumes: []disk.LVMLogicalVolume{
								{
									Name: "varloglv",
									Payload: &disk.Filesystem{
										Label:      "var-log",
										Type:       "xfs",
										Mountpoint: "/var/log",
									},
								},
								{
									Name: "datalv",
									Payload: &disk.Filesystem{
										Label:        "data",
										Type:         "ext4",
										Mountpoint:   "/data",
										FSTabOptions: "defaults",
									},
								},
								{
									Name: "rootlv",
									Payload: &disk.Filesystem{
										Label:        "root",
										Type:         "ext4",
										Mountpoint:   "/root",
										FSTabOptions: "defaults",
									},
								},
								{
									Name: "rootlv00",
									Payload: &disk.Filesystem{
										Label:        "root00",
										Type:         "xfs",
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
		"btrfs-collision": {
			pt: disk.PartitionTable{
				Partitions: []disk.Partition{
					{
						Payload: &disk.Btrfs{
							Subvolumes: []disk.BtrfsSubvolume{
								{
									Name:       "subvol/home",
									Mountpoint: "/home",
								},
								{
									Name:       "root",
									Mountpoint: "/root",
								},
							},
						},
					},
				},
			},
			expected: disk.PartitionTable{
				Partitions: []disk.Partition{
					{
						Payload: &disk.Btrfs{
							Subvolumes: []disk.BtrfsSubvolume{
								{
									Name:       "subvol/home",
									Mountpoint: "/home",
								},
								{
									Name:       "root",
									Mountpoint: "/root",
								},
								{
									Name:       "root00",
									Mountpoint: "/",
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
			pt := tc.pt
			err := disk.EnsureRootFilesystem(&pt, tc.defaultFsType)
			assert.NoError(err)
			assert.Equal(tc.expected, pt)
		})
	}
}

func TestEnsureRootFilesystemErrors(t *testing.T) {
	type testCase struct {
		pt            disk.PartitionTable
		defaultFsType disk.FSType
		errmsg        string
	}

	testCases := map[string]testCase{
		"err-empty": {
			pt:     disk.PartitionTable{},
			errmsg: "error creating root partition: no default filesystem type",
		},
		"err-no-pt-type": {
			pt:            disk.PartitionTable{},
			defaultFsType: disk.FS_EXT4,
			errmsg:        "error creating root partition: unknown or unsupported partition table enum: 0",
		},
		"err-plain": {
			pt: disk.PartitionTable{
				Partitions: []disk.Partition{
					{
						Payload: &disk.Filesystem{
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
			pt: disk.PartitionTable{
				Partitions: []disk.Partition{
					{
						Payload: &disk.LVMVolumeGroup{
							Name: "testvg",
							LogicalVolumes: []disk.LVMLogicalVolume{
								{
									Name: "varloglv",
									Payload: &disk.Filesystem{
										Label:      "var-log",
										Type:       "xfs",
										Mountpoint: "/var/log",
									},
								},
								{
									Name: "datalv",
									Payload: &disk.Filesystem{
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
			err := disk.EnsureRootFilesystem(&pt, tc.defaultFsType)
			assert.EqualError(err, tc.errmsg)
		})
	}
}

func TestAddBootPartition(t *testing.T) {
	type testCase struct {
		pt       disk.PartitionTable
		expected disk.PartitionTable
		fsType   disk.FSType
		errmsg   string
	}

	testCases := map[string]testCase{
		"empty-plain-gpt": {
			pt:     disk.PartitionTable{Type: disk.PT_GPT},
			fsType: disk.FS_EXT4,
			expected: disk.PartitionTable{
				Type: disk.PT_GPT,
				Partitions: []disk.Partition{
					{
						Start:    0,
						Size:     512 * datasizes.MiB,
						Type:     disk.XBootLDRPartitionGUID,
						Bootable: false,
						UUID:     "",
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "boot",
							Mountpoint:   "/boot",
							FSTabOptions: "defaults",
						},
					},
				},
			},
		},
		"empty-plain-dos": {
			pt:     disk.PartitionTable{Type: disk.PT_DOS},
			fsType: disk.FS_EXT4,
			expected: disk.PartitionTable{
				Type: disk.PT_DOS,
				Partitions: []disk.Partition{
					{
						Start:    0,
						Size:     512 * datasizes.MiB,
						Type:     disk.DosLinuxTypeID,
						Bootable: false,
						UUID:     "",
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "boot",
							Mountpoint:   "/boot",
							FSTabOptions: "defaults",
						},
					},
				},
			},
		},
		"simple-plain-gpt": {
			pt: disk.PartitionTable{
				Type: disk.PT_GPT,
				Partitions: []disk.Partition{
					{
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "home",
							Mountpoint:   "/home",
							FSTabOptions: "defaults",
						},
					},
				},
			},
			fsType: disk.FS_EXT4,
			expected: disk.PartitionTable{
				Type: disk.PT_GPT,
				Partitions: []disk.Partition{
					{
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "home",
							Mountpoint:   "/home",
							FSTabOptions: "defaults",
						},
					},
					{
						Start:    0,
						Size:     512 * datasizes.MiB,
						Type:     disk.XBootLDRPartitionGUID,
						Bootable: false,
						UUID:     "",
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "boot",
							Mountpoint:   "/boot",
							FSTabOptions: "defaults",
						},
					},
				},
			},
		},
		"simple-plain-dos": {
			pt: disk.PartitionTable{
				Type: disk.PT_DOS,
				Partitions: []disk.Partition{
					{
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "home",
							Mountpoint:   "/home",
							FSTabOptions: "defaults",
						},
					},
				},
			},
			fsType: disk.FS_EXT4,
			expected: disk.PartitionTable{
				Type: disk.PT_DOS,
				Partitions: []disk.Partition{
					{
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "home",
							Mountpoint:   "/home",
							FSTabOptions: "defaults",
						},
					},
					{
						Start:    0,
						Size:     512 * datasizes.MiB,
						Type:     disk.DosLinuxTypeID,
						Bootable: false,
						UUID:     "",
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "boot",
							Mountpoint:   "/boot",
							FSTabOptions: "defaults",
						},
					},
				},
			},
		},
		"label-collision": {
			pt: disk.PartitionTable{
				Type: disk.PT_GPT,
				Partitions: []disk.Partition{
					{
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "boot",
							Mountpoint:   "/collections/footwear/boot",
							FSTabOptions: "defaults",
						},
					},
				},
			},
			fsType: disk.FS_EXT4,
			expected: disk.PartitionTable{
				Type: disk.PT_GPT,
				Partitions: []disk.Partition{
					{
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "boot",
							Mountpoint:   "/collections/footwear/boot",
							FSTabOptions: "defaults",
						},
					},
					{
						Start:    0,
						Size:     512 * datasizes.MiB,
						Type:     disk.XBootLDRPartitionGUID,
						Bootable: false,
						UUID:     "",
						Payload: &disk.Filesystem{
							Type:         "ext4",
							Label:        "boot00",
							Mountpoint:   "/boot",
							FSTabOptions: "defaults",
						},
					},
				},
			},
		},
		"err-nofs": {
			pt:     disk.PartitionTable{},
			errmsg: "error creating boot partition: no filesystem type",
		},
	}

	for name := range testCases {
		tc := testCases[name]
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			pt := tc.pt
			err := disk.AddBootPartition(&pt, tc.fsType)
			if tc.errmsg == "" {
				assert.NoError(err)
				assert.Equal(tc.expected, pt)
			} else {
				assert.EqualError(err, tc.errmsg)
			}
		})
	}
}

func TestAddPartitionsForBootMode(t *testing.T) {
	type testCase struct {
		pt       disk.PartitionTable
		bootMode platform.BootMode
		expected disk.PartitionTable
		errmsg   string
	}

	testCases := map[string]testCase{
		// the partition table type shouldn't matter when the boot mode is
		// none, but let's test with both anyway
		"none-gpt": {
			pt:       disk.PartitionTable{Type: disk.PT_GPT},
			bootMode: platform.BOOT_NONE,
			expected: disk.PartitionTable{Type: disk.PT_GPT},
		},
		"none-dos": {
			pt:       disk.PartitionTable{Type: disk.PT_DOS},
			bootMode: platform.BOOT_NONE,
			expected: disk.PartitionTable{Type: disk.PT_DOS},
		},
		"bios-gpt": {
			pt:       disk.PartitionTable{Type: disk.PT_GPT},
			bootMode: platform.BOOT_LEGACY,
			expected: disk.PartitionTable{
				Type: disk.PT_GPT,
				Partitions: []disk.Partition{
					{
						Bootable: true,
						Start:    0,
						Size:     1 * datasizes.MiB,
						Type:     disk.BIOSBootPartitionGUID,
						UUID:     disk.BIOSBootPartitionUUID,
					},
				},
			},
		},
		"bios-dos": {
			pt:       disk.PartitionTable{Type: disk.PT_DOS},
			bootMode: platform.BOOT_LEGACY,
			expected: disk.PartitionTable{
				Type: disk.PT_DOS,
				Partitions: []disk.Partition{
					{
						Bootable: true,
						Start:    0,
						Size:     1 * datasizes.MiB,
						Type:     disk.DosBIOSBootID,
						UUID:     disk.BIOSBootPartitionUUID,
					},
				},
			},
		},
		"uefi-gpt": {
			pt:       disk.PartitionTable{Type: disk.PT_GPT},
			bootMode: platform.BOOT_UEFI,
			expected: disk.PartitionTable{
				Type: disk.PT_GPT,
				Partitions: []disk.Partition{
					{
						Start: 0 * datasizes.MiB,
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
				},
			},
		},
		"uefi-dos": {
			pt:       disk.PartitionTable{Type: disk.PT_DOS},
			bootMode: platform.BOOT_UEFI,
			expected: disk.PartitionTable{
				Type: disk.PT_DOS,
				Partitions: []disk.Partition{
					{
						Start: 0 * datasizes.MiB,
						Size:  200 * datasizes.MiB,
						Type:  disk.DosESPID,
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
				},
			},
		},
		"hybrid-gpt": {
			pt:       disk.PartitionTable{Type: disk.PT_GPT},
			bootMode: platform.BOOT_HYBRID,
			expected: disk.PartitionTable{
				Type: disk.PT_GPT,
				Partitions: []disk.Partition{
					{
						Size:     1 * datasizes.MiB,
						Bootable: true,
						Type:     disk.BIOSBootPartitionGUID,
						UUID:     disk.BIOSBootPartitionUUID,
					},
					{
						Size: 200 * datasizes.MiB,
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
				},
			},
		},
		"hybrid-dos": {
			pt:       disk.PartitionTable{Type: disk.PT_DOS},
			bootMode: platform.BOOT_HYBRID,
			expected: disk.PartitionTable{
				Type: disk.PT_DOS,
				Partitions: []disk.Partition{
					{
						Size:     1 * datasizes.MiB,
						Bootable: true,
						Type:     disk.DosBIOSBootID,
						UUID:     disk.BIOSBootPartitionUUID,
					},
					{
						Size: 200 * datasizes.MiB,
						Type: disk.DosESPID,
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
				},
			},
		},
		"bad-pttype-bios": {
			pt:       disk.PartitionTable{Type: disk.PartitionTableType(911)},
			bootMode: platform.BOOT_LEGACY,
			errmsg:   "error creating BIOS boot partition: unknown or unsupported partition table enum: 911",
		},
		"bad-pttype-uefi": {
			pt:       disk.PartitionTable{Type: disk.PartitionTableType(911)},
			bootMode: platform.BOOT_UEFI,
			errmsg:   "error creating EFI system partition: unknown or unsupported partition table enum: 911",
		},
		"bad-pttype-hybrid": {
			pt:       disk.PartitionTable{Type: disk.PartitionTableType(911)},
			bootMode: platform.BOOT_HYBRID,
			errmsg:   "error creating BIOS boot partition: unknown or unsupported partition table enum: 911",
		},
		"bad-bootmode": {
			pt:       disk.PartitionTable{Type: disk.PT_GPT},
			bootMode: 4,
			errmsg:   "unknown or unsupported boot mode type with enum value 4",
		},
	}

	for name := range testCases {
		tc := testCases[name]
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			pt := tc.pt
			err := disk.AddPartitionsForBootMode(&pt, tc.bootMode)
			if tc.errmsg == "" {
				assert.NoError(err)
				assert.Equal(tc.expected, pt)
			} else {
				assert.EqualError(err, tc.errmsg)
			}
		})
	}
}
