package main_test

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	genpart "github.com/osbuild/images/cmd/otk/osbuild-gen-partition-table"
	"github.com/osbuild/images/internal/otkdisk"
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/datasizes"
	"github.com/osbuild/images/pkg/disk"
)

// see https://github.com/achilleas-k/images/pull/2#issuecomment-2136025471
// Note that this partition table contains all json keys for testing, some may
// contradict each other
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
      "bootable": true,
      "mountpoint": "/",
      "label": "root",
      "size": "7 GiB",
      "type": "ext4",
      "part_uuid": "0FC63DAF-8483-4772-8E79-3D69D8477DE4"
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
			Bootable:   true,
			Mountpoint: "/",
			Label:      "root",
			Size:       "7 GiB",
			Type:       "ext4",
			PartUUID:   "0FC63DAF-8483-4772-8E79-3D69D8477DE4",
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
							Type: "119119",
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
        "size": 911,
        "type": "",
        "partitions": [
          {
            "size": 0,
            "type": "119119",
            "uuid": "911911",
            "payload": {
              "type": "ext4"
            },
            "payload_type": "filesystem"
          }
        ]
      }
    },
    "filename": "disk.img"
  }
}`
	output, err := json.MarshalIndent(fakeOtkOutput, "", "  ")
	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, string(output))
}

var partInputsSimple = `{
  "tree": {
    "properties": {
      "create": {
        "bios_boot_partition": true,
        "esp_partition": true,
        "esp_partition_size": "2 GiB"
      },
      "type": "gpt",
      "default_size": "10 GiB",
      "start_offset": "8 MB",
      "architecture": "x86_64"
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
}
`

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
          "size": 11821645824,
          "uuid": "dbd21911-1c4e-4107-8a9f-14fe6e751358",
          "type": "gpt",
          "partitions": [
            {
              "start": 9048576,
              "size": 1048576,
              "type": "21686148-6449-6E6F-744E-656564454649",
              "bootable": true,
              "uuid": "FAC7F1FB-3E8D-4137-A512-961DE09A5549"
            },
            {
              "start": 10097152,
              "size": 2147483648,
              "type": "C12A7328-F81F-11D2-BA4B-00A0C93EC93B",
              "uuid": "68B2905B-DF3E-4FB3-80FA-49D1E773AA33",
              "payload": {
                "type": "vfat",
                "uuid": "7B77-95E7",
                "label": "EFI-SYSTEM",
                "mountpoint": "/boot/efi",
                "fstab_options": "defaults,uid=0,gid=0,umask=077,shortname=winnt",
                "fstab_passno": 2
              },
              "payload_type": "filesystem"
            },
            {
              "start": 4305064448,
              "size": 7516564480,
              "uuid": "ed130be6-c822-49af-83bb-4ea648bb2264",
              "payload": {
                "type": "ext4",
                "uuid": "9851898e-0b30-437d-8fad-51ec16c3697f",
                "label": "root",
                "mountpoint": "/"
              },
              "payload_type": "filesystem"
            },
            {
              "start": 2157580800,
              "size": 2147483648,
              "uuid": "9f6173fd-edc9-4dbe-9313-632af556c607",
              "payload": {
                "type": "ext4",
                "uuid": "d8bb61b8-81cf-4c85-937b-69439a23dc5e",
                "label": "home",
                "mountpoint": "/home"
              },
              "payload_type": "filesystem"
            }
          ],
          "start_offset": 8000000
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

func TestGenPartitionTableBootable(t *testing.T) {
	inp := &genpart.Input{
		Properties: genpart.InputProperties{
			Type: "dos",
		},
		Partitions: []*genpart.InputPartition{
			{
				Bootable:   true,
				Mountpoint: "/",
				Size:       "10 GiB",
				Type:       "ext4",
			},
		},
	}

	output, err := genpart.GenPartitionTable(inp, rand.New(rand.NewSource(0))) /* #nosec G404 */
	assert.NoError(t, err)
	assert.Equal(t, true, output.Const.Internal.PartitionTable.Partitions[0].Bootable)
}

func TestGenPartitionTableIntegrationPPC(t *testing.T) {
	inp := &genpart.Input{
		Properties: genpart.InputProperties{
			Type:        "dos",
			DefaultSize: "10 GiB",
			UUID:        "0x14fc63d2",
		},
		Partitions: []*genpart.InputPartition{
			{
				Name:     "ppc-boot",
				Bootable: true,
				Size:     "4 MiB",
				PartType: disk.PRepPartitionDOSID,
				PartUUID: "",
			},
			{
				Name:       "root",
				Size:       "10 GiB",
				Type:       "xfs",
				Mountpoint: "/",
			},
		},
	}
	expectedOutput := &otkdisk.Data{
		Const: otkdisk.Const{
			KernelOptsList: []string{},
			PartitionMap: map[string]otkdisk.Partition{
				"root": {
					UUID: "0194fdc2-fa2f-4cc0-81d3-ff12045b73c8",
				},
			},
			Filename: "disk.img",
			Internal: otkdisk.Internal{
				PartitionTable: &disk.PartitionTable{
					Size: 10742661120,
					UUID: "0x14fc63d2",
					Type: disk.PT_DOS,
					Partitions: []disk.Partition{
						{
							Bootable: true,
							Start:    1048576,
							Size:     4194304,
							Type:     disk.PRepPartitionDOSID,
						},
						{
							Start: 5242880,
							Size:  10737418240,
							Payload: &disk.Filesystem{
								Type:       "xfs",
								UUID:       "0194fdc2-fa2f-4cc0-81d3-ff12045b73c8",
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
					Type: disk.PT_DOS,
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
					MinSize:    3 * datasizes.GigaByte,
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
					Type: disk.PT_DOS,
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
							Type:  disk.LVMPartitionDOSID,
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
					MinSize:    3 * datasizes.GigaByte,
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
					Type: disk.PT_DOS,
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
					Type: disk.PT_DOS,
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
					Type: disk.PT_DOS,
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
					Type: disk.PT_DOS,
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

func TestGenPartitionCreateESPDos(t *testing.T) {
	inp := &genpart.Input{
		Properties: genpart.InputProperties{
			Type: "dos",
			Create: genpart.InputCreate{
				EspPartition:     true,
				EspPartitionSize: "2 GiB",
			},
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
					Size: 12885950464,
					Type: disk.PT_DOS,
					UUID: "0194fdc2-fa2f-4cc0-81d3-ff12045b73c8",
					Partitions: []disk.Partition{
						{
							Start:    1048576,
							Size:     2147483648,
							Bootable: true,
							Type:     "06",
							Payload: &disk.Filesystem{
								Type:         "vfat",
								UUID:         "7B77-95E7",
								Label:        "EFI-SYSTEM",
								Mountpoint:   "/boot/efi",
								FSTabOptions: "defaults,uid=0,gid=0,umask=077,shortname=winnt",
								FSTabPassNo:  2,
							},
						},
						{
							Start: 2148532224,
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
