package disk

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBtrfsSubvolume_GetFSTabOptions(t *testing.T) {
	subvol := BtrfsSubvolume{
		Name:       "root",
		Mountpoint: "/",
		Compress:   "zstd:1",
	}
	actual := subvol.GetFSTabOptions()

	assert.Equal(t, FSTabOptions{
		MntOps: "subvol=root,compress=zstd:1",
	}, actual)
}
