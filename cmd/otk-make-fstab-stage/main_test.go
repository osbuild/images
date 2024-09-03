package main_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	makefstab "github.com/osbuild/images/cmd/otk-make-fstab-stage"
	"github.com/osbuild/images/internal/otkdisk"
	"github.com/osbuild/images/internal/testdisk"
)

// this is not symetrical to the output, this is sad but also
// okay because the input is really just a dump of the internal
// disk.PartitionTable so encoding it in json here will not add
// a benefit for the test
var minimalInputBase = makefstab.Input{
	Tree: otkdisk.Data{
		Const: otkdisk.Const{
			Internal: otkdisk.Internal{
				PartitionTable: testdisk.MakeFakePartitionTable("/", "/var"),
			},
		},
	},
}

var minimalExpectedStages = `{
  "tree": {
    "type": "org.osbuild.fstab",
    "options": {
      "filesystems": [
        {
          "uuid": "6264D520-3FB9-423F-8AB8-7A0A8E3D3562",
          "vfs_type": "ext4",
          "path": "/"
        },
        {
          "uuid": "CB07C243-BC44-4717-853E-28852021225B",
          "vfs_type": "ext4",
          "path": "/var"
        }
      ]
    }
  }
}
`

func TestIntegration(t *testing.T) {
	minimalInput := minimalInputBase
	minimalInput.Tree.Const.Filename = "disk.img"
	expectedStages := minimalExpectedStages

	inpJSON, err := json.Marshal(&minimalInput)
	assert.NoError(t, err)
	fakeStdin := bytes.NewBuffer(inpJSON)
	fakeStdout := bytes.NewBuffer(nil)

	err = makefstab.Run(fakeStdin, fakeStdout)
	assert.NoError(t, err)

	assert.Equal(t, expectedStages, fakeStdout.String())
}

func TestIntegrationNoPartitionTable(t *testing.T) {
	fakeStdin := bytes.NewBufferString(`{}`)
	fakeStdout := bytes.NewBuffer(nil)
	err := makefstab.Run(fakeStdin, fakeStdout)
	assert.EqualError(t, err, "cannot validate input data: no partition table")
}
