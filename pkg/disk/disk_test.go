package disk

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/blueprint"
)

const (
	KiB = common.KiB
	MiB = common.MiB
	GiB = common.GiB
)

func TestDisk_AlignUp(t *testing.T) {

	pt := PartitionTable{}
	firstAligned := DefaultGrainBytes

	tests := []struct {
		size uint64
		want uint64
	}{
		{0, 0},
		{1, firstAligned},
		{firstAligned - 1, firstAligned},
		{firstAligned, firstAligned}, // grain is already aligned => no change
		{firstAligned / 2, firstAligned},
		{firstAligned + 1, firstAligned * 2},
	}

	for _, tt := range tests {
		got := pt.AlignUp(tt.size)
		assert.Equal(t, tt.want, got, "Expected %d, got %d", tt.want, got)
	}
}

func TestDisk_DynamicallyResizePartitionTable(t *testing.T) {
	mountpoints := []blueprint.FilesystemCustomization{
		{
			MinSize:    2 * GiB,
			Mountpoint: "/usr",
		},
	}
	pt := PartitionTable{
		UUID: "D209C89E-EA5E-4FBD-B161-B461CCE297E0",
		Type: "gpt",
		Partitions: []Partition{
			{
				Size:     2048,
				Bootable: true,
				Type:     BIOSBootPartitionGUID,
				UUID:     BIOSBootPartitionUUID,
			},
			{
				Type: FilesystemDataGUID,
				UUID: RootPartitionUUID,
				Payload: &Filesystem{
					Type:         "xfs",
					Label:        "root",
					Mountpoint:   "/",
					FSTabOptions: "defaults",
					FSTabFreq:    0,
					FSTabPassNo:  0,
				},
			},
		},
	}
	var expectedSize uint64 = 2 * GiB
	// math/rand is good enough in this case
	/* #nosec G404 */
	rng := rand.New(rand.NewSource(0))
	newpt, err := NewPartitionTable(&pt, mountpoints, 1024, RawPartitioningMode, nil, rng)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, newpt.Size, expectedSize)
}

var testPartitionTables = map[string]PartitionTable{
	"plain": {
		UUID: "D209C89E-EA5E-4FBD-B161-B461CCE297E0",
		Type: "gpt",
		Partitions: []Partition{
			{
				Size:     1 * MiB,
				Bootable: true,
				Type:     BIOSBootPartitionGUID,
				UUID:     BIOSBootPartitionUUID,
			},
			{
				Size: 200 * MiB,
				Type: EFISystemPartitionGUID,
				UUID: EFISystemPartitionUUID,
				Payload: &Filesystem{
					Type:         "vfat",
					UUID:         EFIFilesystemUUID,
					Mountpoint:   "/boot/efi",
					Label:        "EFI-SYSTEM",
					FSTabOptions: "defaults,uid=0,gid=0,umask=077,shortname=winnt",
					FSTabFreq:    0,
					FSTabPassNo:  2,
				},
			},
			{
				Size: 500 * MiB,
				Type: FilesystemDataGUID,
				UUID: FilesystemDataUUID,
				Payload: &Filesystem{
					Type:         "xfs",
					Mountpoint:   "/boot",
					Label:        "boot",
					FSTabOptions: "defaults",
					FSTabFreq:    0,
					FSTabPassNo:  0,
				},
			},
			{
				Type: FilesystemDataGUID,
				UUID: RootPartitionUUID,
				Payload: &Filesystem{
					Type:         "xfs",
					Label:        "root",
					Mountpoint:   "/",
					FSTabOptions: "defaults",
					FSTabFreq:    0,
					FSTabPassNo:  0,
				},
			},
		},
	},

	"plain-noboot": {
		UUID: "D209C89E-EA5E-4FBD-B161-B461CCE297E0",
		Type: "gpt",
		Partitions: []Partition{
			{
				Size:     1 * MiB,
				Bootable: true,
				Type:     BIOSBootPartitionGUID,
				UUID:     BIOSBootPartitionUUID,
			},
			{
				Size: 200 * MiB,
				Type: EFISystemPartitionGUID,
				UUID: EFISystemPartitionUUID,
				Payload: &Filesystem{
					Type:         "vfat",
					UUID:         EFIFilesystemUUID,
					Mountpoint:   "/boot/efi",
					Label:        "EFI-SYSTEM",
					FSTabOptions: "defaults,uid=0,gid=0,umask=077,shortname=winnt",
					FSTabFreq:    0,
					FSTabPassNo:  2,
				},
			},
			{
				Type: FilesystemDataGUID,
				UUID: RootPartitionUUID,
				Payload: &Filesystem{
					Type:         "xfs",
					Label:        "root",
					Mountpoint:   "/",
					FSTabOptions: "defaults",
					FSTabFreq:    0,
					FSTabPassNo:  0,
				},
			},
		},
	},

	"luks": {
		UUID: "D209C89E-EA5E-4FBD-B161-B461CCE297E0",
		Type: "gpt",
		Partitions: []Partition{
			{
				Size:     1 * MiB,
				Bootable: true,
				Type:     BIOSBootPartitionGUID,
				UUID:     BIOSBootPartitionUUID,
			},
			{
				Size: 200 * MiB,
				Type: EFISystemPartitionGUID,
				UUID: EFISystemPartitionUUID,
				Payload: &Filesystem{
					Type:         "vfat",
					UUID:         EFIFilesystemUUID,
					Mountpoint:   "/boot/efi",
					Label:        "EFI-SYSTEM",
					FSTabOptions: "defaults,uid=0,gid=0,umask=077,shortname=winnt",
					FSTabFreq:    0,
					FSTabPassNo:  2,
				},
			},
			{
				Size: 500 * MiB,
				Type: FilesystemDataGUID,
				UUID: FilesystemDataUUID,
				Payload: &Filesystem{
					Type:         "xfs",
					Mountpoint:   "/boot",
					Label:        "boot",
					FSTabOptions: "defaults",
					FSTabFreq:    0,
					FSTabPassNo:  0,
				},
			},
			{
				Type: FilesystemDataGUID,
				UUID: RootPartitionUUID,
				Payload: &LUKSContainer{
					UUID:  "",
					Label: "crypt_root",
					Payload: &Filesystem{
						Type:         "xfs",
						Label:        "root",
						Mountpoint:   "/",
						FSTabOptions: "defaults",
						FSTabFreq:    0,
						FSTabPassNo:  0,
					},
				},
			},
		},
	},
	"luks+lvm": {
		UUID: "D209C89E-EA5E-4FBD-B161-B461CCE297E0",
		Type: "gpt",
		Partitions: []Partition{
			{
				Size:     1 * MiB,
				Bootable: true,
				Type:     BIOSBootPartitionGUID,
				UUID:     BIOSBootPartitionUUID,
			},
			{
				Size: 200 * MiB,
				Type: EFISystemPartitionGUID,
				UUID: EFISystemPartitionUUID,
				Payload: &Filesystem{
					Type:         "vfat",
					UUID:         EFIFilesystemUUID,
					Mountpoint:   "/boot/efi",
					Label:        "EFI-SYSTEM",
					FSTabOptions: "defaults,uid=0,gid=0,umask=077,shortname=winnt",
					FSTabFreq:    0,
					FSTabPassNo:  2,
				},
			},
			{
				Size: 500 * MiB,
				Type: FilesystemDataGUID,
				UUID: FilesystemDataUUID,
				Payload: &Filesystem{
					Type:         "xfs",
					Mountpoint:   "/boot",
					Label:        "boot",
					FSTabOptions: "defaults",
					FSTabFreq:    0,
					FSTabPassNo:  0,
				},
			},
			{
				Type: FilesystemDataGUID,
				UUID: RootPartitionUUID,
				Size: 5 * GiB,
				Payload: &LUKSContainer{
					UUID: "",
					Payload: &LVMVolumeGroup{
						Name:        "",
						Description: "",
						LogicalVolumes: []LVMLogicalVolume{
							{
								Size: 2 * GiB,
								Payload: &Filesystem{
									Type:         "xfs",
									Label:        "root",
									Mountpoint:   "/",
									FSTabOptions: "defaults",
									FSTabFreq:    0,
									FSTabPassNo:  0,
								},
							},
							{
								Size: 2 * GiB,
								Payload: &Filesystem{
									Type:         "xfs",
									Label:        "root",
									Mountpoint:   "/home",
									FSTabOptions: "defaults",
									FSTabFreq:    0,
									FSTabPassNo:  0,
								},
							},
						},
					},
				},
			},
		},
	},
	"btrfs": {
		UUID: "D209C89E-EA5E-4FBD-B161-B461CCE297E0",
		Type: "gpt",
		Partitions: []Partition{
			{
				Size:     1 * MiB,
				Bootable: true,
				Type:     BIOSBootPartitionGUID,
				UUID:     BIOSBootPartitionUUID,
			},
			{
				Size: 200 * MiB,
				Type: EFISystemPartitionGUID,
				UUID: EFISystemPartitionUUID,
				Payload: &Filesystem{
					Type:         "vfat",
					UUID:         EFIFilesystemUUID,
					Mountpoint:   "/boot/efi",
					Label:        "EFI-SYSTEM",
					FSTabOptions: "defaults,uid=0,gid=0,umask=077,shortname=winnt",
					FSTabFreq:    0,
					FSTabPassNo:  2,
				},
			},
			{
				Size: 500 * MiB,
				Type: FilesystemDataGUID,
				UUID: FilesystemDataUUID,
				Payload: &Filesystem{
					Type:         "xfs",
					Mountpoint:   "/boot",
					Label:        "boot",
					FSTabOptions: "defaults",
					FSTabFreq:    0,
					FSTabPassNo:  0,
				},
			},
			{
				Type: FilesystemDataGUID,
				UUID: RootPartitionUUID,
				Size: 10 * GiB,
				Payload: &Btrfs{
					UUID:       "",
					Label:      "",
					Mountpoint: "",
					Subvolumes: []BtrfsSubvolume{
						{
							Size:       0,
							Mountpoint: "/",
							GroupID:    0,
						},
						{
							Size:       5 * GiB,
							Mountpoint: "/var",
							GroupID:    0,
						},
					},
				},
			},
		},
	},
}

var testBlueprints = map[string][]blueprint.FilesystemCustomization{
	"bp1": {
		{
			Mountpoint: "/",
			MinSize:    10 * GiB,
		},
		{
			Mountpoint: "/home",
			MinSize:    20 * GiB,
		},
		{
			Mountpoint: "/opt",
			MinSize:    7 * GiB,
		},
	},
	"bp2": {
		{
			Mountpoint: "/opt",
			MinSize:    7 * GiB,
		},
		{
			Type:    blueprint.FilesystemTypeSwap,
			MinSize: 7 * GiB,
		},
	},
	"small": {
		{
			Mountpoint: "/opt",
			MinSize:    20 * MiB,
		},
		{
			Mountpoint: "/home",
			MinSize:    500 * MiB,
		},
	},
	"empty": nil,
}

func TestDisk_ForEachEntity(t *testing.T) {

	count := 0

	plain := testPartitionTables["plain"]
	err := plain.ForEachEntity(func(e Entity, path []Entity) error {
		assert.NotNil(t, e)
		assert.NotNil(t, path)

		count += 1
		return nil
	})

	assert.NoError(t, err)

	// PartitionTable, 4 partitions, 3 filesystems -> 8 entities
	assert.Equal(t, 8, count)
}

// blueprintApplied checks if the blueprint was applied correctly
// returns nil if the blueprint was applied correctly, an error otherwise
func blueprintApplied(pt *PartitionTable, bp []blueprint.FilesystemCustomization) error {

	// finds an entity in the partition table representing the desired custom mountpoint
	// and returns an entity path to it
	customMountpointPath := func(mnt blueprint.FilesystemCustomization) []Entity {
		if mnt.Type == blueprint.FilesystemTypeSwap {
			isSwap := func(e Entity) bool {
				if fs, ok := e.(*Filesystem); ok {
					return fs.Type == "swap"
				}
				return false
			}
			return entityPath(pt, isSwap)
		}
		return entityPathForMountpoint(pt, mnt.Mountpoint)
	}

	for _, mnt := range bp {
		path := customMountpointPath(mnt)
		if path == nil {
			return fmt.Errorf("mountpoint %s (type %s) not found", mnt.Mountpoint, mnt.Type)
		}
		for idx, ent := range path {
			if sz, ok := ent.(Sizeable); ok {
				if sz.GetSize() < mnt.MinSize {
					return fmt.Errorf("entity %d in the path from %s is smaller (%d) than the requested minsize %d", idx, mnt.Mountpoint, sz.GetSize(), mnt.MinSize)
				}
			}
		}
	}

	return nil
}

func TestCreatePartitionTable(t *testing.T) {
	assert := assert.New(t)

	sizeCheckCB := func(mnt Mountable, path []Entity) error {
		if strings.HasPrefix(mnt.GetMountpoint(), "/boot") {
			// /boot and subdirectories is exempt from this rule
			return nil
		}
		// go up the path and check every sizeable
		for idx, ent := range path {
			if sz, ok := ent.(Sizeable); ok {
				size := sz.GetSize()
				if size < 1*GiB {
					return fmt.Errorf("entity %d in the path from %s is smaller than the minimum 1 GiB (%d)", idx, mnt.GetMountpoint(), size)
				}
			}
		}
		return nil
	}

	sumSizes := func(bp []blueprint.FilesystemCustomization) (sum uint64) {
		for _, mnt := range bp {
			sum += mnt.MinSize
		}
		return sum
	}
	// math/rand is good enough in this case
	/* #nosec G404 */
	rng := rand.New(rand.NewSource(13))
	for ptName := range testPartitionTables {
		pt := testPartitionTables[ptName]
		for bpName, bp := range testBlueprints {
			ptMode := RawPartitioningMode
			if ptName == "luks+lvm" {
				ptMode = AutoLVMPartitioningMode
			}
			mpt, err := NewPartitionTable(&pt, bp, uint64(13*MiB), ptMode, nil, rng)
			require.NoError(t, err, "Partition table generation failed: PT %q BP %q (%s)", ptName, bpName, err)
			assert.NotNil(mpt, "Partition table generation failed: PT %q BP %q (nil partition table)", ptName, bpName)
			assert.Greater(mpt.GetSize(), sumSizes(bp))

			assert.NotNil(mpt.Type, "Partition table generation failed: PT %q BP %q (nil partition table type)", ptName, bpName)

			mnt := pt.FindMountable("/")
			assert.NotNil(mnt, "PT %q BP %q: failed to find root mountable", ptName, bpName)

			assert.NoError(mpt.ForEachMountable(sizeCheckCB))
			assert.NoError(blueprintApplied(mpt, bp), "PT %q BP %q: blueprint check failed", ptName, bpName)
		}
	}
}

func TestCreatePartitionTableLVMify(t *testing.T) {
	assert := assert.New(t)
	// math/rand is good enough in this case
	/* #nosec G404 */
	rng := rand.New(rand.NewSource(13))
	for bpName, tbp := range testBlueprints {
		for ptName := range testPartitionTables {
			pt := testPartitionTables[ptName]

			if tbp != nil && (ptName == "btrfs" || ptName == "luks") {
				_, err := NewPartitionTable(&pt, tbp, uint64(13*MiB), AutoLVMPartitioningMode, nil, rng)
				assert.Error(err, "PT %q BP %q: should return an error with LVMPartitioningMode", ptName, bpName)
				continue
			}

			mpt, err := NewPartitionTable(&pt, tbp, uint64(13*MiB), AutoLVMPartitioningMode, nil, rng)
			assert.NoError(err, "PT %q BP %q: Partition table generation failed: (%s)", ptName, bpName, err)

			rootPath := entityPathForMountpoint(mpt, "/")
			if rootPath == nil {
				panic(fmt.Sprintf("PT %q BP %q: no root mountpoint", ptName, bpName))
			}

			bootPath := entityPathForMountpoint(mpt, "/boot")
			if tbp != nil && bootPath == nil {
				panic(fmt.Sprintf("PT %q BP %q: no boot mountpoint", ptName, bpName))
			}

			if tbp != nil {
				parent := rootPath[1]
				_, ok := parent.(*LVMLogicalVolume)
				assert.True(ok, "PT %q BP %q: root's parent (%q) is not an LVM logical volume", ptName, bpName, parent)
			}
			assert.NoError(blueprintApplied(mpt, tbp), "PT %q BP %q: blueprint check failed", ptName, bpName)
		}
	}
}

func TestCreatePartitionTableBtrfsify(t *testing.T) {
	assert := assert.New(t)
	// math/rand is good enough in this case
	/* #nosec G404 */
	rng := rand.New(rand.NewSource(13))
	for bpName, tbp := range testBlueprints {
		for ptName := range testPartitionTables {
			pt := testPartitionTables[ptName]

			if ptName == "auto-lvm" || ptName == "luks" || ptName == "luks+lvm" {
				_, err := NewPartitionTable(&pt, tbp, uint64(13*MiB), BtrfsPartitioningMode, nil, rng)
				assert.Error(err, "PT %q BP %q: should return an error with BtrfsPartitioningMode", ptName, bpName)
				continue
			}

			mpt, err := NewPartitionTable(&pt, tbp, uint64(13*MiB), BtrfsPartitioningMode, nil, rng)
			assert.NoError(err, "PT %q BP %q: Partition table generation failed: (%s)", ptName, bpName, err)

			rootPath := entityPathForMountpoint(mpt, "/")
			if rootPath == nil {
				panic(fmt.Sprintf("PT %q BP %q: no root mountpoint", ptName, bpName))
			}

			bootPath := entityPathForMountpoint(mpt, "/boot")
			if tbp != nil && bootPath == nil {
				panic(fmt.Sprintf("PT %q BP %q: no boot mountpoint", ptName, bpName))
			}

			if tbp != nil {
				parent := rootPath[1]
				_, ok := parent.(*Btrfs)
				assert.True(ok, "PT %q BP %q: root's parent (%+v) is not an btrfs volume but %T", ptName, bpName, parent, parent)
			}
			assert.NoError(blueprintApplied(mpt, tbp), "PT %q BP %q: blueprint check failed", ptName, bpName)
		}
	}
}

func TestCreatePartitionTableLVMOnly(t *testing.T) {
	assert := assert.New(t)
	// math/rand is good enough in this case
	/* #nosec G404 */
	rng := rand.New(rand.NewSource(13))
	for bpName, tbp := range testBlueprints {
		for ptName := range testPartitionTables {
			pt := testPartitionTables[ptName]

			if ptName == "btrfs" || ptName == "luks" {
				_, err := NewPartitionTable(&pt, tbp, uint64(13*MiB), LVMPartitioningMode, nil, rng)
				assert.Error(err, "PT %q BP %q: should return an error with LVMPartitioningMode", ptName, bpName)
				continue
			}

			mpt, err := NewPartitionTable(&pt, tbp, uint64(13*MiB), LVMPartitioningMode, nil, rng)
			require.NoError(t, err, "PT %q BP %q: Partition table generation failed: (%s)", ptName, bpName, err)

			rootPath := entityPathForMountpoint(mpt, "/")
			if rootPath == nil {
				panic(fmt.Sprintf("PT %q BP %q: no root mountpoint", ptName, bpName))
			}

			bootPath := entityPathForMountpoint(mpt, "/boot")
			if tbp != nil && bootPath == nil {
				panic(fmt.Sprintf("PT %q BP %q: no boot mountpoint", ptName, bpName))
			}

			// root should always be on a LVM
			rootParent := rootPath[1]
			{
				_, ok := rootParent.(*LVMLogicalVolume)
				assert.True(ok, "PT %q BP %q: root's parent (%+v) is not an LVM logical volume", ptName, bpName, rootParent)
			}

			// check logical volume sizes against blueprint
			var lvsum uint64
			for _, mnt := range tbp {
				if mnt.Mountpoint == "/boot" || mnt.Type == blueprint.FilesystemTypeSwap {
					// not on LVM or swap; skipping
					continue
				}
				mntPath := entityPathForMountpoint(mpt, mnt.Mountpoint)
				mntParent := mntPath[1]
				mntLV, ok := mntParent.(*LVMLogicalVolume) // the partition's parent should be the logical volume
				assert.True(ok, "PT %q BP %q: %s's parent (%+v) is not an LVM logical volume", ptName, bpName, mnt.Mountpoint, mntParent)
				assert.GreaterOrEqualf(mntLV.Size, mnt.MinSize, "PT %q BP %q: %s's size (%d) is smaller than the requested minsize (%d)", ptName, bpName, mnt.Mountpoint, mntLV.Size, mnt.MinSize)
				lvsum += mntLV.Size
			}

			// root LV's parent should be the VG
			lvParent := rootPath[2]
			{
				_, ok := lvParent.(*LVMVolumeGroup)
				assert.True(ok, "PT %q BP %q: root LV's parent (%+v) is not an LVM volume group", ptName, bpName, lvParent)
			}

			// the root partition is the second to last entity in the path (the last is the partition table itself)
			rootTop := rootPath[len(rootPath)-2]
			vgPart, ok := rootTop.(*Partition)

			// if the VG is in a LUKS container, check that there's enough space for the header too
			{
				vgParent := rootPath[3]
				luksContainer, ok := vgParent.(*LUKSContainer)
				if ok {
					// this isn't the lvsum anymore, but the lvsum + the luks
					// header, which should regardless be equal to or smaller
					// than the partition
					lvsum += luksContainer.MetadataSize()
				}
			}

			assert.True(ok, "PT %q BP %q: root VG top level entity (%+v) is not a partition", ptName, bpName, rootTop)
			assert.GreaterOrEqualf(vgPart.Size, lvsum, "PT %q BP %q: VG partition's size (%d) is smaller than the sum of logical volumes (%d)", ptName, bpName, vgPart.Size, lvsum)

			assert.NoError(blueprintApplied(mpt, tbp), "PT %q BP %q: blueprint check failed", ptName, bpName)
		}
	}
}

func TestMinimumSizes(t *testing.T) {
	assert := assert.New(t)

	// math/rand is good enough in this case
	/* #nosec G404 */
	rng := rand.New(rand.NewSource(13))
	pt := testPartitionTables["plain"]

	type testCase struct {
		Blueprint        []blueprint.FilesystemCustomization
		ExpectedMinSizes map[string]uint64
	}

	testCases := []testCase{
		{ // specify small /usr -> / and /usr get default size
			Blueprint: []blueprint.FilesystemCustomization{
				{
					Mountpoint: "/usr",
					MinSize:    1 * MiB,
				},
			},
			ExpectedMinSizes: map[string]uint64{
				"/usr": 2 * GiB,
				"/":    1 * GiB,
			},
		},
		{ // specify small / and /usr -> / and /usr get default size
			Blueprint: []blueprint.FilesystemCustomization{
				{
					Mountpoint: "/",
					MinSize:    1 * MiB,
				},
				{
					Mountpoint: "/usr",
					MinSize:    1 * KiB,
				},
			},
			ExpectedMinSizes: map[string]uint64{
				"/usr": 2 * GiB,
				"/":    1 * GiB,
			},
		},
		{ // big /usr -> / gets default size
			Blueprint: []blueprint.FilesystemCustomization{
				{
					Mountpoint: "/usr",
					MinSize:    10 * GiB,
				},
			},
			ExpectedMinSizes: map[string]uint64{
				"/usr": 10 * GiB,
				"/":    1 * GiB,
			},
		},
		{
			Blueprint: []blueprint.FilesystemCustomization{
				{
					Mountpoint: "/",
					MinSize:    10 * GiB,
				},
				{
					Mountpoint: "/home",
					MinSize:    1 * MiB,
				},
			},
			ExpectedMinSizes: map[string]uint64{
				"/":     10 * GiB,
				"/home": 1 * GiB,
			},
		},
		{ // no separate /usr and no size for / -> / gets sum of default sizes for / and /usr
			Blueprint: []blueprint.FilesystemCustomization{
				{
					Mountpoint: "/opt",
					MinSize:    10 * GiB,
				},
			},
			ExpectedMinSizes: map[string]uint64{
				"/opt": 10 * GiB,
				"/":    3 * GiB,
			},
		},
	}

	for idx, tc := range testCases {
		{ // without LVM
			mpt, err := NewPartitionTable(&pt, tc.Blueprint, uint64(3*GiB), RawPartitioningMode, nil, rng)
			assert.NoError(err)
			for mnt, minSize := range tc.ExpectedMinSizes {
				path := entityPathForMountpoint(mpt, mnt)
				assert.NotNil(path, "[%d] mountpoint %q not found", idx, mnt)
				parent := path[1]
				part, ok := parent.(*Partition)
				assert.True(ok, "%q parent (%v) is not a partition", mnt, parent)
				assert.GreaterOrEqual(part.GetSize(), minSize,
					"[%d] %q size %d should be greater or equal to %d", idx, mnt, part.GetSize(), minSize)
			}
		}

		{ // with LVM
			mpt, err := NewPartitionTable(&pt, tc.Blueprint, uint64(3*GiB), AutoLVMPartitioningMode, nil, rng)
			assert.NoError(err)
			for mnt, minSize := range tc.ExpectedMinSizes {
				path := entityPathForMountpoint(mpt, mnt)
				assert.NotNil(path, "[%d] mountpoint %q not found", idx, mnt)
				parent := path[1]
				part, ok := parent.(*LVMLogicalVolume)
				assert.True(ok, "[%d] %q parent (%v) is not an LVM logical volume", idx, mnt, parent)
				assert.GreaterOrEqual(part.GetSize(), minSize,
					"[%d] %q size %d should be greater or equal to %d", idx, mnt, part.GetSize(), minSize)
			}
		}
	}
}

func TestLVMExtentAlignment(t *testing.T) {
	assert := assert.New(t)

	// math/rand is good enough in this case
	/* #nosec G404 */
	rng := rand.New(rand.NewSource(13))
	pt := testPartitionTables["plain"]

	type testCase struct {
		Blueprint     []blueprint.FilesystemCustomization
		ExpectedSizes map[string]uint64
	}

	testCases := []testCase{
		{
			Blueprint: []blueprint.FilesystemCustomization{
				{
					Mountpoint: "/var",
					MinSize:    1*GiB + 1,
				},
			},
			ExpectedSizes: map[string]uint64{
				"/var": 1*GiB + LVMDefaultExtentSize,
			},
		},
		{
			// lots of mount points in /var
			// https://bugzilla.redhat.com/show_bug.cgi?id=2141738
			Blueprint: []blueprint.FilesystemCustomization{
				{
					Mountpoint: "/",
					MinSize:    32000000000,
				},
				{
					Mountpoint: "/var",
					MinSize:    4096000000,
				},
				{
					Mountpoint: "/var/log",
					MinSize:    4096000000,
				},
			},
			ExpectedSizes: map[string]uint64{
				"/":        32002539520,
				"/var":     3908 * MiB,
				"/var/log": 3908 * MiB,
			},
		},
		{
			Blueprint: []blueprint.FilesystemCustomization{
				{
					Mountpoint: "/",
					MinSize:    32 * GiB,
				},
				{
					Mountpoint: "/var",
					MinSize:    4 * GiB,
				},
				{
					Mountpoint: "/var/log",
					MinSize:    4 * GiB,
				},
			},
			ExpectedSizes: map[string]uint64{
				"/":        32 * GiB,
				"/var":     4 * GiB,
				"/var/log": 4 * GiB,
			},
		},
	}

	for idx, tc := range testCases {
		mpt, err := NewPartitionTable(&pt, tc.Blueprint, uint64(3*GiB), AutoLVMPartitioningMode, nil, rng)
		assert.NoError(err)
		for mnt, expSize := range tc.ExpectedSizes {
			path := entityPathForMountpoint(mpt, mnt)
			assert.NotNil(path, "[%d] mountpoint %q not found", idx, mnt)
			parent := path[1]
			part, ok := parent.(*LVMLogicalVolume)
			assert.True(ok, "[%d] %q parent (%v) is not an LVM logical volume", idx, mnt, parent)
			assert.Equal(part.GetSize(), expSize,
				"[%d] %q size %d should be equal to %d", idx, mnt, part.GetSize(), expSize)
		}
	}
}

func TestNewBootWithSizeLVMify(t *testing.T) {
	pt := testPartitionTables["plain-noboot"]
	assert := assert.New(t)

	// math/rand is good enough in this case
	/* #nosec G404 */
	rng := rand.New(rand.NewSource(13))

	custom := []blueprint.FilesystemCustomization{
		{
			Mountpoint: "/boot",
			MinSize:    700 * MiB,
		},
	}

	mpt, err := NewPartitionTable(&pt, custom, uint64(3*GiB), AutoLVMPartitioningMode, nil, rng)
	assert.NoError(err)

	for idx, c := range custom {
		mnt, minSize := c.Mountpoint, c.MinSize
		path := entityPathForMountpoint(mpt, mnt)
		assert.NotNil(path, "[%d] mountpoint %q not found", idx, mnt)
		parent := path[1]
		part, ok := parent.(*Partition)
		assert.True(ok, "%q parent (%v) is not a partition", mnt, parent)
		assert.GreaterOrEqual(part.GetSize(), minSize,
			"[%d] %q size %d should be greater or equal to %d", idx, mnt, part.GetSize(), minSize)
	}
}

func collectEntities(pt *PartitionTable) []Entity {
	entities := make([]Entity, 0)
	collector := func(ent Entity, path []Entity) error {
		entities = append(entities, ent)
		return nil
	}
	_ = pt.ForEachEntity(collector)
	return entities
}

func TestClone(t *testing.T) {
	for name := range testPartitionTables {
		basePT := testPartitionTables[name]
		baseEntities := collectEntities(&basePT)

		clonePT := basePT.Clone().(*PartitionTable)
		cloneEntities := collectEntities(clonePT)

		for idx := range baseEntities {
			for jdx := range cloneEntities {
				if fmt.Sprintf("%p", baseEntities[idx]) == fmt.Sprintf("%p", cloneEntities[jdx]) {
					t.Fatalf("found reference to same entity %#v in list of clones for partition table %q", baseEntities[idx], name)
				}
			}
		}
	}
}

func TestFindDirectoryPartition(t *testing.T) {
	assert := assert.New(t)
	usr := Partition{
		Type: FilesystemDataGUID,
		UUID: RootPartitionUUID,
		Payload: &Filesystem{
			Type:         "xfs",
			Label:        "root",
			Mountpoint:   "/usr",
			FSTabOptions: "defaults",
			FSTabFreq:    0,
			FSTabPassNo:  0,
		},
	}

	{
		pt := testPartitionTables["plain"]
		assert.Equal("/", pt.findDirectoryEntityPath("/opt")[0].(Mountable).GetMountpoint())
		assert.Equal("/boot/efi", pt.findDirectoryEntityPath("/boot/efi/Linux")[0].(Mountable).GetMountpoint())
		assert.Equal("/boot", pt.findDirectoryEntityPath("/boot/loader")[0].(Mountable).GetMountpoint())
		assert.Equal("/boot", pt.findDirectoryEntityPath("/boot")[0].(Mountable).GetMountpoint())

		ptMod := pt.Clone().(*PartitionTable)
		ptMod.Partitions = append(ptMod.Partitions, usr)
		assert.Equal("/", ptMod.findDirectoryEntityPath("/opt")[0].(Mountable).GetMountpoint())
		assert.Equal("/usr", ptMod.findDirectoryEntityPath("/usr")[0].(Mountable).GetMountpoint())
		assert.Equal("/usr", ptMod.findDirectoryEntityPath("/usr/bin")[0].(Mountable).GetMountpoint())

		// invalid dir should return nil
		assert.Nil(pt.findDirectoryEntityPath("invalid"))
	}

	{
		pt := testPartitionTables["plain-noboot"]
		assert.Equal("/", pt.findDirectoryEntityPath("/opt")[0].(Mountable).GetMountpoint())
		assert.Equal("/", pt.findDirectoryEntityPath("/boot")[0].(Mountable).GetMountpoint())
		assert.Equal("/", pt.findDirectoryEntityPath("/boot/loader")[0].(Mountable).GetMountpoint())

		ptMod := pt.Clone().(*PartitionTable)
		ptMod.Partitions = append(ptMod.Partitions, usr)
		assert.Equal("/", ptMod.findDirectoryEntityPath("/opt")[0].(Mountable).GetMountpoint())
		assert.Equal("/usr", ptMod.findDirectoryEntityPath("/usr")[0].(Mountable).GetMountpoint())
		assert.Equal("/usr", ptMod.findDirectoryEntityPath("/usr/bin")[0].(Mountable).GetMountpoint())

		// invalid dir should return nil
		assert.Nil(pt.findDirectoryEntityPath("invalid"))
	}

	{
		pt := testPartitionTables["luks"]
		assert.Equal("/", pt.findDirectoryEntityPath("/opt")[0].(Mountable).GetMountpoint())
		assert.Equal("/boot", pt.findDirectoryEntityPath("/boot")[0].(Mountable).GetMountpoint())
		assert.Equal("/boot", pt.findDirectoryEntityPath("/boot/loader")[0].(Mountable).GetMountpoint())

		ptMod := pt.Clone().(*PartitionTable)
		ptMod.Partitions = append(ptMod.Partitions, usr)
		assert.Equal("/", ptMod.findDirectoryEntityPath("/opt")[0].(Mountable).GetMountpoint())
		assert.Equal("/usr", ptMod.findDirectoryEntityPath("/usr")[0].(Mountable).GetMountpoint())
		assert.Equal("/usr", ptMod.findDirectoryEntityPath("/usr/bin")[0].(Mountable).GetMountpoint())

		// invalid dir should return nil
		assert.Nil(pt.findDirectoryEntityPath("invalid"))
	}

	{
		pt := testPartitionTables["luks+lvm"]
		assert.Equal("/", pt.findDirectoryEntityPath("/opt")[0].(Mountable).GetMountpoint())
		assert.Equal("/boot", pt.findDirectoryEntityPath("/boot")[0].(Mountable).GetMountpoint())
		assert.Equal("/boot", pt.findDirectoryEntityPath("/boot/loader")[0].(Mountable).GetMountpoint())

		ptMod := pt.Clone().(*PartitionTable)
		ptMod.Partitions = append(ptMod.Partitions, usr)
		assert.Equal("/", ptMod.findDirectoryEntityPath("/opt")[0].(Mountable).GetMountpoint())
		assert.Equal("/usr", ptMod.findDirectoryEntityPath("/usr")[0].(Mountable).GetMountpoint())
		assert.Equal("/usr", ptMod.findDirectoryEntityPath("/usr/bin")[0].(Mountable).GetMountpoint())

		// invalid dir should return nil
		assert.Nil(pt.findDirectoryEntityPath("invalid"))
	}

	{
		pt := testPartitionTables["btrfs"]
		assert.Equal("/", pt.findDirectoryEntityPath("/opt")[0].(Mountable).GetMountpoint())
		assert.Equal("/boot", pt.findDirectoryEntityPath("/boot")[0].(Mountable).GetMountpoint())
		assert.Equal("/boot", pt.findDirectoryEntityPath("/boot/loader")[0].(Mountable).GetMountpoint())

		ptMod := pt.Clone().(*PartitionTable)
		ptMod.Partitions = append(ptMod.Partitions, usr)
		assert.Equal("/", ptMod.findDirectoryEntityPath("/opt")[0].(Mountable).GetMountpoint())
		assert.Equal("/usr", ptMod.findDirectoryEntityPath("/usr")[0].(Mountable).GetMountpoint())
		assert.Equal("/usr", ptMod.findDirectoryEntityPath("/usr/bin")[0].(Mountable).GetMountpoint())

		// invalid dir should return nil
		assert.Nil(pt.findDirectoryEntityPath("invalid"))
	}

	{
		pt := PartitionTable{} // pt with no root should return nil
		assert.Nil(pt.findDirectoryEntityPath("/var"))
	}
}

func TestEnsureDirectorySizes(t *testing.T) {
	assert := assert.New(t)

	varSizes := map[string]uint64{
		"/var/lib":         uint64(3 * GiB),
		"/var/cache":       uint64(2 * GiB),
		"/var/log/journal": uint64(2 * GiB),
	}

	varAndHomeSizes := map[string]uint64{
		"/var/lib":         uint64(3 * GiB),
		"/var/cache":       uint64(2 * GiB),
		"/var/log/journal": uint64(2 * GiB),
		"/home/user/data":  uint64(10 * GiB),
	}

	{
		pt := testPartitionTables["plain"]
		pt = *pt.Clone().(*PartitionTable) // don't modify the original test data

		{
			// make sure we have the correct volume
			// guard against changes in the test pt
			rootPart := pt.Partitions[3]
			rootPayload := rootPart.Payload.(*Filesystem)

			assert.Equal("/", rootPayload.Mountpoint)
			assert.Equal(uint64(0), rootPart.Size)
		}

		{
			// add requirements for /var subdirs that are > 5 GiB
			pt.EnsureDirectorySizes(varSizes)
			rootPart := pt.Partitions[3]
			assert.Equal(uint64(7*GiB), rootPart.Size)

			// invalid
			assert.Panics(func() { pt.EnsureDirectorySizes(map[string]uint64{"invalid": uint64(300)}) })
		}
	}

	{
		pt := testPartitionTables["luks+lvm"]
		pt = *pt.Clone().(*PartitionTable) // don't modify the original test data

		{
			// make sure we have the correct volume
			// guard against changes in the test pt
			rootPart := pt.Partitions[3]
			rootLUKS := rootPart.Payload.(*LUKSContainer)
			rootVG := rootLUKS.Payload.(*LVMVolumeGroup)
			rootLV := rootVG.LogicalVolumes[0]
			rootFS := rootLV.Payload.(*Filesystem)
			homeLV := rootVG.LogicalVolumes[1]
			homeFS := homeLV.Payload.(*Filesystem)

			assert.Equal(uint64(5*GiB), rootPart.Size)
			assert.Equal("/", rootFS.Mountpoint)
			assert.Equal(uint64(2*GiB), rootLV.Size)
			assert.Equal("/home", homeFS.Mountpoint)
			assert.Equal(uint64(2*GiB), homeLV.Size)
		}

		{
			// add requirements for /var subdirs that are > 5 GiB
			pt.EnsureDirectorySizes(varAndHomeSizes)
			rootPart := pt.Partitions[3]
			rootLUKS := rootPart.Payload.(*LUKSContainer)
			rootVG := rootLUKS.Payload.(*LVMVolumeGroup)
			rootLV := rootVG.LogicalVolumes[0]
			homeLV := rootVG.LogicalVolumes[1]
			assert.Equal(uint64(17*GiB)+rootVG.MetadataSize()+rootLUKS.MetadataSize(), rootPart.Size)
			assert.Equal(uint64(7*GiB), rootLV.Size)
			assert.Equal(uint64(10*GiB), homeLV.Size)

			// invalid
			assert.Panics(func() { pt.EnsureDirectorySizes(map[string]uint64{"invalid": uint64(300)}) })
		}
	}

	{
		pt := testPartitionTables["btrfs"]
		pt = *pt.Clone().(*PartitionTable) // don't modify the original test data

		{
			// make sure we have the correct volume
			// guard against changes in the test pt
			rootPart := pt.Partitions[3]
			rootPayload := rootPart.Payload.(*Btrfs)
			assert.Equal("/", rootPayload.Subvolumes[0].Mountpoint)
			assert.Equal(uint64(0), rootPayload.Subvolumes[0].Size)
			assert.Equal("/var", rootPayload.Subvolumes[1].Mountpoint)
			assert.Equal(uint64(5*GiB), rootPayload.Subvolumes[1].Size)
		}

		{
			// add requirements for /var subdirs that are > 5 GiB
			pt.EnsureDirectorySizes(varSizes)
			rootPart := pt.Partitions[3]
			rootPayload := rootPart.Payload.(*Btrfs)
			assert.Equal(uint64(7*GiB), rootPayload.Subvolumes[1].Size)

			// invalid
			assert.Panics(func() { pt.EnsureDirectorySizes(map[string]uint64{"invalid": uint64(300)}) })
		}
	}

}

func TestMinimumSizesWithRequiredSizes(t *testing.T) {
	assert := assert.New(t)

	// math/rand is good enough in this case
	/* #nosec G404 */
	rng := rand.New(rand.NewSource(13))
	pt := testPartitionTables["plain"]

	type testCase struct {
		Blueprint        []blueprint.FilesystemCustomization
		ExpectedMinSizes map[string]uint64
	}

	testCases := []testCase{
		{ // specify small /usr -> / and /usr get default size
			Blueprint: []blueprint.FilesystemCustomization{
				{
					Mountpoint: "/usr",
					MinSize:    1 * MiB,
				},
			},
			ExpectedMinSizes: map[string]uint64{
				"/usr": 3 * GiB,
				"/":    1 * GiB,
			},
		},
		{ // specify small / and /usr -> / and /usr get default size
			Blueprint: []blueprint.FilesystemCustomization{
				{
					Mountpoint: "/",
					MinSize:    1 * MiB,
				},
				{
					Mountpoint: "/usr",
					MinSize:    1 * KiB,
				},
			},
			ExpectedMinSizes: map[string]uint64{
				"/usr": 3 * GiB,
				"/":    1 * GiB,
			},
		},
		{ // big /usr -> / gets default size
			Blueprint: []blueprint.FilesystemCustomization{
				{
					Mountpoint: "/usr",
					MinSize:    10 * GiB,
				},
			},
			ExpectedMinSizes: map[string]uint64{
				"/usr": 10 * GiB,
				"/":    1 * GiB,
			},
		},
		{
			Blueprint: []blueprint.FilesystemCustomization{
				{
					Mountpoint: "/",
					MinSize:    10 * GiB,
				},
				{
					Mountpoint: "/home",
					MinSize:    1 * MiB,
				},
			},
			ExpectedMinSizes: map[string]uint64{
				"/":     10 * GiB,
				"/home": 1 * GiB,
			},
		},
		{ // no separate /usr and no size for / -> / gets sum of default sizes for / and /usr
			Blueprint: []blueprint.FilesystemCustomization{
				{
					Mountpoint: "/opt",
					MinSize:    10 * GiB,
				},
			},
			ExpectedMinSizes: map[string]uint64{
				"/opt": 10 * GiB,
				"/":    4 * GiB,
			},
		},
	}

	for idx, tc := range testCases {
		{ // without LVM
			mpt, err := NewPartitionTable(&pt, tc.Blueprint, uint64(3*GiB), RawPartitioningMode, map[string]uint64{"/": 1 * GiB, "/usr": 3 * GiB}, rng)
			assert.NoError(err)
			for mnt, minSize := range tc.ExpectedMinSizes {
				path := entityPathForMountpoint(mpt, mnt)
				assert.NotNil(path, "[%d] mountpoint %q not found", idx, mnt)
				parent := path[1]
				part, ok := parent.(*Partition)
				assert.True(ok, "%q parent (%v) is not a partition", mnt, parent)
				assert.GreaterOrEqual(part.GetSize(), minSize,
					"[%d] %q size %d should be greater or equal to %d", idx, mnt, part.GetSize(), minSize)
			}
		}

		{ // with LVM
			mpt, err := NewPartitionTable(&pt, tc.Blueprint, uint64(3*GiB), AutoLVMPartitioningMode, map[string]uint64{"/": 1 * GiB, "/usr": 3 * GiB}, rng)
			assert.NoError(err)
			for mnt, minSize := range tc.ExpectedMinSizes {
				path := entityPathForMountpoint(mpt, mnt)
				assert.NotNil(path, "[%d] mountpoint %q not found", idx, mnt)
				parent := path[1]
				part, ok := parent.(*LVMLogicalVolume)
				assert.True(ok, "[%d] %q parent (%v) is not an LVM logical volume", idx, mnt, parent)
				assert.GreaterOrEqual(part.GetSize(), minSize,
					"[%d] %q size %d should be greater or equal to %d", idx, mnt, part.GetSize(), minSize)
			}
		}
	}
}

func TestFSTabOptionsReadOnly(t *testing.T) {
	cases := []struct {
		options    string
		expectedRO bool
	}{
		{"ro", true},
		{"ro,relatime,seclabel,compress=zstd:1,ssd,discard=async,space_cache,subvolid=257,subvol=/root", true},

		{"defaults", false},
		{"rw", false},
		{"rw,ro", false},
		{"rw,relatime,seclabel,compress=zstd:1,ssd,discard=async,space_cache,subvolid=257,subvol=/root", false},
	}

	for _, c := range cases {
		t.Run(c.options, func(t *testing.T) {
			options := FSTabOptions{MntOps: c.options}
			assert.Equal(t, c.expectedRO, options.ReadOnly())
		})
	}
}
