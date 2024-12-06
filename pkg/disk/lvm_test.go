package disk

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/pkg/datasizes"
)

func TestLVMVCreateMountpoint(t *testing.T) {

	assert := assert.New(t)

	vg := &LVMVolumeGroup{
		Name:        "root",
		Description: "root volume group",
	}

	entity, err := vg.CreateMountpoint("/", 0)
	assert.NoError(err)
	rootlv := entity.(*LVMLogicalVolume)
	assert.Equal("rootlv", rootlv.Name)

	_, err = vg.CreateMountpoint("/home_test", 0)
	assert.NoError(err)

	entity, err = vg.CreateMountpoint("/home/test", 0)
	assert.NoError(err)

	dedup := entity.(*LVMLogicalVolume)
	assert.Equal("home_testlv00", dedup.Name)

	// Lets collide it
	for i := 0; i < 99; i++ {
		_, err = vg.CreateMountpoint("/home/test", 0)
		assert.NoError(err)
	}

	_, err = vg.CreateMountpoint("/home/test", 0)
	assert.Error(err)
}

func TestLVMVCreateLogicalVolumeSwap(t *testing.T) {
	vg := &LVMVolumeGroup{
		Name:        "root",
		Description: "root volume group",
	}
	swap := &Swap{}
	lv, err := vg.CreateLogicalVolume("", 12345, swap)
	assert.NoError(t, err)
	assert.Equal(t, "swaplv", lv.Name)
	// one more
	lv2, err := vg.CreateLogicalVolume("", 12345, swap)
	assert.NoError(t, err)
	assert.Equal(t, "swaplv00", lv2.Name)
}

func TestLVMVCreateLogicalVolumeWrongType(t *testing.T) {
	vg := &LVMVolumeGroup{
		Name: "root",
	}
	_, err := vg.CreateLogicalVolume("", 12345, &LUKSContainer{})
	assert.EqualError(t, err, `could not create logical volume: no name provided and payload *disk.LUKSContainer is not mountable or swap`)
}

func TestImplementsInterfacesCompileTimeCheckLVM(t *testing.T) {
	var _ = Container(&LVMVolumeGroup{})
	var _ = Sizeable(&LVMLogicalVolume{})
}

func TestLVMLogicalVolumeEnsureSize(t *testing.T) {
	lv := &LVMLogicalVolume{
		Size: 1024 * 1024,
	}
	resized := lv.EnsureSize(1024*1024 + 17)
	assert.True(t, resized)
	assert.Equal(t, uint64(4*datasizes.MiB), lv.Size)
}
