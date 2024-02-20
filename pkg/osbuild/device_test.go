package osbuild

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/internal/testdisk"
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/disk"
)

func TestGenDeviceCreationStages(t *testing.T) {
	assert := assert.New(t)

	// math/rand is good enough in this case
	/* #nosec G404 */
	rng := rand.New(rand.NewSource(13))

	luks_lvm := testPartitionTables["luks+lvm"]

	pt, err := disk.NewPartitionTable(&luks_lvm, []blueprint.FilesystemCustomization{}, 0, disk.AutoLVMPartitioningMode, make(map[string]uint64), rng)
	assert.NoError(err)

	stages := GenDeviceCreationStages(pt, "image.raw")

	// we should have two stages
	assert.Equal(len(stages), 2)

	// first one should be a "org.osbuild.luks2.format"
	luks := stages[0]
	assert.Equal(luks.Type, "org.osbuild.luks2.format")

	// it needs to have one device
	assert.Equal(len(luks.Devices), 1)

	// the device should be called `device`
	device, ok := luks.Devices["device"]
	assert.True(ok, "Need device called `device`")

	// device should be a loopback device
	assert.Equal(device.Type, "org.osbuild.loopback")

	lvm := stages[1]
	assert.Equal(lvm.Type, "org.osbuild.lvm2.create")
	lvmOptions, ok := lvm.Options.(*LVM2CreateStageOptions)
	assert.True(ok, "Need LVM2CreateStageOptions for org.osbuild.lvm2.create")

	// LVM should have two volumes
	assert.Equal(len(lvmOptions.Volumes), 2)
	rootlv := lvmOptions.Volumes[0]
	assert.Equal(rootlv.Name, "rootlv")

	homelv := lvmOptions.Volumes[1]
	assert.Equal(homelv.Name, "homelv")

	// it needs to have two(!) devices, the loopback and the luks
	assert.Equal(len(lvm.Devices), 2)

	// this is the target one, which should be the luks one
	device, ok = lvm.Devices["device"]
	assert.True(ok, "Need device called `device`")
	assert.Equal(device.Type, "org.osbuild.luks2")
	assert.NotEmpty(device.Parent, "Need a parent device for LUKS on loopback")

	luksOptions, ok := device.Options.(*LUKS2DeviceOptions)
	assert.True(ok, "Need LUKS2DeviceOptions for luks device")
	assert.Equal(luksOptions.Passphrase, "osbuild")

	parent, ok := lvm.Devices[device.Parent]
	assert.True(ok, "Need device called `device`")
	assert.Equal(parent.Type, "org.osbuild.loopback")

}

func TestGenDeviceFinishStages(t *testing.T) {
	assert := assert.New(t)

	// math/rand is good enough in this case
	/* #nosec G404 */
	rng := rand.New(rand.NewSource(13))

	luks_lvm := testPartitionTables["luks+lvm"]

	pt, err := disk.NewPartitionTable(&luks_lvm, []blueprint.FilesystemCustomization{}, 0, disk.AutoLVMPartitioningMode, make(map[string]uint64), rng)
	assert.NoError(err)

	stages := GenDeviceFinishStages(pt, "image.raw")

	// we should have one stage
	assert.Equal(1, len(stages))

	// it should be a "org.osbuild.lvm2.metadata"
	lvm := stages[0]
	assert.Equal("org.osbuild.lvm2.metadata", lvm.Type)

	// it should have two devices
	assert.Equal(2, len(lvm.Devices))

	// this is the target one, which should be the luks one
	device, ok := lvm.Devices["device"]
	assert.True(ok, "Need device called `device`")
	assert.Equal("org.osbuild.luks2", device.Type)
	assert.NotEmpty(device.Parent, "Need a parent device for LUKS on loopback")

	luksOptions, ok := device.Options.(*LUKS2DeviceOptions)
	assert.True(ok, "Need LUKS2DeviceOptions for luks device")
	assert.Equal("osbuild", luksOptions.Passphrase)

	parent, ok := lvm.Devices[device.Parent]
	assert.True(ok, "Need device called `device`")
	assert.Equal("org.osbuild.loopback", parent.Type)

	opts, ok := lvm.Options.(*LVM2MetadataStageOptions)
	assert.True(ok, "Need LVM2MetadataStageOptions for org.osbuild.lvm2.metadata")
	assert.Equal("root", opts.VGName)
}

func TestGenDeviceFinishStagesOrderWithLVMClevisBind(t *testing.T) {
	assert := assert.New(t)

	// math/rand is good enough in this case
	/* #nosec G404 */
	rng := rand.New(rand.NewSource(13))

	luks_lvm := testPartitionTables["luks+lvm+clevisBind"]

	pt, err := disk.NewPartitionTable(&luks_lvm, []blueprint.FilesystemCustomization{}, 0, disk.AutoLVMPartitioningMode, make(map[string]uint64), rng)
	assert.NoError(err)

	stages := GenDeviceFinishStages(pt, "image.raw")

	// we should have two stages
	assert.Equal(2, len(stages))
	lvm := stages[0]
	luks := stages[1]

	// the first one should be "org.osbuild.lvm2.metadata"
	assert.Equal("org.osbuild.lvm2.metadata", lvm.Type)
	// followed by "org.osbuild.luks2.remove-key"
	assert.Equal("org.osbuild.luks2.remove-key", luks.Type)
}

func TestPathEscape(t *testing.T) {
	testCases := []struct {
		path     string
		expected string
	}{
		{"", "-"},
		{"/", "-"},
		{"/root", "root"},
		{"/root/", "root"},
		{"/home/shadowman", "home-shadowman"},
		{"/home/s.o.s", "home-s.o.s"},
		{"/path/to/dir", "path-to-dir"},
		{"/path/with\\backslash", "path-with\\x5cbackslash"},
		{"/path-with-dash", "path\\x2dwith\\x2ddash"},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			result := pathEscape(tc.path)
			if result != tc.expected {
				t.Errorf("pathEscape(%q) = %q; expected %q", tc.path, result, tc.expected)
			}
		})
	}
}

func TestMountsDeviceFromPtEmptyErrors(t *testing.T) {
	filename := "fake-disk.img"
	fakePt := testdisk.MakeFakePartitionTable()
	fsRootMntName, mounts, devices, err := genMountsDevicesFromPt(filename, fakePt)
	assert.ErrorContains(t, err, "no mount found for the filesystem root")
	assert.Equal(t, fsRootMntName, "")
	require.Nil(t, mounts)
	require.Nil(t, devices)
}

func TestMountsDeviceFromPtNoRootErrors(t *testing.T) {
	filename := "fake-disk.img"
	fakePt := testdisk.MakeFakePartitionTable("/not-root")
	_, _, _, err := genMountsDevicesFromPt(filename, fakePt)
	assert.ErrorContains(t, err, "no mount found for the filesystem root")
}

func TestMountsDeviceFromPtHappy(t *testing.T) {
	filename := "fake-disk.img"
	fakePt := testdisk.MakeFakePartitionTable("/")
	fsRootMntName, mounts, devices, err := genMountsDevicesFromPt(filename, fakePt)
	require.Nil(t, err)
	assert.Equal(t, fsRootMntName, "-")
	assert.Equal(t, mounts, []Mount{
		{Name: "-", Type: "org.osbuild.ext4", Source: "-", Target: "/"},
	})
	assert.Equal(t, devices, map[string]Device{
		"-": {
			Type: "org.osbuild.loopback",
			Options: &LoopbackDeviceOptions{
				Filename: "fake-disk.img",
				Size:     testdisk.FakePartitionSize / 512,
			},
		},
	})
}
