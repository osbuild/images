package main_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	makeGrub2Inst "github.com/osbuild/images/cmd/otk/osbuild-make-grub2-inst-stage"
	"github.com/osbuild/images/internal/otkdisk"
	"github.com/osbuild/images/pkg/datasizes"
	"github.com/osbuild/images/pkg/disk"
)

var fakePt = &disk.PartitionTable{
	Type: disk.PT_GPT,
	Partitions: []disk.Partition{
		{
			Size:     1 * datasizes.MiB,
			Start:    1 * datasizes.MiB,
			Bootable: true,
			Type:     disk.BIOSBootPartitionGUID,
			UUID:     disk.BIOSBootPartitionUUID,
		},
		{
			Size: 1 * datasizes.GiB,
			Payload: &disk.Filesystem{
				Type:       "ext4",
				Mountpoint: "/",
				UUID:       disk.RootPartitionUUID,
			},
		},
	},
}

// this is not symetrical to the output, this is sad but also
// okay because the input is really just a dump of the internal
// disk.PartitionTable so encoding it in json here will not add
// a benefit for the test
var minimalInputBase = makeGrub2Inst.Input{
	Tree: makeGrub2Inst.Tree{
		Platform: "i386-pc",
		Filesystem: otkdisk.Data{
			Const: otkdisk.Const{
				Internal: otkdisk.Internal{
					PartitionTable: fakePt,
				},
			},
		},
	},
}

var minimalExpectedStages = `{
  "tree": {
    "type": "org.osbuild.grub2.inst",
    "options": {
      "filename": "disk.img",
      "platform": "i386-pc",
      "location": 2048,
      "core": {
        "type": "mkimage",
        "partlabel": "gpt",
        "filesystem": "ext4"
      },
      "prefix": {
        "type": "partition",
        "partlabel": "gpt",
        "number": 1,
        "path": "/boot/grub2"
      }
    }
  }
}
`

func TestIntegration(t *testing.T) {
	minimalInput := minimalInputBase
	minimalInput.Tree.Filesystem.Const.Filename = "disk.img"
	expectedStages := minimalExpectedStages

	inpJSON, err := json.Marshal(&minimalInput)
	assert.NoError(t, err)
	fakeStdin := bytes.NewBuffer(inpJSON)
	fakeStdout := bytes.NewBuffer(nil)

	err = makeGrub2Inst.Run(fakeStdin, fakeStdout)
	assert.NoError(t, err)

	assert.Equal(t, expectedStages, fakeStdout.String())
}
