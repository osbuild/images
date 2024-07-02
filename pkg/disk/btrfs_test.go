package disk

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBtrfsSubvolume_GetFSTabOptions(t *testing.T) {
	for _, tc := range []struct {
		subvol          BtrfsSubvolume
		expectedMntOpts string
	}{
		{BtrfsSubvolume{Name: "name"}, "subvol=name"},
		{BtrfsSubvolume{Name: "name", Compress: "gzip"}, "subvol=name,compress=gzip"},
		{BtrfsSubvolume{Name: "root", Compress: "zstd:1", ReadOnly: true},
			"subvol=root,compress=zstd:1,ro"},
	} {
		actual := tc.subvol.GetFSTabOptions()

		assert.Equal(t, FSTabOptions{MntOps: tc.expectedMntOpts}, actual)
	}
}
