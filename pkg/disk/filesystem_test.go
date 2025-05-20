package disk_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/pkg/disk"
)

func TestImplementsInterfacesCompileTimeCheckFilesystem(t *testing.T) {
	var _ = disk.Mountable(&disk.Filesystem{})
	var _ = disk.UniqueEntity(&disk.Filesystem{})
	var _ = disk.FSTabEntity(&disk.Filesystem{})
}

func TestMkfsOptionUnmarshalHappy(t *testing.T) {
	for _, tc := range []struct {
		inp      string
		expected disk.MkfsOption
	}{
		{`"verity"`, disk.MkfsVerity},
	} {
		var opt disk.MkfsOption
		err := json.Unmarshal([]byte(tc.inp), &opt)
		assert.NoError(t, err)
		assert.Equal(t, tc.expected, opt)

		// encoding again yields the same result
		encoded, err := json.Marshal(opt)
		assert.NoError(t, err)
		assert.Equal(t, tc.inp, string(encoded))
	}
}

func TestMkfsOptionUnmarshalSad(t *testing.T) {
	var opt disk.MkfsOption
	err := json.Unmarshal([]byte(`"invalid-mkfs-option"`), &opt)
	assert.EqualError(t, err, `invalid mkfsoption: invalid-mkfs-option`)
}
