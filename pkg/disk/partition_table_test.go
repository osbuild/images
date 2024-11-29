package disk_test

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func TestNewCustomPartitionTable(t *testing.T) {
	type testCase struct {
		customizations *blueprint.DiskCustomization
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
				Type: disk.PT_DOS,
				Size: 202 * datasizes.MiB,
				UUID: "0194fdc2-fa2f-4cc0-81d3-ff12045b73c8",
				Partitions: []disk.Partition{
					{
						Start:    1 * datasizes.MiB, // header
						Bootable: true,
						Size:     1 * datasizes.MiB,
						Type:     disk.DosBIOSBootID,
						UUID:     disk.BIOSBootPartitionUUID,
					},
					{
						Start: 2 * datasizes.MiB,
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
					{
						Start:    202 * datasizes.MiB,
						Size:     0,
						Type:     disk.DosLinuxTypeID,
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
			customizations: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						MinSize: 20 * datasizes.MiB,
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							Mountpoint: "/data",
							Label:      "data",
							FSType:     "ext4",
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
				Type: disk.PT_DOS,
				Size: 222 * datasizes.MiB,
				UUID: "0194fdc2-fa2f-4cc0-81d3-ff12045b73c8",
				Partitions: []disk.Partition{
					{
						Start:    1 * datasizes.MiB, // header
						Size:     1 * datasizes.MiB,
						Bootable: true,
						Type:     disk.DosBIOSBootID,
						UUID:     disk.BIOSBootPartitionUUID,
					},
					{
						Start: 2 * datasizes.MiB,
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
					{
						Start:    202 * datasizes.MiB,
						Size:     20 * datasizes.MiB,
						Type:     disk.DosLinuxTypeID,
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
						Type:     disk.DosLinuxTypeID,
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
		"plain+swap": {
			customizations: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						MinSize: 20 * datasizes.MiB,
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							Mountpoint: "/data",
							Label:      "data",
							FSType:     "ext4",
						},
					},
					{
						MinSize: 5 * datasizes.MiB,
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							Label:  "swap",
							FSType: "swap",
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
				Type: disk.PT_DOS,
				Size: 227 * datasizes.MiB,
				UUID: "0194fdc2-fa2f-4cc0-81d3-ff12045b73c8",
				Partitions: []disk.Partition{
					{
						Start:    1 * datasizes.MiB, // header
						Size:     1 * datasizes.MiB,
						Bootable: true,
						Type:     disk.DosBIOSBootID,
						UUID:     disk.BIOSBootPartitionUUID,
					},
					{
						Start: 2 * datasizes.MiB,
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
					{
						Start:    202 * datasizes.MiB,
						Size:     20 * datasizes.MiB,
						Type:     disk.DosLinuxTypeID,
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
						Size:     5 * datasizes.MiB,
						Type:     disk.DosSwapID,
						UUID:     "", // partitions on dos PTs don't have UUIDs
						Bootable: false,
						Payload: &disk.Swap{
							Label:        "swap",
							UUID:         "fb180daf-48a7-4ee0-b10d-394651850fd4",
							FSTabOptions: "defaults",
						},
					},
					{
						Start:    227 * datasizes.MiB,
						Size:     0,
						Type:     disk.DosLinuxTypeID,
						UUID:     "", // partitions on dos PTs don't have UUIDs
						Bootable: false,
						Payload: &disk.Filesystem{
							Type:         "xfs",
							Label:        "root",
							Mountpoint:   "/",
							UUID:         "a178892e-e285-4ce1-9114-55780875d64e",
							FSTabOptions: "defaults",
							FSTabFreq:    0,
							FSTabPassNo:  0,
						},
					},
				},
			},
		},
		"plain-legacy": {
			customizations: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						MinSize: 20 * datasizes.MiB,
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							Mountpoint: "/data",
							Label:      "data",
							FSType:     "ext4",
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
				Type: disk.PT_DOS,
				Size: 22 * datasizes.MiB,
				UUID: "0194fdc2-fa2f-4cc0-81d3-ff12045b73c8",
				Partitions: []disk.Partition{
					{
						Start:    1 * datasizes.MiB, // header
						Size:     1 * datasizes.MiB,
						Bootable: true,
						Type:     disk.DosBIOSBootID,
						UUID:     disk.BIOSBootPartitionUUID,
					},
					{
						Start:    2 * datasizes.MiB,
						Size:     20 * datasizes.MiB,
						Type:     disk.DosLinuxTypeID,
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
						Type:     disk.DosLinuxTypeID,
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
			customizations: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						MinSize: 20 * datasizes.MiB,
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							Mountpoint: "/data",
							Label:      "data",
							FSType:     "ext4",
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
				Type: disk.PT_DOS,
				Size: 221 * datasizes.MiB,
				UUID: "0194fdc2-fa2f-4cc0-81d3-ff12045b73c8",
				Partitions: []disk.Partition{
					{
						Start: 1 * datasizes.MiB,
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
					{
						Start:    201 * datasizes.MiB,
						Size:     20 * datasizes.MiB,
						Type:     disk.DosLinuxTypeID,
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
						Type:     disk.DosLinuxTypeID,
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
			customizations: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						MinSize: 20 * datasizes.MiB,
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							Mountpoint: "/data",
							Label:      "data",
							FSType:     "ext4",
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
				Type: disk.PT_DOS,
				Size: 222*datasizes.MiB + 3*datasizes.GiB,
				UUID: "0194fdc2-fa2f-4cc0-81d3-ff12045b73c8",
				Partitions: []disk.Partition{
					{
						Start:    1 * datasizes.MiB, // header
						Size:     1 * datasizes.MiB,
						Bootable: true,
						Type:     disk.DosBIOSBootID,
						UUID:     disk.BIOSBootPartitionUUID,
					},
					{
						Start: 2 * datasizes.MiB,
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
					{
						Start:    202 * datasizes.MiB,
						Size:     20 * datasizes.MiB,
						Type:     disk.DosLinuxTypeID,
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
						Type:     disk.DosLinuxTypeID,
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
			customizations: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						MinSize: 50 * datasizes.MiB,
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							Mountpoint: "/",
							Label:      "root",
							FSType:     "xfs",
						},
					},
					{
						MinSize: 20 * datasizes.MiB,
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							Mountpoint: "/home",
							Label:      "home",
							FSType:     "ext4",
						},
					},
					{
						MinSize: 12 * datasizes.MiB,
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							Label:  "swappyswaps",
							FSType: "swap",
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
				Type: disk.PT_GPT,
				Size: 234*datasizes.MiB + 3*datasizes.GiB + datasizes.MiB, // start + size of last partition + footer

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
						Start:    234 * datasizes.MiB,
						Size:     3*datasizes.GiB + datasizes.MiB - (disk.DefaultSectorSize + (128 * 128)), // grows by 1 grain size (1 MiB) minus the unaligned size of the header to fit the gpt footer
						Type:     disk.FilesystemDataGUID,
						UUID:     "e2d3d0d0-de6b-48f9-b44c-e85ff044c6b1",
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
						UUID:     "f83b8e88-3bbf-457a-ab99-c5b252c7429c",
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
					{
						Start: 222 * datasizes.MiB,
						Size:  12 * datasizes.MiB,
						Type:  disk.SwapPartitionGUID,
						UUID:  "32f3a8ae-b79e-4856-b659-c18f0dcecc77",
						Payload: &disk.Swap{
							Label:        "swappyswaps",
							UUID:         "a178892e-e285-4ce1-9114-55780875d64e",
							FSTabOptions: "defaults",
						},
					},
				},
			},
		},
		"lvm": {
			customizations: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type:    "lvm",
						MinSize: 100 * datasizes.MiB,
						VGCustomization: blueprint.VGCustomization{
							Name: "testvg",
							LogicalVolumes: []blueprint.LVCustomization{
								{
									Name:    "varloglv",
									MinSize: 10 * datasizes.MiB,
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
										Mountpoint: "/var/log",
										Label:      "var-log",
										FSType:     "xfs",
									},
								},
								{
									Name:    "rootlv",
									MinSize: 50 * datasizes.MiB,
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
										Mountpoint: "/",
										Label:      "root",
										FSType:     "xfs",
									},
								},
								{ // unnamed + untyped logical volume
									MinSize: 100 * datasizes.MiB,
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
										Mountpoint: "/data",
										Label:      "data",
										FSType:     "ext4", // TODO: remove when we reintroduce the default fs
									},
								},
								{ // swap on LV
									Name:    "swaplv",
									MinSize: 30 * datasizes.MiB,
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
										Label:  "swap-on-lv",
										FSType: "swap",
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
				Type: disk.PT_GPT, // default when unspecified
				UUID: "0194fdc2-fa2f-4cc0-81d3-ff12045b73c8",
				Size: 714*datasizes.MiB + 200*datasizes.MiB + datasizes.MiB, // start + size of last partition (VG) + footer
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
						UUID:     "32f3a8ae-b79e-4856-b659-c18f0dcecc77",
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
						Size:     200*datasizes.MiB + datasizes.MiB - (disk.DefaultSectorSize + (128 * 128)), // the sum of the LVs (rounded to the next 4 MiB extent) grows by 1 grain size (1 MiB) minus the unaligned size of the header to fit the gpt footer
						Type:     disk.LVMPartitionGUID,
						UUID:     "c75e7a81-bfde-475f-a7cf-e242cf3cc354",
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
								{
									Name: "swaplv",
									Size: 32 * datasizes.MiB, // rounded up to the next extent (4 MiB)
									Payload: &disk.Swap{
										Label:        "swap-on-lv",
										UUID:         "f83b8e88-3bbf-457a-ab99-c5b252c7429c",
										FSTabOptions: "defaults",
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
			// PT creation function can handle it
			customizations: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type:    "lvm",
						MinSize: 100 * datasizes.MiB,
						VGCustomization: blueprint.VGCustomization{
							LogicalVolumes: []blueprint.LVCustomization{
								{
									Name:    "varloglv",
									MinSize: 10 * datasizes.MiB,
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
										Mountpoint: "/var/log",
										Label:      "var-log",
										FSType:     "xfs",
									},
								},
							},
						},
					},
					{
						Type: "lvm",
						VGCustomization: blueprint.VGCustomization{
							LogicalVolumes: []blueprint.LVCustomization{
								{ // unnamed + untyped logical volume
									MinSize: 100 * datasizes.MiB,
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
										Mountpoint: "/data",
										Label:      "data",
										FSType:     "ext4", // TODO: remove when we reintroduce the default fs
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
				Type: disk.PT_GPT, // default when unspecified
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
			customizations: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type:    "btrfs",
						MinSize: 230 * datasizes.MiB,
						BtrfsVolumeCustomization: blueprint.BtrfsVolumeCustomization{
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
					{
						MinSize: 120 * datasizes.MiB,
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							Label:  "butterswap",
							FSType: "swap",
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
				Type: disk.PT_GPT,
				Size: 834*datasizes.MiB + 230*datasizes.MiB + datasizes.MiB, // start + size of last partition + footer
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
						UUID:     "e2d3d0d0-de6b-48f9-b44c-e85ff044c6b1",
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
						Start:    834 * datasizes.MiB,
						Size:     231*datasizes.MiB - (disk.DefaultSectorSize + (128 * 128)), // grows by 1 grain size (1 MiB) minus the unaligned size of the header to fit the gpt footer
						Type:     disk.FilesystemDataGUID,
						UUID:     "f83b8e88-3bbf-457a-ab99-c5b252c7429c",
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
					{
						Start: 714 * datasizes.MiB,
						Size:  120 * datasizes.MiB,
						Type:  disk.SwapPartitionGUID,
						UUID:  "32f3a8ae-b79e-4856-b659-c18f0dcecc77",
						Payload: &disk.Swap{
							Label:        "butterswap",
							UUID:         "a178892e-e285-4ce1-9114-55780875d64e",
							FSTabOptions: "defaults",
						},
					},
				},
			},
		},
		"autorootbtrfs": {
			customizations: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "btrfs",
						BtrfsVolumeCustomization: blueprint.BtrfsVolumeCustomization{
							Subvolumes: []blueprint.BtrfsSubvolumeCustomization{
								{
									Name:       "data",
									Mountpoint: "/data",
								},
							},
						},
					},
				},
			},
			options: nil,
			expected: &disk.PartitionTable{
				Type: disk.PT_GPT,
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
						Start: 513 * datasizes.MiB,
						Size:  1*datasizes.MiB - (disk.DefaultSectorSize + (128 * 128)),

						Type:     disk.FilesystemDataGUID,
						UUID:     "e2d3d0d0-de6b-48f9-b44c-e85ff044c6b1",
						Bootable: false,
						Payload: &disk.Btrfs{
							UUID: "fb180daf-48a7-4ee0-b10d-394651850fd4",
							Subvolumes: []disk.BtrfsSubvolume{
								{
									Name:       "data",
									Mountpoint: "/data",
									UUID:       "fb180daf-48a7-4ee0-b10d-394651850fd4",
								},
								{
									Name:       "root",
									Mountpoint: "/",
									UUID:       "fb180daf-48a7-4ee0-b10d-394651850fd4",
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
		customizations *blueprint.DiskCustomization
		options        *disk.CustomPartitionTableOptions
		errmsg         string
	}

	testCases := map[string]testCase{
		"autoroot-notype": {
			customizations: nil,
			options:        nil,
			errmsg:         "error generating partition table: error creating root partition: no default filesystem type",
		},
		"autorootlv-notype": {
			customizations: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "lvm",
						VGCustomization: blueprint.VGCustomization{
							Name: "vg-without-root",
						},
					},
				},
			},
			options: nil,
			errmsg:  "error generating partition table: error creating root logical volume: no default filesystem type",
		},
		"notype-nodefault": {
			customizations: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							Mountpoint: "/",
						},
					},
				},
			},
			options: nil,
			// NOTE: this error message will change when we allow empty fs_type
			// in customizations but with a requirement to define a default
			errmsg: "error generating partition table: invalid partitioning customizations:\nunknown or invalid filesystem type for mountpoint \"/\": ",
		},
		"lvm-notype-nodefault": {
			customizations: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "lvm",
						VGCustomization: blueprint.VGCustomization{
							Name: "rootvg",
							LogicalVolumes: []blueprint.LVCustomization{
								{
									Name: "rootlv",
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
										Mountpoint: "/",
									},
								},
							},
						},
					},
				},
			},
			options: nil,
			// NOTE: this error message will change when we allow empty fs_type
			// in customizations but with a requirement to define a default
			errmsg: "error generating partition table: invalid partitioning customizations:\nunknown or invalid filesystem type for logical volume with mountpoint \"/\": ",
		},
		"bad-pt-type": {
			options: &disk.CustomPartitionTableOptions{
				PartitionTableType: 100,
			},
			errmsg: `error generating partition table: invalid partition table type enum value: 100`,
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

func TestPartitionTableFeatures(t *testing.T) {
	require := require.New(t)

	testCases := map[string]disk.PartitionTableFeatures{
		"plain":        {XFS: true, FAT: true},
		"plain-noboot": {XFS: true, FAT: true},
		"plain-swap":   {XFS: true, FAT: true, Swap: true},
		"luks":         {XFS: true, FAT: true, LUKS: true},
		"luks+lvm":     {XFS: true, FAT: true, LUKS: true, LVM: true},
		"btrfs":        {XFS: true, FAT: true, Btrfs: true},
	}

	for name := range testdisk.TestPartitionTables {
		// print an informative failure message if a new test partition
		// table is added and this test is not updated (instead of failing
		// at the final Equal() check)
		exp, ok := testCases[name]
		require.True(ok, "expected test result not defined for test partition table %q: please update the %s test", name, t.Name())
		pt := testdisk.TestPartitionTables[name]
		require.Equal(exp, disk.GetPartitionTableFeatures(pt))
	}
}
