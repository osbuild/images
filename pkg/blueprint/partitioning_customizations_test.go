package blueprint_test

import (
	"encoding/json"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/datasizes"
	"github.com/stretchr/testify/assert"
)

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
