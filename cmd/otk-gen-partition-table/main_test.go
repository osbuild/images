package main_test

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	genpart "github.com/osbuild/images/cmd/otk-gen-partition-table"
	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/internal/otkdisk"
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/disk"
)

// see https://github.com/achilleas-k/images/pull/2#issuecomment-2136025471
var partInputsComplete = `
{
  "properties": {
    "create": {
      "bios_boot_partition": true,
      "esp_partition": true,
      "esp_partition_size": "2 GiB"
    },
    "bios": true,
    "type": "gpt",
    "default_size": "10 GiB",
    "start_offset": "8 MB"
  },
  "partitions": [
    {
      "name": "root",
      "mountpoint": "/",
      "label": "root",
      "size": "7 GiB",
      "type": "ext4"
    },
    {
      "name": "home",
      "mountpoint": "/home",
      "label": "home",
      "size": "2 GiB",
      "type": "ext4"
    }
  ],
  "modifications": {
    "min_disk_size": "20 GiB",
    "partition_mode": "auto-lvm",
    "filesystems": [
      {"mountpoint": "/var/log", "minsize": 10241024}
    ]
  }
}`

var expectedInput = &genpart.Input{
	Properties: genpart.InputProperties{
		Create: genpart.InputCreate{
			BIOSBootPartition: true,
			EspPartition:      true,
			EspPartitionSize:  "2 GiB",
		},
		Type:        "gpt",
		DefaultSize: "10 GiB",
		StartOffset: "8 MB",
	},
	Partitions: []*genpart.InputPartition{
		{
			Name:       "root",
			Mountpoint: "/",
			Label:      "root",
			Size:       "7 GiB",
			Type:       "ext4",
		}, {
			Name:       "home",
			Mountpoint: "/home",
			Label:      "home",
			Size:       "2 GiB",
			Type:       "ext4",
		},
	},
	Modifications: genpart.InputModifications{
		MinDiskSize:   "20 GiB",
		PartitionMode: disk.AutoLVMPartitioningMode,
		Filesystems: []blueprint.FilesystemCustomization{
			{
				Mountpoint: "/var/log",
				MinSize:    10241024,
			},
		},
	},
}

func TestUnmarshalInput(t *testing.T) {
	var otkInput genpart.Input
	err := json.Unmarshal([]byte(partInputsComplete), &otkInput)
	assert.NoError(t, err)
	assert.Equal(t, expectedInput, &otkInput)
}

func TestUnmarshalOutput(t *testing.T) {
	fakeOtkOutput := &otkdisk.Data{
		Const: otkdisk.Const{
			KernelOptsList: []string{"root=UUID=1234"},
			PartitionMap: map[string]otkdisk.Partition{
				"root": {
					UUID: "12345",
				},
			},
			Filename: "disk.img",
			Internal: otkdisk.Internal{
				PartitionTable: &disk.PartitionTable{
					Size: 911,
					Partitions: []disk.Partition{
						{
							UUID: "911911",
							Payload: &disk.Filesystem{
								Type: "ext4",
							},
						},
					},
				},
			},
		},
	}
	// XXX: anything under "internal" we don't actually need to test
	// as we do not make any gurantees to the outside
	expectedOutput := `{
  "const": {
    "kernel_opts_list": [
      "root=UUID=1234"
    ],
    "partition_map": {
      "root": {
        "uuid": "12345"
      }
    },
    "internal": {
      "partition-table": {
        "Size": 911,
        "UUID": "",
        "Type": "",
        "Partitions": [
          {
            "Start": 0,
            "Size": 0,
            "Type": "",
            "Bootable": false,
            "UUID": "911911",
            "Payload": {
              "Type": "ext4",
              "UUID": "",
              "Label": "",
              "Mountpoint": "",
              "FSTabOptions": "",
              "FSTabFreq": 0,
              "FSTabPassNo": 0
            },
            "PayloadType": "filesystem"
          }
        ],
        "SectorSize": 0,
        "ExtraPadding": 0,
        "StartOffset": 0
      }
    },
    "filename": "disk.img"
  }
}`
	output, err := json.MarshalIndent(fakeOtkOutput, "", "  ")
	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, string(output))
}

var partInputsSimple = `
{
  "tree": {
    "properties": {
      "create": {
	"bios_boot_partition": true,
	"esp_partition": true,
	"esp_partition_size": "2 GiB"
      },
      "type": "gpt",
      "default_size": "10 GiB",
      "start_offset": "8 MB"
    },
    "partitions": [
      {
	"name": "root",
	"mountpoint": "/",
	"label": "root",
	"size": "7 GiB",
	"type": "ext4"
      },
      {
	"name": "home",
	"mountpoint": "/home",
	"label": "home",
	"size": "2 GiB",
	"type": "ext4"
      }
    ]
  }
}`

// XXX: anything under "internal" we don't actually need to test
// as we do not make any gurantees to the outside
var expectedSimplePartOutput = `{
  "tree": {
    "const": {
      "kernel_opts_list": [],
      "partition_map": {
        "root": {
          "uuid": "9851898e-0b30-437d-8fad-51ec16c3697f"
        }
      },
      "internal": {
        "partition-table": {
          "Size": 11821645824,
          "UUID": "dbd21911-1c4e-4107-8a9f-14fe6e751358",
          "Type": "gpt",
          "Partitions": [
            {
              "Start": 9048576,
              "Size": 1048576,
              "Type": "21686148-6449-6E6F-744E-656564454649",
              "Bootable": true,
              "UUID": "FAC7F1FB-3E8D-4137-A512-961DE09A5549",
              "Payload": null,
              "PayloadType": "no-payload"
            },
            {
              "Start": 10097152,
              "Size": 2147483648,
              "Type": "C12A7328-F81F-11D2-BA4B-00A0C93EC93B",
              "Bootable": false,
              "UUID": "68B2905B-DF3E-4FB3-80FA-49D1E773AA33",
              "Payload": {
                "Type": "vfat",
                "UUID": "7B77-95E7",
                "Label": "EFI-SYSTEM",
                "Mountpoint": "/boot/efi",
                "FSTabOptions": "defaults,uid=0,gid=0,umask=077,shortname=winnt",
                "FSTabFreq": 0,
                "FSTabPassNo": 2
              },
              "PayloadType": "filesystem"
            },
            {
              "Start": 4305064448,
              "Size": 7516564480,
              "Type": "",
              "Bootable": false,
              "UUID": "ed130be6-c822-49af-83bb-4ea648bb2264",
              "Payload": {
                "Type": "ext4",
                "UUID": "9851898e-0b30-437d-8fad-51ec16c3697f",
                "Label": "root",
                "Mountpoint": "/",
                "FSTabOptions": "",
                "FSTabFreq": 0,
                "FSTabPassNo": 0
              },
              "PayloadType": "filesystem"
            },
            {
              "Start": 2157580800,
              "Size": 2147483648,
              "Type": "",
              "Bootable": false,
              "UUID": "9f6173fd-edc9-4dbe-9313-632af556c607",
              "Payload": {
                "Type": "ext4",
                "UUID": "d8bb61b8-81cf-4c85-937b-69439a23dc5e",
                "Label": "home",
                "Mountpoint": "/home",
                "FSTabOptions": "",
                "FSTabFreq": 0,
                "FSTabPassNo": 0
              },
              "PayloadType": "filesystem"
            }
          ],
          "SectorSize": 0,
          "ExtraPadding": 0,
          "StartOffset": 8000000
        }
      },
      "filename": "disk.img"
    }
  }
}
`

func TestIntegrationRealistic(t *testing.T) {
	t.Setenv("OSBUILD_TESTING_RNG_SEED", "0")

	inp := bytes.NewBufferString(partInputsSimple)
	outp := bytes.NewBuffer(nil)
	err := genpart.Run(inp, outp)
	assert.NoError(t, err)
	assert.Equal(t, expectedSimplePartOutput, outp.String())
}

func TestGenPartitionTableMinimal(t *testing.T) {
	// XXX: think about what the smalltest inputs can be and validate
	// that it's complete and/or provide defaults (e.g. for "type" for
	// partition and filesystem type)
	inp := &genpart.Input{
		Properties: genpart.InputProperties{
			Type: "dos",
		},
		Partitions: []*genpart.InputPartition{
			{
				Mountpoint: "/",
				Size:       "10 GiB",
				Type:       "ext4",
			},
		},
	}
	expectedOutput := &otkdisk.Data{
		Const: otkdisk.Const{
			KernelOptsList: []string{},
			PartitionMap: map[string]otkdisk.Partition{
				"root": {
					UUID: "6e4ff95f-f662-45ee-a82a-bdf44a2d0b75",
				},
			},
			Filename: "disk.img",
			Internal: otkdisk.Internal{
				PartitionTable: &disk.PartitionTable{
					Size: 10738466816,
					UUID: "0194fdc2-fa2f-4cc0-81d3-ff12045b73c8",
					Type: "dos",
					Partitions: []disk.Partition{
						{
							Start: 1048576,
							Size:  10737418240,
							Payload: &disk.Filesystem{
								Type:       "ext4",
								UUID:       "6e4ff95f-f662-45ee-a82a-bdf44a2d0b75",
								Mountpoint: "/",
							},
						},
					},
				},
			},
		},
	}
	output, err := genpart.GenPartitionTable(inp, rand.New(rand.NewSource(0))) /* #nosec G404 */
	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, output)
}

func TestGenPartitionTableCustomizationExtraMp(t *testing.T) {
	inp := &genpart.Input{
		Properties: genpart.InputProperties{
			Type: "dos",
		},
		Partitions: []*genpart.InputPartition{
			{
				Mountpoint: "/boot",
				Size:       "2 GiB",
				Type:       "ext4",
			},
			{
				Mountpoint: "/",
				Size:       "10 GiB",
				Type:       "ext4",
			},
		},
		Modifications: genpart.InputModifications{
			Filesystems: []blueprint.FilesystemCustomization{
				{
					Mountpoint: "/var/log",
					MinSize:    3 * common.GigaByte,
				},
			},
		},
	}
	expectedOutput := &otkdisk.Data{
		Const: otkdisk.Const{
			KernelOptsList: []string{},
			PartitionMap: map[string]otkdisk.Partition{
				"boot": {
					UUID: "6e4ff95f-f662-45ee-a82a-bdf44a2d0b75",
				},
			},
			Filename: "disk.img",
			Internal: otkdisk.Internal{
				PartitionTable: &disk.PartitionTable{
					Size: 15893266432,
					UUID: "0194fdc2-fa2f-4cc0-81d3-ff12045b73c8",
					Type: "dos",
					Partitions: []disk.Partition{
						{
							Start: 1048576,
							Size:  2147483648,
							Payload: &disk.Filesystem{
								Type:       "ext4",
								UUID:       "6e4ff95f-f662-45ee-a82a-bdf44a2d0b75",
								Mountpoint: "/boot",
							},
						},
						{
							Start: 2148532224,
							Size:  13744734208,
							Type:  "8e",
							Payload: &disk.LVMVolumeGroup{
								Name:        "rootvg",
								Description: "created via lvm2 and osbuild",
								LogicalVolumes: []disk.LVMLogicalVolume{
									{
										Name: "rootlv",
										Size: 10737418240,
										Payload: &disk.Filesystem{
											Mountpoint: "/",
											Type:       "ext4",
											UUID:       "fb180daf-48a7-4ee0-b10d-394651850fd4",
										},
									}, {
										Name: "var_loglv",
										Size: 3003121664,
										Payload: &disk.Filesystem{
											Mountpoint: "/var/log",
											// XXX: this is confusing
											Type: "xfs",
											UUID: "a178892e-e285-4ce1-9114-55780875d64e",
											// XXX: is this needed?
											FSTabOptions: "defaults",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	// default partition mode is "auto-lvm" so LVM is created by default
	output, err := genpart.GenPartitionTable(inp, rand.New(rand.NewSource(0))) /* #nosec G404 */
	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, output)
}

func TestGenPartitionTableCustomizationExtraMpPlusModificationPartitionMode(t *testing.T) {
	inp := &genpart.Input{
		Properties: genpart.InputProperties{
			Type: "dos",
		},
		Partitions: []*genpart.InputPartition{
			{
				Mountpoint: "/",
				Size:       "10 GiB",
				Type:       "ext4",
			},
		},
		Modifications: genpart.InputModifications{
			// note that the extra partitin mode is used here
			PartitionMode: disk.RawPartitioningMode,
			Filesystems: []blueprint.FilesystemCustomization{
				{
					Mountpoint: "/var/log",
					MinSize:    3 * common.GigaByte,
				},
			},
		},
	}
	expectedOutput := &otkdisk.Data{
		Const: otkdisk.Const{
			KernelOptsList: []string{},
			PartitionMap: map[string]otkdisk.Partition{
				"root": {
					UUID: "6e4ff95f-f662-45ee-a82a-bdf44a2d0b75",
				},
			},
			Filename: "disk.img",
			Internal: otkdisk.Internal{
				PartitionTable: &disk.PartitionTable{
					Size: 13739491328,
					UUID: "0194fdc2-fa2f-4cc0-81d3-ff12045b73c8",
					Type: "dos",
					Partitions: []disk.Partition{
						{
							Start: 3002073088,
							Size:  10737418240,
							Payload: &disk.Filesystem{
								Mountpoint: "/",
								Type:       "ext4",
								UUID:       "6e4ff95f-f662-45ee-a82a-bdf44a2d0b75",
							},
						}, {
							Start: 1048576,
							Size:  3001024512,
							Payload: &disk.Filesystem{
								Mountpoint: "/var/log",
								// XXX: this is confusing
								Type:         "xfs",
								UUID:         "fb180daf-48a7-4ee0-b10d-394651850fd4",
								FSTabOptions: "defaults",
							},
						},
					},
				},
			},
		},
	}
	output, err := genpart.GenPartitionTable(inp, rand.New(rand.NewSource(0))) /* #nosec G404 */
	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, output)
}

func TestGenPartitionTablePropertiesDefaultSize(t *testing.T) {
	inp := &genpart.Input{
		Properties: genpart.InputProperties{
			Type:        "dos",
			DefaultSize: "15 GiB",
		},
		Partitions: []*genpart.InputPartition{
			{
				Mountpoint: "/",
				Size:       "10 GiB",
				Type:       "ext4",
			},
		},
	}
	expectedOutput := &otkdisk.Data{
		Const: otkdisk.Const{
			KernelOptsList: []string{},
			PartitionMap: map[string]otkdisk.Partition{
				"root": {
					UUID: "6e4ff95f-f662-45ee-a82a-bdf44a2d0b75",
				},
			},
			Filename: "disk.img",
			Internal: otkdisk.Internal{
				PartitionTable: &disk.PartitionTable{
					Size: 16106127360,
					UUID: "0194fdc2-fa2f-4cc0-81d3-ff12045b73c8",
					Type: "dos",
					Partitions: []disk.Partition{
						{
							Start: 1048576,
							Size:  16105078784,
							Payload: &disk.Filesystem{
								Type:       "ext4",
								UUID:       "6e4ff95f-f662-45ee-a82a-bdf44a2d0b75",
								Mountpoint: "/",
							},
						},
					},
				},
			},
		},
	}
	output, err := genpart.GenPartitionTable(inp, rand.New(rand.NewSource(0))) /* #nosec G404 */
	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, output)
}

func TestGenPartitionTableModificationMinDiskSize(t *testing.T) {
	inp := &genpart.Input{
		Properties: genpart.InputProperties{
			Type:        "dos",
			DefaultSize: "15 GiB",
		},
		Partitions: []*genpart.InputPartition{
			{
				Mountpoint: "/",
				Size:       "10 GiB",
				Type:       "ext4",
			},
		},
		Modifications: genpart.InputModifications{
			MinDiskSize: "20 GiB",
		},
	}
	expectedOutput := &otkdisk.Data{
		Const: otkdisk.Const{
			KernelOptsList: []string{},
			PartitionMap: map[string]otkdisk.Partition{
				"root": {
					UUID: "6e4ff95f-f662-45ee-a82a-bdf44a2d0b75",
				},
			},
			Filename: "disk.img",
			Internal: otkdisk.Internal{
				PartitionTable: &disk.PartitionTable{
					Size: 21474836480,
					UUID: "0194fdc2-fa2f-4cc0-81d3-ff12045b73c8",
					Type: "dos",
					Partitions: []disk.Partition{
						{
							Start: 1048576,
							Size:  21473787904,
							Payload: &disk.Filesystem{
								Type:       "ext4",
								UUID:       "6e4ff95f-f662-45ee-a82a-bdf44a2d0b75",
								Mountpoint: "/",
							},
						},
					},
				},
			},
		},
	}
	output, err := genpart.GenPartitionTable(inp, rand.New(rand.NewSource(0))) /* #nosec G404 */
	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, output)
}

func TestGenPartitionTableModificationFilename(t *testing.T) {
	inp := &genpart.Input{
		Properties: genpart.InputProperties{
			Type: "dos",
		},
		Partitions: []*genpart.InputPartition{
			{
				Mountpoint: "/",
				Size:       "10 GiB",
				Type:       "ext4",
			},
		},
		Modifications: genpart.InputModifications{
			Filename: "custom-disk.img",
		},
	}
	expectedOutput := &otkdisk.Data{
		Const: otkdisk.Const{
			KernelOptsList: []string{},
			PartitionMap: map[string]otkdisk.Partition{
				"root": {
					UUID: "6e4ff95f-f662-45ee-a82a-bdf44a2d0b75",
				},
			},
			Filename: "custom-disk.img",
			Internal: otkdisk.Internal{
				PartitionTable: &disk.PartitionTable{
					Size: 10738466816,
					UUID: "0194fdc2-fa2f-4cc0-81d3-ff12045b73c8",
					Type: "dos",
					Partitions: []disk.Partition{
						{
							Start: 1048576,
							Size:  10737418240,
							Payload: &disk.Filesystem{
								Type:       "ext4",
								UUID:       "6e4ff95f-f662-45ee-a82a-bdf44a2d0b75",
								Mountpoint: "/",
							},
						},
					},
				},
			},
		},
	}
	output, err := genpart.GenPartitionTable(inp, rand.New(rand.NewSource(0))) /* #nosec G404 */
	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, output)
}

func TestGenPartitionTableValidates(t *testing.T) {
	inp := &genpart.Input{
		Properties: genpart.InputProperties{
			Type: "invalid-type",
		},
	}
	_, err := genpart.GenPartitionTable(inp, rand.New(rand.NewSource(0))) /* #nosec G404 */
	assert.EqualError(t, err, `cannot validate inputs: unsupported partition type "invalid-type"`)
}
