package osbuild

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/osbuild/blueprint/pkg/blueprint"
	"github.com/osbuild/images/internal/testdisk"
	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/datasizes"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/disk/partition"
	"github.com/stretchr/testify/assert"
)

// collectUUIDs returns the filesystem UUID for each mountpoint in the
// partition table. It also returns the UUID for the LUKS container keyed by
// 'luks'.
func collectUUIDs(pt *disk.PartitionTable) map[string]string {
	uuids := make(map[string]string)
	findUUIDs := func(e disk.Entity, path []disk.Entity) error {
		switch ent := e.(type) {
		case *disk.LUKSContainer:
			uuids["luks"] = ent.UUID
		case *disk.Filesystem:
			uuids[ent.Mountpoint] = ent.UUID
		case *disk.BtrfsSubvolume:
			uuids[ent.Mountpoint] = ent.GetFSSpec().UUID
		}

		return nil
	}
	_ = pt.ForEachEntity(findUUIDs)
	return uuids
}

func TestGenImageKernelOptions(t *testing.T) {
	assert := assert.New(t)

	// math/rand is good enough in this case
	/* #nosec G404 */
	rng := rand.New(rand.NewSource(13))

	luks_lvm := testPartitionTables["luks+lvm"]

	pt, err := disk.NewPartitionTable(&luks_lvm, []blueprint.FilesystemCustomization{}, 0, partition.AutoLVMPartitioningMode, arch.ARCH_X86_64, make(map[string]uint64), "", rng)
	assert.NoError(err)

	uuids := collectUUIDs(pt)
	assert.NotEmpty(uuids["/"], "Could not find root filesystem")
	assert.NotEmpty(uuids["luks"], "Could not find LUKS container")
	rootUUID, cmdline, err := GenImageKernelOptions(pt, false)
	assert.NoError(err)

	assert.Equal(rootUUID, uuids["/"])
	assert.Subset(cmdline, []string{"luks.uuid=" + uuids["luks"]})
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
	actualStages := GenImagePrepareStages(pt, filename, PTSfdisk, "build")

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

// addMountOptions appends mount options to filesystems based on their
// mountpoint.
func addMountOptions(pt *disk.PartitionTable, fsopts map[string]string) {

	addOptions := func(mnt disk.Mountable, path []disk.Entity) error {
		mntpt := mnt.GetMountpoint()
		if opts, ok := fsopts[mntpt]; ok {
			switch ent := mnt.(type) {
			case *disk.Filesystem:
				if ent.FSTabOptions != "" {
					opts = "," + opts
				}
				ent.FSTabOptions += opts
			case *disk.BtrfsSubvolume:
				// NOTE: we don't support mountopts on btrfs (sub)volumes???
			default:
				panic("please update the addMountOptions() test utility function to support all mountables")
			}
		}
		return nil
	}
	_ = pt.ForEachMountable(addOptions)
}

func TestGenImageKernelOptionsMountUnitsPlain(t *testing.T) {
	assert := assert.New(t)

	pt := testdisk.MakeFakePartitionTable("/", "/home")
	mntOpts := map[string]string{
		"/": "noatime,discard",
	}
	addMountOptions(pt, mntOpts)

	uuids := collectUUIDs(pt)
	assert.Len(uuids, 2)

	rootUUID, cmdline, err := GenImageKernelOptions(pt, true)
	assert.NoError(err)

	assert.Equal(rootUUID, uuids["/"])
	assert.Contains(cmdline, "rootflags=noatime,discard")
}

func TestGenImageKernelOptionsMountUnitsPlainWithUsr(t *testing.T) {
	assert := assert.New(t)

	pt := testdisk.MakeFakePartitionTable("/", "/home", "/usr")
	mntOpts := map[string]string{
		"/":    "noatime,discard",
		"/usr": "discard,noatime",
	}
	addMountOptions(pt, mntOpts)

	uuids := collectUUIDs(pt)
	assert.Len(uuids, 3)

	rootUUID, cmdline, err := GenImageKernelOptions(pt, true)
	assert.NoError(err)

	assert.Equal(rootUUID, uuids["/"])
	assert.Contains(cmdline, "rootflags=noatime,discard")
	assert.Contains(cmdline, "mount.usrflags=discard,noatime")
	assert.Contains(cmdline, "mount.usr=UUID="+uuids["/usr"])
	assert.Contains(cmdline, "mount.usrfstype=ext4")
}

func TestGenImageKernelOptionsMountUnitsBtrfs(t *testing.T) {
	assert := assert.New(t)

	pt := testdisk.MakeFakeBtrfsPartitionTable("/", "/home")

	uuids := collectUUIDs(pt)
	assert.Len(uuids, 2)

	rootUUID, cmdline, err := GenImageKernelOptions(pt, true)
	assert.NoError(err)

	assert.Equal(rootUUID, uuids["/"])
	// NOTE: these are statically defined for btrfs subvolumes - this will
	// change
	assert.Contains(cmdline, "rootflags=subvol=root,compress=zstd:1")
}

func TestGenImageKernelOptionsMountUnitsBtrfsWithUsr(t *testing.T) {
	assert := assert.New(t)

	pt := testdisk.MakeFakeBtrfsPartitionTable("/", "/home", "/usr")
	mntOpts := map[string]string{
		"/":    "noatime,discard",
		"/usr": "discard,noatime",
	}
	addMountOptions(pt, mntOpts)

	uuids := collectUUIDs(pt)
	assert.Len(uuids, 3)

	rootUUID, cmdline, err := GenImageKernelOptions(pt, true)
	assert.NoError(err)

	assert.Equal(rootUUID, uuids["/"])
	// NOTE: these are statically defined for btrfs subvolumes - this will
	// change
	assert.Contains(cmdline, "rootflags=subvol=root,compress=zstd:1")
	assert.Contains(cmdline, "mount.usrflags=subvol=/usr,compress=zstd:1")

	assert.Contains(cmdline, "mount.usr=UUID="+uuids["/usr"])
}

func TestGenImageKernelOptionsMountUnitsLVM(t *testing.T) {
	assert := assert.New(t)

	pt := testdisk.MakeFakeLVMPartitionTable("/", "/home")
	mntOpts := map[string]string{
		"/": "noatime,discard",
	}
	addMountOptions(pt, mntOpts)

	uuids := collectUUIDs(pt)
	assert.Len(uuids, 2)

	rootUUID, cmdline, err := GenImageKernelOptions(pt, true)
	assert.NoError(err)

	assert.Equal(rootUUID, uuids["/"])
	assert.Contains(cmdline, "rootflags=noatime,discard")
}

func TestGenImageKernelOptionsMountUnitsLVMWithUsr(t *testing.T) {
	assert := assert.New(t)

	pt := testdisk.MakeFakeLVMPartitionTable("/", "/home", "/usr")
	mntOpts := map[string]string{
		"/":    "noatime,discard",
		"/usr": "discard,noatime",
	}
	addMountOptions(pt, mntOpts)

	uuids := collectUUIDs(pt)
	assert.Len(uuids, 3)

	rootUUID, cmdline, err := GenImageKernelOptions(pt, true)
	assert.NoError(err)

	assert.Equal(rootUUID, uuids["/"])
	assert.Contains(cmdline, "rootflags=noatime,discard")
	assert.Contains(cmdline, "mount.usrflags=discard,noatime")
	assert.Contains(cmdline, "mount.usr=UUID="+uuids["/usr"])
	assert.Contains(cmdline, "mount.usrfstype=xfs")
}
