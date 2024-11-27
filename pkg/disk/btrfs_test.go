package disk

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
		actual, err := tc.subvol.GetFSTabOptions()
		assert.NoError(t, err)

		assert.Equal(t, FSTabOptions{MntOps: tc.expectedMntOpts}, actual)
	}
}

func TestBtrfsSubvolume_GetFSTabOptionsPanics(t *testing.T) {
	subvol := &BtrfsSubvolume{}
	_, err := subvol.GetFSTabOptions()
	assert.EqualError(t, err, `internal error: BtrfsSubvolume.GetFSTabOptions() for &{Name: Size:0 Mountpoint: GroupID:0 Compress: ReadOnly:false UUID:} called without a name`)
}

func TestImplementsInterfacesCompileTimeCheckBtrfs(t *testing.T) {
	var _ = Container(&Btrfs{})
	var _ = UniqueEntity(&Btrfs{})
	var _ = Mountable(&BtrfsSubvolume{})
	var _ = Sizeable(&BtrfsSubvolume{})
	var _ = FSTabEntity(&BtrfsSubvolume{})
}
