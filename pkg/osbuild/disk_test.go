package osbuild

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/osbuild/images/internal/testdisk"
	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/datasizes"
	"github.com/osbuild/images/pkg/disk"
	"github.com/stretchr/testify/assert"
)

func TestGenImageKernelOptions(t *testing.T) {
	assert := assert.New(t)

	// math/rand is good enough in this case
	/* #nosec G404 */
	rng := rand.New(rand.NewSource(13))

	luks_lvm := testPartitionTables["luks+lvm"]

	pt, err := disk.NewPartitionTable(&luks_lvm, []blueprint.FilesystemCustomization{}, 0, disk.AutoLVMPartitioningMode, arch.ARCH_X86_64, make(map[string]uint64), rng)
	assert.NoError(err)

	var expRootUUID, expLuksUUID string

	findUUIDs := func(e disk.Entity, path []disk.Entity) error {
		switch ent := e.(type) {
		case *disk.LUKSContainer:
			expLuksUUID = ent.UUID
		case *disk.Filesystem:
			if ent.Mountpoint == "/" {
				expRootUUID = ent.UUID
			}
		}

		return nil
	}
	_ = pt.ForEachEntity(findUUIDs)

	assert.NotEmpty(expRootUUID, "Could not find root filesystem")
	assert.NotEmpty(expLuksUUID, "Could not find LUKS container")
	rootUUID, cmdline, err := GenImageKernelOptions(pt, false)
	assert.NoError(err)

	assert.Equal(rootUUID, expRootUUID)
	assert.Subset(cmdline, []string{"luks.uuid=" + expLuksUUID})
}

func TestGenImageKernelOptionsBtrfs(t *testing.T) {
	pt := testdisk.MakeFakeBtrfsPartitionTable("/")
	_, actual, err := GenImageKernelOptions(pt, false)
	assert.NoError(t, err)
	assert.Equal(t, []string{"rootflags=subvol=root"}, actual)
}

func TestGenImageKernelOptionsBtrfsNotRootCmdlineGenerated(t *testing.T) {
	pt := testdisk.MakeFakeBtrfsPartitionTable("/var")
	_, kopts, err := GenImageKernelOptions(pt, false)
	assert.EqualError(t, err, "root filesystem must be defined for kernel-cmdline stage, this is a programming error")
	assert.Equal(t, len(kopts), 0)
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
				Size:     fmt.Sprintf("%d", 10*datasizes.GiB),
			},
		},
		{
			Type: "org.osbuild.sfdisk",
			Options: &SfdiskStageOptions{
				Label: "gpt",
				Partitions: []SfdiskPartition{
					{
						Size: 1 * datasizes.GiB / 512,
					},
					{
						Start: 1 * datasizes.GiB / 512,
						Size:  9 * datasizes.GiB / 512,
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
						Size:     1 * datasizes.GiB / 512,
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
						Start:    1 * datasizes.GiB / 512,
						Size:     9 * datasizes.GiB / 512,
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
				"device": {
					Type: "org.osbuild.loopback",
					Options: &LoopbackDeviceOptions{
						Filename: filename,
						Start:    1 * datasizes.GiB / 512,
						Size:     9 * datasizes.GiB / 512,
						Lock:     true,
					},
				},
			},
			Mounts: []Mount{
				{
					Name:    "volume",
					Type:    "org.osbuild.btrfs",
					Source:  "device",
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
