package blueprint_test

import (
	"testing"

	"github.com/osbuild/images/pkg/blueprint"
	"github.com/stretchr/testify/assert"
)

func TestPartitioningValidation(t *testing.T) {
	type testCase struct {
		partitioning *blueprint.PartitioningCustomization
		expectedMsg  string
	}

	testCases := map[string]testCase{
		"null": {
			partitioning: nil,
			expectedMsg:  "",
		},
		"happy-plain": {
			partitioning: &blueprint.PartitioningCustomization{
				Plain: &blueprint.PlainFilesystemCustomization{
					Filesystems: []blueprint.FilesystemCustomization{
						{
							Mountpoint: "/data",
						},
					},
				},
			},
			expectedMsg: "",
		},
		"happy-plain+btrfs": {
			partitioning: &blueprint.PartitioningCustomization{
				Plain: &blueprint.PlainFilesystemCustomization{
					Filesystems: []blueprint.FilesystemCustomization{
						{
							Mountpoint: "/data",
						},
					},
				},
				Btrfs: &blueprint.BtrfsCustomization{
					Volumes: []blueprint.BtrfsVolumeCustomization{
						{
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
			partitioning: &blueprint.PartitioningCustomization{
				Plain: &blueprint.PlainFilesystemCustomization{
					Filesystems: []blueprint.FilesystemCustomization{
						{
							Mountpoint: "/data",
						},
					},
				},
				LVM: &blueprint.LVMCustomization{
					VolumeGroups: []blueprint.VGCustomization{
						{
							LogicalVolumes: []blueprint.LVCustomization{
								{
									FilesystemCustomization: blueprint.FilesystemCustomization{Mountpoint: "/"},
								},
							},
						},
					},
				},
			},
			expectedMsg: "",
		},
		"happy-plain-with-boot-and-efi-nofs": {
			partitioning: &blueprint.PartitioningCustomization{
				Plain: &blueprint.PlainFilesystemCustomization{
					Filesystems: []blueprint.FilesystemCustomization{
						{
							Mountpoint: "/data",
						},
						{
							Mountpoint: "/",
						},
						{
							Mountpoint: "/home",
						},
						{
							Mountpoint: "/boot",
						},
						{
							Mountpoint: "/boot/efi",
						},
					},
				},
			},
			expectedMsg: "",
		},
		"unhappy-btrfs+lvm": {
			partitioning: &blueprint.PartitioningCustomization{
				Plain: &blueprint.PlainFilesystemCustomization{
					Filesystems: []blueprint.FilesystemCustomization{
						{
							Mountpoint: "/data",
						},
					},
				},
				Btrfs: &blueprint.BtrfsCustomization{
					Volumes: []blueprint.BtrfsVolumeCustomization{
						{
							Subvolumes: []blueprint.BtrfsSubvolumeCustomization{
								{
									Mountpoint: "/backup",
								},
							},
						},
					},
				},
				LVM: &blueprint.LVMCustomization{
					VolumeGroups: []blueprint.VGCustomization{
						{
							LogicalVolumes: []blueprint.LVCustomization{
								{
									FilesystemCustomization: blueprint.FilesystemCustomization{Mountpoint: "/"},
								},
							},
						},
					},
				},
			},
			expectedMsg: `btrfs and lvm partitioning cannot be combined`,
		},
		"unhappy-plain-dupes": {
			partitioning: &blueprint.PartitioningCustomization{
				Plain: &blueprint.PlainFilesystemCustomization{
					Filesystems: []blueprint.FilesystemCustomization{
						{
							Mountpoint: "/data",
						},
						{
							Mountpoint: "/",
						},
						{
							Mountpoint: "/home",
						},
						{
							Mountpoint: "/data",
						},
					},
				},
			},
			expectedMsg: `duplicate mountpoint "/data" in partitioning customizations`,
		},
		"unhappy-plain-badfstype": {
			partitioning: &blueprint.PartitioningCustomization{
				Plain: &blueprint.PlainFilesystemCustomization{
					Filesystems: []blueprint.FilesystemCustomization{
						{
							Mountpoint: "/",
						},
						{
							Mountpoint: "/home",
						},
						{
							Mountpoint: "/boot",
							Type:       "zfs",
						},
						{
							Mountpoint: "/data",
						},
					},
				},
			},
			expectedMsg: `invalid plain filesystem customization: unsupported filesystem type for "/boot": zfs`,
		},
		"unhappy-plain-badfstype-efi": {
			partitioning: &blueprint.PartitioningCustomization{
				Plain: &blueprint.PlainFilesystemCustomization{
					Filesystems: []blueprint.FilesystemCustomization{
						{
							Mountpoint: "/",
						},
						{
							Mountpoint: "/home",
						},
						{
							Mountpoint: "/boot/efi",
							Type:       "ext4",
						},
						{
							Mountpoint: "/data",
						},
					},
				},
			},
			expectedMsg: `invalid plain filesystem customization: unsupported filesystem type for "/boot/efi": ext4`,
		},
		"unhappy-plain-btrfstype": {
			partitioning: &blueprint.PartitioningCustomization{
				Plain: &blueprint.PlainFilesystemCustomization{
					Filesystems: []blueprint.FilesystemCustomization{
						{
							Mountpoint: "/",
							Type:       "btrfs",
						},
						{
							Mountpoint: "/home",
						},
					},
				},
			},
			expectedMsg: `btrfs filesystem defined under plain partitioning customization: please use the "btrfs" customization to define btrfs volumes and subvolumes`,
		},
		"unhappy-plain+btrfs-dupes": {
			partitioning: &blueprint.PartitioningCustomization{
				Plain: &blueprint.PlainFilesystemCustomization{
					Filesystems: []blueprint.FilesystemCustomization{
						{
							Mountpoint: "/data",
						},
					},
				},
				Btrfs: &blueprint.BtrfsCustomization{
					Volumes: []blueprint.BtrfsVolumeCustomization{
						{
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
			expectedMsg: `duplicate mountpoint "/data" in partitioning customizations`,
		},
		"unhappy-plain+lvm-dupes": {
			partitioning: &blueprint.PartitioningCustomization{
				Plain: &blueprint.PlainFilesystemCustomization{
					Filesystems: []blueprint.FilesystemCustomization{
						{
							Mountpoint: "/dupydupe",
						},
						{
							Mountpoint: "/data",
						},
					},
				},
				LVM: &blueprint.LVMCustomization{
					VolumeGroups: []blueprint.VGCustomization{
						{
							LogicalVolumes: []blueprint.LVCustomization{
								{
									FilesystemCustomization: blueprint.FilesystemCustomization{Mountpoint: "/"},
								},
								{
									FilesystemCustomization: blueprint.FilesystemCustomization{Mountpoint: "/home"},
								},
								{
									FilesystemCustomization: blueprint.FilesystemCustomization{Mountpoint: "/dupydupe"},
								},
							},
						},
					},
				},
			},
			expectedMsg: `duplicate mountpoint "/dupydupe" in partitioning customizations`,
		},
		"unhappy-multibtrfs": {
			partitioning: &blueprint.PartitioningCustomization{
				Plain: &blueprint.PlainFilesystemCustomization{
					Filesystems: []blueprint.FilesystemCustomization{
						{
							Mountpoint: "/data",
						},
					},
				},
				Btrfs: &blueprint.BtrfsCustomization{
					Volumes: []blueprint.BtrfsVolumeCustomization{
						{
							Subvolumes: []blueprint.BtrfsSubvolumeCustomization{
								{
									Name:       "root",
									Mountpoint: "/",
								},
							},
						},
						{
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
			partitioning: &blueprint.PartitioningCustomization{
				Plain: &blueprint.PlainFilesystemCustomization{
					Filesystems: []blueprint.FilesystemCustomization{
						{
							Mountpoint: "/data",
						},
					},
				},
				LVM: &blueprint.LVMCustomization{
					VolumeGroups: []blueprint.VGCustomization{
						{
							LogicalVolumes: []blueprint.LVCustomization{
								{
									FilesystemCustomization: blueprint.FilesystemCustomization{Mountpoint: "/"},
								},
							},
						},
						{
							LogicalVolumes: []blueprint.LVCustomization{
								{
									FilesystemCustomization: blueprint.FilesystemCustomization{Mountpoint: "/var/log"},
								},
							},
						},
					},
				},
			},
			expectedMsg: `multiple LVM volume groups are not yet supported`,
		},
		"unhappy-emptymp": {
			partitioning: &blueprint.PartitioningCustomization{
				Plain: &blueprint.PlainFilesystemCustomization{
					Filesystems: []blueprint.FilesystemCustomization{
						{},
					},
				},
			},
			expectedMsg: `invalid plain filesystem customization: mountpoint is empty`,
		},
		"unhappy-noabsmp": {
			partitioning: &blueprint.PartitioningCustomization{
				Plain: &blueprint.PlainFilesystemCustomization{
					Filesystems: []blueprint.FilesystemCustomization{
						{Mountpoint: "i-am-not-absolute"},
					},
				},
			},
			expectedMsg: `invalid plain filesystem customization: mountpoint "i-am-not-absolute" is not an absolute path`,
		},
		"unhappy-badmp": {
			partitioning: &blueprint.PartitioningCustomization{
				Plain: &blueprint.PlainFilesystemCustomization{
					Filesystems: []blueprint.FilesystemCustomization{
						{Mountpoint: "/home/../root"},
					},
				},
			},
			expectedMsg: `invalid plain filesystem customization: mountpoint "/home/../root" is not a canonical path (did you mean "/root"?)`,
		},
		"unhappy-emptymp-btrfs": {
			partitioning: &blueprint.PartitioningCustomization{
				Btrfs: &blueprint.BtrfsCustomization{
					Volumes: []blueprint.BtrfsVolumeCustomization{
						{
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
			expectedMsg: `invalid btrfs subvolume customization: mountpoint is empty`,
		},
		"unhappy-noabsmp-btrfs": {
			partitioning: &blueprint.PartitioningCustomization{
				Btrfs: &blueprint.BtrfsCustomization{
					Volumes: []blueprint.BtrfsVolumeCustomization{
						{
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
			expectedMsg: `invalid btrfs subvolume customization: mountpoint "blorpsmp" is not an absolute path`,
		},
		"unhappy-badmp-btrfs": {
			partitioning: &blueprint.PartitioningCustomization{
				Btrfs: &blueprint.BtrfsCustomization{
					Volumes: []blueprint.BtrfsVolumeCustomization{
						{
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
			expectedMsg: `invalid btrfs subvolume customization: mountpoint "/home//bork" is not a canonical path (did you mean "/home/bork"?)`,
		},
		"unhappy-emptymp-lvm": {
			partitioning: &blueprint.PartitioningCustomization{
				LVM: &blueprint.LVMCustomization{
					VolumeGroups: []blueprint.VGCustomization{
						{
							LogicalVolumes: []blueprint.LVCustomization{
								{
									Name: "testlv",
									FilesystemCustomization: blueprint.FilesystemCustomization{
										Mountpoint: "/stuff",
									},
								},
								{
									Name: "testlv2",
									FilesystemCustomization: blueprint.FilesystemCustomization{
										Mountpoint: "",
									},
								},
							},
						},
					},
				},
			},
			expectedMsg: `invalid logical volume customization: mountpoint is empty`,
		},
		"unhappy-noabsmp-lvm": {
			partitioning: &blueprint.PartitioningCustomization{
				LVM: &blueprint.LVMCustomization{
					VolumeGroups: []blueprint.VGCustomization{
						{
							LogicalVolumes: []blueprint.LVCustomization{
								{
									Name: "testlv",
									FilesystemCustomization: blueprint.FilesystemCustomization{
										Mountpoint: "/stuff",
									},
								},
								{
									Name: "testlv2",
									FilesystemCustomization: blueprint.FilesystemCustomization{
										Mountpoint: "i/like/relative/paths",
									},
								},
							},
						},
					},
				},
			},
			expectedMsg: `invalid logical volume customization: mountpoint "i/like/relative/paths" is not an absolute path`,
		},
		"unhappy-badmp-lvm": {
			partitioning: &blueprint.PartitioningCustomization{
				LVM: &blueprint.LVMCustomization{
					VolumeGroups: []blueprint.VGCustomization{
						{
							LogicalVolumes: []blueprint.LVCustomization{
								{
									Name: "testlv",
									FilesystemCustomization: blueprint.FilesystemCustomization{
										Mountpoint: "/../../../what/",
									},
								},
								{
									Name: "testlv2",
									FilesystemCustomization: blueprint.FilesystemCustomization{
										Mountpoint: "/test",
									},
								},
							},
						},
					},
				},
			},
			expectedMsg: `invalid logical volume customization: mountpoint "/../../../what/" is not a canonical path (did you mean "/what"?)`,
		},
		"unhappy-dupesubvolname": {
			partitioning: &blueprint.PartitioningCustomization{
				Btrfs: &blueprint.BtrfsCustomization{
					Volumes: []blueprint.BtrfsVolumeCustomization{
						{
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
			expectedMsg: `duplicate btrfs subvolume name "root" in partitioning customizations`,
		},
		"unhappy-dupelvname": {
			partitioning: &blueprint.PartitioningCustomization{
				LVM: &blueprint.LVMCustomization{
					VolumeGroups: []blueprint.VGCustomization{
						{
							LogicalVolumes: []blueprint.LVCustomization{
								{
									Name: "testlv",
									FilesystemCustomization: blueprint.FilesystemCustomization{
										Mountpoint: "/stuff",
									},
								},
								{
									Name: "testlv",
									FilesystemCustomization: blueprint.FilesystemCustomization{
										Mountpoint: "/stuff2",
									},
								},
							},
						},
					},
				},
			},
			expectedMsg: `duplicate lvm logical volume name "testlv" in volume group "" in partitioning customizations`,
		},
		"unhappy-emptyname-btrfs": {
			partitioning: &blueprint.PartitioningCustomization{
				Btrfs: &blueprint.BtrfsCustomization{
					Volumes: []blueprint.BtrfsVolumeCustomization{
						{
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
			expectedMsg: `btrfs subvolume with empty name in partitioning customizations`,
		},
		"boot-on-lvm": {
			partitioning: &blueprint.PartitioningCustomization{
				LVM: &blueprint.LVMCustomization{
					VolumeGroups: []blueprint.VGCustomization{
						{
							LogicalVolumes: []blueprint.LVCustomization{
								{
									Name: "bewt",
									FilesystemCustomization: blueprint.FilesystemCustomization{
										Mountpoint: "/boot",
									},
								},
								{
									Name: "testlv",
									FilesystemCustomization: blueprint.FilesystemCustomization{
										Mountpoint: "/stuff2",
									},
								},
							},
						},
					},
				},
			},
			expectedMsg: `invalid mountpoint "/boot" for logical volume`,
		},
		"bootefi-on-lvm": {
			partitioning: &blueprint.PartitioningCustomization{
				LVM: &blueprint.LVMCustomization{
					VolumeGroups: []blueprint.VGCustomization{
						{
							LogicalVolumes: []blueprint.LVCustomization{
								{
									Name: "bewtefi",
									FilesystemCustomization: blueprint.FilesystemCustomization{
										Mountpoint: "/boot/efi",
									},
								},
							},
						},
					},
				},
			},
			expectedMsg: `invalid mountpoint "/boot/efi" for logical volume`,
		},
		"boot-on-btrfs": {
			partitioning: &blueprint.PartitioningCustomization{
				Btrfs: &blueprint.BtrfsCustomization{
					Volumes: []blueprint.BtrfsVolumeCustomization{
						{
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
			expectedMsg: `invalid mountpoint "/boot" for btrfs subvolume`,
		},
		"bootefi-on-btrfs": {
			partitioning: &blueprint.PartitioningCustomization{
				Btrfs: &blueprint.BtrfsCustomization{
					Volumes: []blueprint.BtrfsVolumeCustomization{
						{
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
			expectedMsg: `invalid mountpoint "/boot/efi" for btrfs subvolume`,
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
