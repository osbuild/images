package main_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	genpart "github.com/osbuild/images/cmd/otk-gen-partition-table"
	"github.com/osbuild/images/pkg/disk"
)

var expectedInput = &genpart.OtkGenPartitionInput{
	Options: &genpart.OtkPartOptions{
		UEFI: &genpart.OtkPartUEFI{
			Size: "1 GiB",
		},
		BIOS: true,
		Type: "gpt",
		Size: "10 GiB",
	},
	Partitions: []*genpart.OtkPartition{
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
}

func TestUnmarshalInput(t *testing.T) {
	var otkInput genpart.OtkGenPartitionInput
	err := json.Unmarshal([]byte(simplePartOptions), &otkInput)
	assert.NoError(t, err)
	assert.Equal(t, expectedInput, &otkInput)
}

func TestUnmarshalOutput(t *testing.T) {
	fakeOtkOutput := &genpart.OtkGenPartitionsOutput{
		Const: genpart.OtkGenPartConstOutput{
			KernelOptsList: []string{"root=UUID=1234"},
			PartitionMap: map[string]genpart.OtkPublicPartition{
				"root": {
					UUID: "12345",
				},
			},
			Internal: genpart.OtkGenPartitionsInternal{
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
    }
  }
}`
	output, err := json.MarshalIndent(fakeOtkOutput, "", "  ")
	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, string(output))
}

// see https://github.com/achilleas-k/images/pull/2#issuecomment-2136025471
var simplePartOptions = `
{
  "options": {
    "uefi": {
      "size": "1 GiB"
    },
    "bios": true,
    "type": "gpt",
    "size": "10 GiB"
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
`

// XXX: anything under "internal" we don't actually need to test
// as we do not make any gurantees to the outside
var expectedSimplePartOutput = `{
  "const": {
    "kernel_opts_list": [],
    "partition_map": {
      "root": {
        "uuid": "9851898e-0b30-437d-8fad-51ec16c3697f"
      }
    },
    "internal": {
      "partition-table": {
        "Size": 10740563968,
        "UUID": "dbd21911-1c4e-4107-8a9f-14fe6e751358",
        "Type": "gpt",
        "Partitions": [
          {
            "Start": 1048576,
            "Size": 1048576,
            "Type": "21686148-6449-6E6F-744E-656564454649",
            "Bootable": true,
            "UUID": "FAC7F1FB-3E8D-4137-A512-961DE09A5549",
            "Payload": null,
            "PayloadType": "no-payload"
          },
          {
            "Start": 2097152,
            "Size": 1073741824,
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
            "Start": 3223322624,
            "Size": 7517224448,
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
            "Start": 1075838976,
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
        "StartOffset": 0
      }
    }
  }
}
`

var expectedOutput = &genpart.OtkGenPartitionsOutput{
	Const: genpart.OtkGenPartConstOutput{
		KernelOptsList: []string{},
		PartitionMap: map[string]genpart.OtkPublicPartition{
			"root": {
				UUID: "6e4ff95f-f662-45ee-a82a-bdf44a2d0b75",
			},
		},
		Internal: genpart.OtkGenPartitionsInternal{
			PartitionTable: &disk.PartitionTable{
				Size: 10740563968,
				UUID: "0194fdc2-fa2f-4cc0-81d3-ff12045b73c8",
				Type: "gpt",
				Partitions: []disk.Partition{
					{
						Start:    1048576,
						Size:     1048576,
						Type:     "21686148-6449-6E6F-744E-656564454649",
						Bootable: true,
						UUID:     "FAC7F1FB-3E8D-4137-A512-961DE09A5549",
					}, {
						Start:    2097152,
						Size:     1073741824,
						Type:     "C12A7328-F81F-11D2-BA4B-00A0C93EC93B",
						Bootable: false,
						UUID:     "68B2905B-DF3E-4FB3-80FA-49D1E773AA33",
						Payload: &disk.Filesystem{
							Type:         "vfat",
							UUID:         "7B77-95E7",
							Label:        "EFI-SYSTEM",
							Mountpoint:   "/boot/efi",
							FSTabOptions: "defaults,uid=0,gid=0,umask=077,shortname=winnt",
							FSTabFreq:    0,
							FSTabPassNo:  2,
						},
					}, {
						Start: 3223322624,
						Size:  7517224448,
						UUID:  "a178892e-e285-4ce1-9114-55780875d64e",
						Payload: &disk.Filesystem{
							Type:       "ext4",
							UUID:       "6e4ff95f-f662-45ee-a82a-bdf44a2d0b75",
							Label:      "root",
							Mountpoint: "/",
						},
					}, {
						Start: 1075838976,
						Size:  2147483648,
						UUID:  "e2d3d0d0-de6b-48f9-b44c-e85ff044c6b1",
						Payload: &disk.Filesystem{
							Type:       "ext4",
							UUID:       "fb180daf-48a7-4ee0-b10d-394651850fd4",
							Label:      "home",
							Mountpoint: "/home",
						},
					},
				},
			},
		},
	},
}

func TestIntegration(t *testing.T) {
	t.Setenv("OSBUILD_TESTING_RNG_SEED", "0")

	inp := bytes.NewBufferString(simplePartOptions)
	outp := bytes.NewBuffer(nil)
	err := genpart.Run(inp, outp)
	assert.NoError(t, err)
	assert.Equal(t, expectedSimplePartOutput, outp.String())
}
