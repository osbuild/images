package osbuild

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/internal/testdisk"

	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/disk"
	"github.com/stretchr/testify/assert"
)

func TestGenImageKernelOptions(t *testing.T) {
	assert := assert.New(t)

	// math/rand is good enough in this case
	/* #nosec G404 */
	rng := rand.New(rand.NewSource(13))

	luks_lvm := testPartitionTables["luks+lvm"]

	pt, err := disk.NewPartitionTable(&luks_lvm, []blueprint.FilesystemCustomization{}, 0, disk.AutoLVMPartitioningMode, make(map[string]uint64), rng)
	assert.NoError(err)

	var uuid string

	findLuksUUID := func(e disk.Entity, path []disk.Entity) error {
		switch ent := e.(type) {
		case *disk.LUKSContainer:
			uuid = ent.UUID
		}

		return nil
	}
	_ = pt.ForEachEntity(findLuksUUID)

	assert.NotEmpty(uuid, "Could not find LUKS container")
	cmdline := GenImageKernelOptions(pt)

	assert.Subset(cmdline, []string{"luks.uuid=" + uuid})
}

func TestGenImageKernelOptionsBtrfs(t *testing.T) {
	pt := testdisk.MakeFakeBtrfsPartitionTable("/")
	actual := GenImageKernelOptions(pt)
	assert.Equal(t, []string{"rootflags=subvol=root"}, actual)
}

func TestGenImagePrepareStages(t *testing.T) {
	pt := testdisk.MakeFakeBtrfsPartitionTable("/", "/boot")
	filename := "image.raw"
	actualStages := GenImagePrepareStages(pt, filename, PTSfdisk)

	assert.Equal(t, []*Stage{
		{
			Type: "org.osbuild.truncate",
			Options: &TruncateStageOptions{
				Filename: filename,
				Size:     fmt.Sprintf("%d", 10*common.GiB),
			},
		},
		{
			Type: "org.osbuild.sfdisk",
			Options: &SfdiskStageOptions{
				Label: "gpt",
				Partitions: []SfdiskPartition{
					{
						Size: 1 * common.GiB / 512,
					},
					{
						Start: 1 * common.GiB / 512,
						Size:  9 * common.GiB / 512,
					},
				},
			},
			Devices: map[string]Device{
				"device": {
					Type: "org.osbuild.loopback",
					Options: &LoopbackDeviceOptions{
						Filename: filename,
						Lock:     true,
					},
				},
			},
		},
		{
			Type: "org.osbuild.mkfs.ext4",
			Devices: map[string]Device{
				"device": {
					Type: "org.osbuild.loopback",
					Options: &LoopbackDeviceOptions{
						Filename: filename,
						Start:    0,
						Size:     1 * common.GiB / 512,
						Lock:     true,
					},
				},
			},
			Options: &MkfsExt4StageOptions{},
		},
		{
			Type: "org.osbuild.mkfs.btrfs",
			Devices: map[string]Device{
				"device": {
					Type: "org.osbuild.loopback",
					Options: &LoopbackDeviceOptions{
						Filename: filename,
						Start:    1 * common.GiB / 512,
						Size:     9 * common.GiB / 512,
						Lock:     true,
					},
				},
			},
			Options: &MkfsBtrfsStageOptions{
				UUID: "6264D520-3FB9-423F-8AB8-7A0A8E3D3562",
			},
		},
		{
			Type: "org.osbuild.btrfs.subvol",
			Devices: map[string]Device{
				"btrfs-6264": {
					Type: "org.osbuild.loopback",
					Options: &LoopbackDeviceOptions{
						Filename: filename,
						Start:    1 * common.GiB / 512,
						Size:     9 * common.GiB / 512,
						Lock:     false,
					},
				},
			},
			Mounts: []Mount{
				{
					Name:    "btrfs-6264",
					Type:    "org.osbuild.btrfs",
					Source:  "btrfs-6264",
					Target:  "/",
					Options: BtrfsMountOptions{},
				},
			},
			Options: &BtrfsSubVolOptions{
				Subvolumes: []BtrfsSubVol{
					{
						Name: "/root",
					},
				},
			},
		},
	}, actualStages)

}
