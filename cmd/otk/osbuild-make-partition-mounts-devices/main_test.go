package main_test

import (
	"bytes"
	"encoding/json"
	"testing"

	mkdevmnt "github.com/osbuild/images/cmd/otk/osbuild-make-partition-mounts-devices"
	"github.com/osbuild/images/internal/otkdisk"
	"github.com/osbuild/images/internal/testdisk"
	"github.com/stretchr/testify/assert"
)

const expectedOutput = `{
  "tree": {
    "root_mount_name": "-",
    "mounts": [
      {
        "name": "-",
        "type": "org.osbuild.ext4",
        "source": "-",
        "target": "/"
      },
      {
        "name": "boot",
        "type": "org.osbuild.ext4",
        "source": "boot",
        "target": "/boot"
      },
      {
        "name": "boot-efi",
        "type": "org.osbuild.fat",
        "source": "boot-efi",
        "target": "/boot/efi"
      }
    ],
    "devices": {
      "-": {
        "type": "org.osbuild.loopback",
        "options": {
          "filename": "test.disk",
          "size": 1615872
        }
      },
      "boot": {
        "type": "org.osbuild.loopback",
        "options": {
          "filename": "test.disk",
          "size": 1615872
        }
      },
      "boot-efi": {
        "type": "org.osbuild.loopback",
        "options": {
          "filename": "test.disk",
          "size": 1615872
        }
      }
    }
  }
}
`

func TestIntegration(t *testing.T) {
	pt := testdisk.MakeFakePartitionTable("/", "/boot", "/boot/efi")
	input := mkdevmnt.Input{
		Tree: otkdisk.Data{
			Const: otkdisk.Const{
				Filename: "test.disk",
				Internal: otkdisk.Internal{
					PartitionTable: pt,
				},
			},
		},
	}
	inpJSON, err := json.Marshal(&input)
	assert.NoError(t, err)
	fakeStdin := bytes.NewBuffer(inpJSON)
	fakeStdout := bytes.NewBuffer(nil)
	err = mkdevmnt.Run(fakeStdin, fakeStdout)
	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, fakeStdout.String())
}

func TestIntegrationNoPartitionTable(t *testing.T) {
	fakeStdin := bytes.NewBufferString(`{}`)
	fakeStdout := bytes.NewBuffer(nil)
	err := mkdevmnt.Run(fakeStdin, fakeStdout)
	assert.EqualError(t, err, "cannot validate input data: no partition table")
}
