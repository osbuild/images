package blueprint_test

import (
	"encoding/json"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/datasizes"
	"github.com/osbuild/images/pkg/pathpolicy"
	"github.com/stretchr/testify/assert"
)

func TestPartitioningValidation(t *testing.T) {
	type testCase struct {
		partitioning *blueprint.DiskCustomization
		expectedMsg  string
	}

	testCases := map[string]testCase{
		"null": {
			partitioning: nil,
			expectedMsg:  "",
		},
		"happy-plain": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							FSType:     "xfs",
							Mountpoint: "/data",
						},
					},
				},
			},
			expectedMsg: "",
		},
		"happy-plain+btrfs": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							FSType:     "xfs",
							Mountpoint: "/data",
						},
					},
					{
						Type: "btrfs",
						BtrfsVolumeCustomization: blueprint.BtrfsVolumeCustomization{
							Subvolumes: []blueprint.BtrfsSubvolumeCustomization{
								{
									Name:       "root",
									Mountpoint: "/",
								},
							},
						},
					},
				},
			},
			expectedMsg: "",
		},
		"happy-plain+lvm": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "plain",
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							FSType:     "xfs",
							Mountpoint: "/data",
						},
					},
					{
						Type: "lvm",
						VGCustomization: blueprint.VGCustomization{
							Name: "root",
							LogicalVolumes: []blueprint.LVCustomization{
								{
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
										FSType:     "ext4",
										Mountpoint: "/",
									},
								},
							},
						},
					},
				},
			},
			expectedMsg: "",
		},
		"happy-plain-with-boot-and-efi": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							FSType:     "xfs",
							Mountpoint: "/data",
						},
					},
					{
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							FSType:     "ext4",
							Mountpoint: "/",
						},
					},
					{
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							FSType:     "ext4",
							Mountpoint: "/home",
						},
					},
					{
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							FSType:     "ext4",
							Mountpoint: "/boot",
						},
					},
					{
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							FSType:     "vfat",
							Mountpoint: "/boot/efi",
						},
					},
				},
			},
			expectedMsg: "",
		},
		"unhappy-plain-dupes": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							FSType:     "xfs",
							Mountpoint: "/data",
						},
					},
					{
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							FSType:     "ext4",
							Mountpoint: "/",
						},
					},
					{
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							FSType:     "ext4",
							Mountpoint: "/home",
						},
					},
					{
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							FSType:     "xfs",
							Mountpoint: "/data",
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\nduplicate mountpoint \"/data\" in partitioning customizations",
		},
		"unhappy-plain-badfstype": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							FSType:     "ext4",
							Mountpoint: "/",
						},
					},
					{
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							FSType:     "ntfs",
							Mountpoint: "/home",
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\nunknown or invalid filesystem type for mountpoint \"/home\": ntfs",
		},
		"unhappy-plain-badfstype-boot": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							FSType:     "xfs",
							Mountpoint: "/data",
						},
					},
					{
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							FSType:     "ext4",
							Mountpoint: "/",
						},
					},
					{
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							FSType:     "ext4",
							Mountpoint: "/home",
						},
					},
					{
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							FSType:     "zfs",
							Mountpoint: "/boot",
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\nunknown or invalid filesystem type for mountpoint \"/boot\": zfs",
		},
		"unhappy-plain-badfstype-efi": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							FSType:     "ext4",
							Mountpoint: "/",
						},
					},
					{
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							FSType:     "ext4",
							Mountpoint: "/home",
						},
					},
					{
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							FSType:     "vfat",
							Mountpoint: "/boot",
						},
					},
					{
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							FSType:     "ext4",
							Mountpoint: "/boot/efi",
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\nunsupported filesystem type for \"/boot\": vfat\nunsupported filesystem type for \"/boot/efi\": ext4",
		},
		"unhappy-plain-btrfstype": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "btrfs",
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							Mountpoint: "/",
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\n\"mountpoint\" is not supported for btrfs volumes (only subvolumes can have mountpoints)",
		},
		"unhappy-plain+btrfs-dupes": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "plain",
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							Mountpoint: "/data",
							FSType:     "xfs",
						},
					},
					{
						Type: "btrfs",
						BtrfsVolumeCustomization: blueprint.BtrfsVolumeCustomization{
							Subvolumes: []blueprint.BtrfsSubvolumeCustomization{
								{
									Name:       "root",
									Mountpoint: "/",
								},
								{
									Name:       "home",
									Mountpoint: "/home",
								},
								{
									Name:       "data",
									Mountpoint: "/data",
								},
							},
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\nduplicate mountpoint \"/data\" in partitioning customizations",
		},
		"unhappy-plain+lvm-dupes": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "plain",
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							Mountpoint: "/dupydupe",
							FSType:     "ext4",
						},
					},
					{
						Type: "plain",
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							Mountpoint: "/data",
							FSType:     "ext4",
						},
					},
					{
						Type: "lvm",
						VGCustomization: blueprint.VGCustomization{
							LogicalVolumes: []blueprint.LVCustomization{
								{
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
										FSType:     "ext4",
										Mountpoint: "/",
									},
								},
								{
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
										FSType:     "ext4",
										Mountpoint: "/home",
									},
								},
								{
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
										FSType:     "ext4",
										Mountpoint: "/dupydupe",
									},
								},
							},
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\nduplicate mountpoint \"/dupydupe\" in partitioning customizations",
		},
		"unhappy-emptymp": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							FSType: "ext4",
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\nmountpoint is empty",
		},
		"unhappy-relativemountpoint": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							FSType:     "xfs",
							Mountpoint: "i-am-not-absolute",
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\nmountpoint \"i-am-not-absolute\" is not an absolute path",
		},
		"unhappy-badmp": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							FSType:     "vfat",
							Mountpoint: "/home/../root",
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\nmountpoint \"/home/../root\" is not a canonical path (did you mean \"/root\"?)",
		},
		"unhappy-emptymp-btrfs": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "btrfs",
						BtrfsVolumeCustomization: blueprint.BtrfsVolumeCustomization{
							Subvolumes: []blueprint.BtrfsSubvolumeCustomization{
								{
									Name:       "test",
									Mountpoint: "/test",
								},
								{
									Name:       "test2",
									Mountpoint: "",
								},
							},
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\ninvalid btrfs subvolume customization: mountpoint is empty",
		},
		"unhappy-relativemountpoint-btrfs": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "btrfs",
						BtrfsVolumeCustomization: blueprint.BtrfsVolumeCustomization{
							Subvolumes: []blueprint.BtrfsSubvolumeCustomization{
								{
									Name:       "blorps",
									Mountpoint: "blorpsmp",
								},
							},
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\ninvalid btrfs subvolume customization: mountpoint \"blorpsmp\" is not an absolute path",
		},
		"unhappy-badmp-btrfs": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "btrfs",
						BtrfsVolumeCustomization: blueprint.BtrfsVolumeCustomization{
							Subvolumes: []blueprint.BtrfsSubvolumeCustomization{
								{
									Name:       "borkage",
									Mountpoint: "/home//bork",
								},
							},
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\ninvalid btrfs subvolume customization: mountpoint \"/home//bork\" is not a canonical path (did you mean \"/home/bork\"?)",
		},
		"unhappy-emptymp-lvm": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "lvm",
						VGCustomization: blueprint.VGCustomization{
							LogicalVolumes: []blueprint.LVCustomization{
								{
									Name: "testlv",
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{

										FSType:     "ext4",
										Mountpoint: "/stuff",
									},
								},
								{
									Name: "testlv2",
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
										FSType:     "ext4",
										Mountpoint: "",
									},
								},
							},
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\ninvalid logical volume customization: mountpoint is empty",
		},
		"unhappy-relativemountpoint-lvm": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "lvm",
						VGCustomization: blueprint.VGCustomization{
							LogicalVolumes: []blueprint.LVCustomization{
								{
									Name: "testlv",
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
										FSType:     "xfs",
										Mountpoint: "/stuff",
									},
								},
								{
									Name: "testlv2",
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
										Mountpoint: "i/like/relative/paths",
									},
								},
							},
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\ninvalid logical volume customization: mountpoint \"i/like/relative/paths\" is not an absolute path",
		},
		"unhappy-badmp-lvm": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "lvm",
						VGCustomization: blueprint.VGCustomization{
							LogicalVolumes: []blueprint.LVCustomization{
								{
									Name: "testlv",
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
										Mountpoint: "/../../../what/",
									},
								},
								{
									Name: "testlv2",
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
										Mountpoint: "/test",
									},
								},
							},
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\ninvalid logical volume customization: mountpoint \"/../../../what/\" is not a canonical path (did you mean \"/what\"?)",
		},
		"unhappy-dupesubvolname": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "btrfs",
						BtrfsVolumeCustomization: blueprint.BtrfsVolumeCustomization{
							Subvolumes: []blueprint.BtrfsSubvolumeCustomization{
								{
									Name:       "root",
									Mountpoint: "/",
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
			expectedMsg: "invalid partitioning customizations:\nduplicate btrfs subvolume name \"root\" in partitioning customizations",
		},
		"unhappy-dupelvname": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "lvm",
						VGCustomization: blueprint.VGCustomization{
							LogicalVolumes: []blueprint.LVCustomization{
								{
									Name: "testlv",
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
										FSType:     "ext4",
										Mountpoint: "/stuff",
									},
								},
								{
									Name: "testlv",
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
										FSType:     "ext4",
										Mountpoint: "/stuff2",
									},
								},
							},
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\nduplicate LVM logical volume name \"testlv\" in volume group \"\" in partitioning customizations",
		},
		"unhappy-vg-with-subvols": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "lvm",
						BtrfsVolumeCustomization: blueprint.BtrfsVolumeCustomization{
							Subvolumes: []blueprint.BtrfsSubvolumeCustomization{{}},
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\nsubvolumes defined for LVM volume group (partition type \"lvm\")",
		},
		"unhappy-vg-with-label": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "lvm",
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							Label: "volume-group",
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\nlabel \"volume-group\" defined for LVM volume group (partition type \"lvm\")",
		},
		"unhappy-dupevgname": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "lvm",
						VGCustomization: blueprint.VGCustomization{
							Name: "testvg",
							LogicalVolumes: []blueprint.LVCustomization{
								{
									Name: "testlv",
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
										FSType:     "ext4",
										Mountpoint: "/test",
									},
								},
							},
						},
					},
					{
						Type: "lvm",
						VGCustomization: blueprint.VGCustomization{
							Name: "testvg",
							LogicalVolumes: []blueprint.LVCustomization{
								{
									Name: "testlv",
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
										FSType:     "ext4",
										Mountpoint: "/stuff",
									},
								},
							},
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\nduplicate LVM volume group name \"testvg\" in partitioning customizations",
		},
		"unhappy-emptyname-btrfs": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "btrfs",
						BtrfsVolumeCustomization: blueprint.BtrfsVolumeCustomization{
							Subvolumes: []blueprint.BtrfsSubvolumeCustomization{
								{
									Name:       "test",
									Mountpoint: "/test",
								},
								{
									Name:       "",
									Mountpoint: "/test2",
								},
							},
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\nbtrfs subvolume with empty name in partitioning customizations",
		},
		"unhappy-emptysubvols-btrfs": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "btrfs",
						BtrfsVolumeCustomization: blueprint.BtrfsVolumeCustomization{
							Subvolumes: []blueprint.BtrfsSubvolumeCustomization{},
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\nbtrfs volume requires subvolumes",
		},
		"unhappy-btrfs-with-lvs": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "btrfs",
						BtrfsVolumeCustomization: blueprint.BtrfsVolumeCustomization{
							Subvolumes: []blueprint.BtrfsSubvolumeCustomization{
								{
									Name:       "test",
									Mountpoint: "/test2",
								},
							},
						},
						VGCustomization: blueprint.VGCustomization{
							LogicalVolumes: []blueprint.LVCustomization{{}},
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\nLVM logical volumes defined for btrfs volume (partition type \"btrfs\")",
		},
		"boot-on-lvm": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "lvm",
						VGCustomization: blueprint.VGCustomization{
							LogicalVolumes: []blueprint.LVCustomization{
								{
									Name: "bewt",
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
										Mountpoint: "/boot",
									},
								},
								{
									Name: "testlv",
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
										Mountpoint: "/stuff2",
									},
								},
							},
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\ninvalid mountpoint \"/boot\" for logical volume",
		},
		"bootefi-on-lvm": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "lvm",
						VGCustomization: blueprint.VGCustomization{
							LogicalVolumes: []blueprint.LVCustomization{
								{
									Name: "bewtefi",
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
										Mountpoint: "/boot/efi",
									},
								},
							},
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\ninvalid mountpoint \"/boot/efi\" for logical volume",
		},
		"boot-on-btrfs": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "btrfs",
						BtrfsVolumeCustomization: blueprint.BtrfsVolumeCustomization{
							Subvolumes: []blueprint.BtrfsSubvolumeCustomization{
								{
									Name:       "test",
									Mountpoint: "/test",
								},
								{
									Name:       "bootbootboot",
									Mountpoint: "/boot",
								},
							},
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\ninvalid mountpoint \"/boot\" for btrfs subvolume",
		},
		"bootefi-on-btrfs": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "btrfs",
						BtrfsVolumeCustomization: blueprint.BtrfsVolumeCustomization{
							Subvolumes: []blueprint.BtrfsSubvolumeCustomization{
								{
									Name:       "test",
									Mountpoint: "/test",
								},
								{
									Name:       "esp",
									Mountpoint: "/boot/efi",
								},
							},
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\ninvalid mountpoint \"/boot/efi\" for btrfs subvolume",
		},
		"unhappy-btrfs-on-lvm": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "lvm",
						VGCustomization: blueprint.VGCustomization{
							LogicalVolumes: []blueprint.LVCustomization{
								{
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
										FSType:     "btrfs",
										Mountpoint: "/var/log",
									},
								},
							},
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\nunknown or invalid filesystem type for logical volume with mountpoint \"/var/log\": btrfs",
		},
		"unhappy-lv-notype": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "lvm",
						VGCustomization: blueprint.VGCustomization{
							LogicalVolumes: []blueprint.LVCustomization{
								{
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
										Mountpoint: "/var/log",
									},
								},
							},
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\nunknown or invalid filesystem type for logical volume with mountpoint \"/var/log\": ",
		},
		"unhappy-bad-part-type": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "what",
						VGCustomization: blueprint.VGCustomization{
							LogicalVolumes: []blueprint.LVCustomization{
								{
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
										Mountpoint: "/var/log",
									},
								},
							},
						},
					},
				},
			},
			expectedMsg: "invalid partitioning customizations:\nunknown partition type: what",
		},
	}

	for name := range testCases {
		tc := testCases[name]
		t.Run(name, func(t *testing.T) {
			err := tc.partitioning.Validate()
			if tc.expectedMsg != "" {
				assert.EqualError(t, err, tc.expectedMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPartitioningLayoutConstraints(t *testing.T) {
	type testCase struct {
		partitioning *blueprint.DiskCustomization
		expectedMsg  string
	}

	testCases := map[string]testCase{
		"unhappy-btrfs+lvm": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "ext4",
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							Mountpoint: "/data",
						},
					},
					{
						Type: "btrfs",
						BtrfsVolumeCustomization: blueprint.BtrfsVolumeCustomization{
							Subvolumes: []blueprint.BtrfsSubvolumeCustomization{
								{
									Mountpoint: "/backup",
								},
							},
						},
					},
					{
						Type: "lvm",
						VGCustomization: blueprint.VGCustomization{
							LogicalVolumes: []blueprint.LVCustomization{
								{
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{Mountpoint: "/"},
								},
							},
						},
					},
				},
			},
			expectedMsg: `btrfs and lvm partitioning cannot be combined`,
		},
		"unhappy-multibtrfs": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "xfs",
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							Mountpoint: "/data",
						},
					},
					{
						Type: "btrfs",
						BtrfsVolumeCustomization: blueprint.BtrfsVolumeCustomization{
							Subvolumes: []blueprint.BtrfsSubvolumeCustomization{
								{
									Name:       "root",
									Mountpoint: "/",
								},
							},
						},
					},
					{
						Type: "btrfs",
						BtrfsVolumeCustomization: blueprint.BtrfsVolumeCustomization{
							Subvolumes: []blueprint.BtrfsSubvolumeCustomization{
								{
									Name:       "home",
									Mountpoint: "/home",
								},
							},
						},
					},
				},
			},
			expectedMsg: `multiple btrfs volumes are not yet supported`,
		},
		"unhappy-multivg": {
			partitioning: &blueprint.DiskCustomization{
				Partitions: []blueprint.PartitionCustomization{
					{
						Type: "xfs",
						FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
							Mountpoint: "/data",
						},
					},
					{
						Type: "lvm",
						VGCustomization: blueprint.VGCustomization{
							LogicalVolumes: []blueprint.LVCustomization{
								{
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{Mountpoint: "/"},
								},
							},
						},
					},
					{
						Type: "lvm",
						VGCustomization: blueprint.VGCustomization{
							LogicalVolumes: []blueprint.LVCustomization{
								{
									FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{Mountpoint: "/var/log"},
								},
							},
						},
					},
				},
			},
			expectedMsg: `multiple LVM volume groups are not yet supported`,
		},
	}

	for name := range testCases {
		tc := testCases[name]
		t.Run(name, func(t *testing.T) {
			err := tc.partitioning.ValidateLayoutConstraints()
			if tc.expectedMsg != "" {
				assert.EqualError(t, err, tc.expectedMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCheckDiskMountpointsPolicy(t *testing.T) {
	strict := pathpolicy.NewPathPolicies(map[string]pathpolicy.PathPolicy{
		"/": {Exact: true},
	})

	noEtc := pathpolicy.NewPathPolicies(map[string]pathpolicy.PathPolicy{
		"/etc": {Deny: true},
	})

	disk := blueprint.DiskCustomization{
		Partitions: []blueprint.PartitionCustomization{
			{
				FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
					Mountpoint: "/some/stuff",
				},
			},
			{
				BtrfsVolumeCustomization: blueprint.BtrfsVolumeCustomization{
					Subvolumes: []blueprint.BtrfsSubvolumeCustomization{
						{
							Mountpoint: "/data/",
						},
						{
							Mountpoint: "/scratch",
						},
					},
				},
			},
			{
				VGCustomization: blueprint.VGCustomization{
					LogicalVolumes: []blueprint.LVCustomization{
						{
							FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
								Mountpoint: "/logicalvolumes/a",
							},
						},
						{
							FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
								Mountpoint: "/logicalvolumes/b",
							},
						},
						{
							FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
								Mountpoint: "/etc",
							},
						},
					},
				},
			},
		},
	}

	strictErr := `The following errors occurred while setting up custom mountpoints:
path "/some/stuff" is not allowed
path "/data/" must be canonical
path "/scratch" is not allowed
path "/logicalvolumes/a" is not allowed
path "/logicalvolumes/b" is not allowed
path "/etc" is not allowed`
	err := blueprint.CheckDiskMountpointsPolicy(&disk, strict)
	assert.EqualError(t, err, strictErr)

	noEtcErr := `The following errors occurred while setting up custom mountpoints:
path "/data/" must be canonical
path "/etc" is not allowed`
	err = blueprint.CheckDiskMountpointsPolicy(&disk, noEtc)
	assert.EqualError(t, err, noEtcErr)
}

func TestPartitionCustomizationUnmarshalJSON(t *testing.T) {
	type testCase struct {
		input    string
		expected *blueprint.PartitionCustomization
		errorMsg string
	}

	testCases := map[string]testCase{
		"nothing": {
			input: "{}",
			expected: &blueprint.PartitionCustomization{
				Type:    "plain",
				MinSize: 0,
				FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
					Mountpoint: "",
					Label:      "",
					FSType:     "",
				},
			},
		},
		"plain": {
			input: `{
				"type": "plain",
				"minsize": "1 GiB",
				"mountpoint": "/",
				"label": "root",
				"fs_type": "xfs"
			}`,
			expected: &blueprint.PartitionCustomization{
				Type:    "plain",
				MinSize: 1 * datasizes.GiB,
				FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
					Mountpoint: "/",
					Label:      "root",
					FSType:     "xfs",
				},
			},
		},
		"plain-with-int": {
			input: `{
				"type": "plain",
				"minsize": 1073741824,
				"mountpoint": "/",
				"label": "root",
				"fs_type": "xfs"
			}`,
			expected: &blueprint.PartitionCustomization{
				Type:    "plain",
				MinSize: 1 * datasizes.GiB,
				FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
					Mountpoint: "/",
					Label:      "root",
					FSType:     "xfs",
				},
			},
		},
		"btrfs": {
			input: `{
				"type": "btrfs",
				"minsize": "10 GiB",
				"subvolumes": [
					{
						"name": "subvols/root",
						"mountpoint": "/"
					},
					{
						"name": "subvols/data",
						"mountpoint": "/data"
					}
				]
			}`,
			expected: &blueprint.PartitionCustomization{
				Type:    "btrfs",
				MinSize: 10 * datasizes.GiB,
				BtrfsVolumeCustomization: blueprint.BtrfsVolumeCustomization{
					Subvolumes: []blueprint.BtrfsSubvolumeCustomization{
						{
							Name:       "subvols/root",
							Mountpoint: "/",
						},
						{
							Name:       "subvols/data",
							Mountpoint: "/data",
						},
					},
				},
			},
		},
		"btrfs-with-int": {
			input: `{
				"type": "btrfs",
				"minsize": 10737418240,
				"subvolumes": [
					{
						"name": "subvols/root",
						"mountpoint": "/"
					},
					{
						"name": "subvols/data",
						"mountpoint": "/data"
					}
				]
			}`,
			expected: &blueprint.PartitionCustomization{
				Type:    "btrfs",
				MinSize: 10 * datasizes.GiB,
				BtrfsVolumeCustomization: blueprint.BtrfsVolumeCustomization{
					Subvolumes: []blueprint.BtrfsSubvolumeCustomization{
						{
							Name:       "subvols/root",
							Mountpoint: "/",
						},
						{
							Name:       "subvols/data",
							Mountpoint: "/data",
						},
					},
				},
			},
		},
		"lvm": {
			input: `{
				"type": "lvm",
				"name": "myvg",
				"minsize": "99 GiB",
				"logical_volumes": [
					{
						"name": "homelv",
						"mountpoint": "/home",
						"label": "home",
						"fs_type": "ext4",
						"minsize": "2 GiB"
					},
					{
						"name": "loglv",
						"mountpoint": "/var/log",
						"label": "log",
						"fs_type": "xfs",
						"minsize": "3 GiB"
					}
				]
			}`,
			expected: &blueprint.PartitionCustomization{
				Type:    "lvm",
				MinSize: 99 * datasizes.GiB,
				VGCustomization: blueprint.VGCustomization{
					Name: "myvg",
					LogicalVolumes: []blueprint.LVCustomization{
						{
							Name:    "homelv",
							MinSize: 2 * datasizes.GiB,
							FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
								Mountpoint: "/home",
								Label:      "home",
								FSType:     "ext4",
							},
						},
						{
							Name:    "loglv",
							MinSize: 3 * datasizes.GiB,
							FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
								Mountpoint: "/var/log",
								Label:      "log",
								FSType:     "xfs",
							},
						},
					},
				},
			},
		},
		"lvm-with-int": {
			input: `{
				"type": "lvm",
				"name": "myvg",
				"minsize": 106300440576,
				"logical_volumes": [
					{
						"name": "homelv",
						"mountpoint": "/home",
						"label": "home",
						"fs_type": "ext4",
						"minsize": 2147483648
					},
					{
						"name": "loglv",
						"mountpoint": "/var/log",
						"label": "log",
						"fs_type": "xfs",
						"minsize": 3221225472
					}
				]
			}`,
			expected: &blueprint.PartitionCustomization{
				Type:    "lvm",
				MinSize: 99 * datasizes.GiB,
				VGCustomization: blueprint.VGCustomization{
					Name: "myvg",
					LogicalVolumes: []blueprint.LVCustomization{
						{
							Name:    "homelv",
							MinSize: 2 * datasizes.GiB,
							FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
								Mountpoint: "/home",
								Label:      "home",
								FSType:     "ext4",
							},
						},
						{
							Name:    "loglv",
							MinSize: 3 * datasizes.GiB,
							FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
								Mountpoint: "/var/log",
								Label:      "log",
								FSType:     "xfs",
							},
						},
					},
				},
			},
		},
		"bad-type": {
			input:    `{"type":"not-a-partition-type"}`,
			errorMsg: "JSON unmarshal: unknown partition type: not-a-partition-type",
		},
		"number": {
			input:    `{"type":5}`,
			errorMsg: "JSON unmarshal: json: cannot unmarshal number into Go struct field .type of type string",
		},
		"negative-size": {
			input: `{
				"minsize": -10,
				"mountpoint": "/",
				"fs_type": "xfs"
			}`,
			errorMsg: "JSON unmarshal: error decoding minsize for partition: cannot be negative",
		},
		"wrong-type/btrfs-with-lvm": {
			input: `{
				"type": "btrfs",
				"name": "myvg",
				"logical_volumes": [
					{
						"name": "homelv",
						"mountpoint": "/home",
						"label": "home",
						"fs_type": "ext4"
					},
					{
						"name": "loglv",
						"mountpoint": "/var/log",
						"label": "log",
						"fs_type": "xfs"
					}
				]
			}`,
			errorMsg: `JSON unmarshal: error decoding partition with type "btrfs": json: unknown field "name"`,
		},
		"wrong-type/plain-with-lvm": {
			input: `{
				"type": "plain",
				"name": "myvg",
				"logical_volumes": [
					{
						"name": "loglv",
						"mountpoint": "/var/log",
						"label": "log",
						"fs_type": "xfs"
					}
				]
			}`,
			errorMsg: `JSON unmarshal: error decoding partition with type "plain": json: unknown field "name"`,
		},
		"wrong-type/lvm-with-btrfs": {
			input: `{
				"type": "lvm",
				"minsize": "10 GiB",
				"subvolumes": [
					{
						"name": "subvols/data",
						"mountpoint": "/data"
					}
				]
			}`,
			errorMsg: `JSON unmarshal: error decoding partition with type "lvm": json: unknown field "subvolumes"`,
		},
		"wrong-type/plain-with-btrfs": {
			input: `{
				"type": "plain",
				"minsize": "10 GiB",
				"subvolumes": [
					{
						"name": "subvols/data",
						"mountpoint": "/data"
					}
				]
			}`,
			errorMsg: `JSON unmarshal: error decoding partition with type "plain": json: unknown field "subvolumes"`,
		},
	}

	for name := range testCases {
		tc := testCases[name]
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			var pc blueprint.PartitionCustomization

			err := json.Unmarshal([]byte(tc.input), &pc)
			if tc.errorMsg == "" {
				assert.NoError(err)
				assert.Equal(tc.expected, &pc)
			} else {
				assert.EqualError(err, tc.errorMsg)
			}
		})
	}
}

func TestPartitionCustomizationUnmarshalTOML(t *testing.T) {
	type testCase struct {
		input    string
		expected *blueprint.PartitionCustomization
		errorMsg string
	}

	testCases := map[string]testCase{
		"nothing": {
			input: "",
			expected: &blueprint.PartitionCustomization{
				Type:    "plain",
				MinSize: 0,
				FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
					Mountpoint: "",
					Label:      "",
					FSType:     "",
				},
			},
		},
		"plain": {
			input: `type = "plain"
					minsize = "1 GiB"
					mountpoint = "/"
					label = "root"
					fs_type = "xfs"`,
			expected: &blueprint.PartitionCustomization{
				Type:    "plain",
				MinSize: 1 * datasizes.GiB,
				FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
					Mountpoint: "/",
					Label:      "root",
					FSType:     "xfs",
				},
			},
		},
		"plain-with-int": {
			input: `type = "plain"
					minsize = 1073741824
					mountpoint = "/"
					label = "root"
					fs_type = "xfs"`,
			expected: &blueprint.PartitionCustomization{
				Type:    "plain",
				MinSize: 1 * datasizes.GiB,
				FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
					Mountpoint: "/",
					Label:      "root",
					FSType:     "xfs",
				},
			},
		},
		"btrfs": {
			input: `type = "btrfs"
					minsize = "10 GiB"

					[[subvolumes]]
					name = "subvols/root"
					mountpoint = "/"

					[[subvolumes]]
					name = "subvols/data"
					mountpoint = "/data"
					`,
			expected: &blueprint.PartitionCustomization{
				Type:    "btrfs",
				MinSize: 10 * datasizes.GiB,
				BtrfsVolumeCustomization: blueprint.BtrfsVolumeCustomization{
					Subvolumes: []blueprint.BtrfsSubvolumeCustomization{
						{
							Name:       "subvols/root",
							Mountpoint: "/",
						},
						{
							Name:       "subvols/data",
							Mountpoint: "/data",
						},
					},
				},
			},
		},
		"btrfs-with-int": {
			input: `type = "btrfs"
					minsize = 10737418240

					[[subvolumes]]
					name = "subvols/root"
					mountpoint = "/"

					[[subvolumes]]
					name = "subvols/data"
					mountpoint = "/data"
					`,
			expected: &blueprint.PartitionCustomization{
				Type:    "btrfs",
				MinSize: 10 * datasizes.GiB,
				BtrfsVolumeCustomization: blueprint.BtrfsVolumeCustomization{
					Subvolumes: []blueprint.BtrfsSubvolumeCustomization{
						{
							Name:       "subvols/root",
							Mountpoint: "/",
						},
						{
							Name:       "subvols/data",
							Mountpoint: "/data",
						},
					},
				},
			},
		},
		"lvm": {
			input: `type = "lvm"
					name = "myvg"
					minsize = "99 GiB"

					[[logical_volumes]]
					name = "homelv"
					mountpoint = "/home"
					label = "home"
					fs_type = "ext4"
					minsize = "2 GiB"

					[[logical_volumes]]
					name = "loglv"
					mountpoint = "/var/log"
					label = "log"
					fs_type = "xfs"
					minsize = "3 GiB"
					`,
			expected: &blueprint.PartitionCustomization{
				Type:    "lvm",
				MinSize: 99 * datasizes.GiB,
				VGCustomization: blueprint.VGCustomization{
					Name: "myvg",
					LogicalVolumes: []blueprint.LVCustomization{
						{
							Name:    "homelv",
							MinSize: 2 * datasizes.GiB,
							FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
								Mountpoint: "/home",
								Label:      "home",
								FSType:     "ext4",
							},
						},
						{
							Name:    "loglv",
							MinSize: 3 * datasizes.GiB,
							FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
								Mountpoint: "/var/log",
								Label:      "log",
								FSType:     "xfs",
							},
						},
					},
				},
			},
		},
		"lvm-with-int": {
			input: `type = "lvm"
					name = "myvg"
					minsize = 106300440576

					[[logical_volumes]]
					name = "homelv"
					mountpoint = "/home"
					label = "home"
					fs_type = "ext4"
					minsize = 2147483648

					[[logical_volumes]]
					name = "loglv"
					mountpoint = "/var/log"
					label = "log"
					fs_type = "xfs"
					minsize = 3221225472
					`,
			expected: &blueprint.PartitionCustomization{
				Type:    "lvm",
				MinSize: 99 * datasizes.GiB,
				VGCustomization: blueprint.VGCustomization{
					Name: "myvg",
					LogicalVolumes: []blueprint.LVCustomization{
						{
							Name:    "homelv",
							MinSize: 2 * datasizes.GiB,
							FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
								Mountpoint: "/home",
								Label:      "home",
								FSType:     "ext4",
							},
						},
						{
							Name:    "loglv",
							MinSize: 3 * datasizes.GiB,
							FilesystemTypedCustomization: blueprint.FilesystemTypedCustomization{
								Mountpoint: "/var/log",
								Label:      "log",
								FSType:     "xfs",
							},
						},
					},
				},
			},
		},
		"bad-type": {
			input:    `type = "not-a-partition-type"`,
			errorMsg: "toml: line 0: TOML unmarshal: unknown partition type: not-a-partition-type",
		},
		"number": {
			input:    `type = 5`,
			errorMsg: `toml: line 0: TOML unmarshal: type must be a string, got "5" of type int64`,
		},
		"negative-size": {
			input: `minsize = -10
					mountpoint = "/"
					fs_type = "xfs"
					`,
			errorMsg: "toml: line 0: TOML unmarshal: error decoding minsize for partition: cannot be negative",
		},
		"wrong-type/btrfs-with-lvm": {
			input: `type = "btrfs"
					name = "myvg"

					[[logical_volumes]]
					name = "homelv"
					mountpoint = "/home"
					label = "home"
					fs_type = "ext4"

					[[logical_volumes]]
					name = "loglv"
					mountpoint = "/var/log"
					label = "log"
					fs_type = "xfs"
					`,
			errorMsg: `toml: line 0: TOML unmarshal: error decoding partition with type "btrfs": json: unknown field "logical_volumes"`,
		},
		"wrong-type/plain-with-lvm": {
			input: `type = "plain"
					name = "myvg"

					[[logical_volumes]]
					name = "homelv"
					mountpoint = "/home"
					label = "home"
					fs_type = "ext4"

					[[logical_volumes]]
					name = "loglv"
					mountpoint = "/var/log"
					label = "log"
					fs_type = "xfs"
					`,
			errorMsg: `toml: line 0: TOML unmarshal: error decoding partition with type "plain": json: unknown field "logical_volumes"`,
		},
		"wrong-type/lvm-with-btrfs": {
			input: `type = "lvm"
					minsize = "10 GiB"

					[[subvolumes]]
					name = "subvols/root"
					mountpoint = "/"

					[[subvolumes]]
					name = "subvols/data"
					mountpoint = "/data"
					`,
			errorMsg: `toml: line 0: TOML unmarshal: error decoding partition with type "lvm": json: unknown field "subvolumes"`,
		},
		"wrong-type/plain-with-btrfs": {
			input: `type = "plain"
					minsize = "10 GiB"

					[[subvolumes]]
					name = "subvols/root"
					mountpoint = "/"

					[[subvolumes]]
					name = "subvols/data"
					mountpoint = "/data"
					`,
			errorMsg: `toml: line 0: TOML unmarshal: error decoding partition with type "plain": json: unknown field "subvolumes"`,
		},
	}

	for name := range testCases {
		tc := testCases[name]
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			var pc blueprint.PartitionCustomization

			err := toml.Unmarshal([]byte(tc.input), &pc)
			if tc.errorMsg == "" {
				assert.NoError(err)
				assert.Equal(tc.expected, &pc)
			} else {
				assert.EqualError(err, tc.errorMsg)
			}
		})
	}
}
