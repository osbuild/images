package disk_test

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/internal/testdisk"
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/datasizes"
	"github.com/osbuild/images/pkg/disk"
)

const (
	KiB = datasizes.KiB
	MiB = datasizes.MiB
	GiB = datasizes.GiB
)

func TestAlignUp(t *testing.T) {

	pt := disk.PartitionTable{}
	firstAligned := disk.DefaultGrainBytes

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

func TestDynamicallyResizePartitionTable(t *testing.T) {
	mountpoints := []blueprint.FilesystemCustomization{
		{
			MinSize:    2 * GiB,
			Mountpoint: "/usr",
		},
	}
	pt := disk.PartitionTable{
		UUID: "D209C89E-EA5E-4FBD-B161-B461CCE297E0",
		Type: disk.PT_GPT,
		Partitions: []disk.Partition{
			{
				Size:     2048,
				Bootable: true,
				Type:     disk.BIOSBootPartitionGUID,
				UUID:     disk.BIOSBootPartitionUUID,
			},
			{
				Type: disk.FilesystemDataGUID,
				UUID: disk.RootPartitionUUID,
				Payload: &disk.Filesystem{
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
	newpt, err := disk.NewPartitionTable(&pt, mountpoints, 1024, disk.RawPartitioningMode, nil, rng)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, newpt.Size, expectedSize)
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

func TestForEachEntity(t *testing.T) {

	count := 0

	plain := testdisk.TestPartitionTables["plain"]
	err := plain.ForEachEntity(func(e disk.Entity, path []disk.Entity) error {
		assert.NotNil(t, e)
		assert.NotNil(t, path)

		count += 1
		return nil
	})

	assert.NoError(t, err)

	// disk.PartitionTable, 4 partitions, 3 filesystems -> 8 entities
	assert.Equal(t, 8, count)
}

// blueprintApplied checks if the blueprint was applied correctly
// returns nil if the blueprint was applied correctly, an error otherwise
func blueprintApplied(pt *disk.PartitionTable, bp []blueprint.FilesystemCustomization) error {
	for _, mnt := range bp {
		path := disk.EntityPath(pt, mnt.Mountpoint)
		if path == nil {
			return fmt.Errorf("mountpoint %s not found", mnt.Mountpoint)
		}
		for idx, ent := range path {
			if sz, ok := ent.(disk.Sizeable); ok {
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

	sizeCheckCB := func(mnt disk.Mountable, path []disk.Entity) error {
		if strings.HasPrefix(mnt.GetMountpoint(), "/boot") {
			// /boot and subdirectories is exempt from this rule
			return nil
		}
		// go up the path and check every sizeable
		for idx, ent := range path {
			if sz, ok := ent.(disk.Sizeable); ok {
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
	for ptName := range testdisk.TestPartitionTables {
		pt := testdisk.TestPartitionTables[ptName]
		for bpName, bp := range testBlueprints {
			ptMode := disk.RawPartitioningMode
			if ptName == "luks+lvm" {
				ptMode = disk.AutoLVMPartitioningMode
			}
			mpt, err := disk.NewPartitionTable(&pt, bp, uint64(13*MiB), ptMode, nil, rng)
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
		for ptName := range testdisk.TestPartitionTables {
			pt := testdisk.TestPartitionTables[ptName]

			if tbp != nil && (ptName == "btrfs" || ptName == "luks") {
				_, err := disk.NewPartitionTable(&pt, tbp, uint64(13*MiB), disk.AutoLVMPartitioningMode, nil, rng)
				assert.Error(err, "PT %q BP %q: should return an error with LVMPartitioningMode", ptName, bpName)
				continue
			}

			mpt, err := disk.NewPartitionTable(&pt, tbp, uint64(13*MiB), disk.AutoLVMPartitioningMode, nil, rng)
			assert.NoError(err, "PT %q BP %q: Partition table generation failed: (%s)", ptName, bpName, err)

			rootPath := disk.EntityPath(mpt, "/")
			if rootPath == nil {
				panic(fmt.Sprintf("PT %q BP %q: no root mountpoint", ptName, bpName))
			}

			bootPath := disk.EntityPath(mpt, "/boot")
			if tbp != nil && bootPath == nil {
				panic(fmt.Sprintf("PT %q BP %q: no boot mountpoint", ptName, bpName))
			}

			if tbp != nil {
				parent := rootPath[1]
				_, ok := parent.(*disk.LVMLogicalVolume)
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
		for ptName := range testdisk.TestPartitionTables {
			pt := testdisk.TestPartitionTables[ptName]

			if ptName == "auto-lvm" || ptName == "luks" || ptName == "luks+lvm" {
				_, err := disk.NewPartitionTable(&pt, tbp, uint64(13*MiB), disk.BtrfsPartitioningMode, nil, rng)
				assert.Error(err, "PT %q BP %q: should return an error with BtrfsPartitioningMode", ptName, bpName)
				continue
			}

			mpt, err := disk.NewPartitionTable(&pt, tbp, uint64(13*MiB), disk.BtrfsPartitioningMode, nil, rng)
			assert.NoError(err, "PT %q BP %q: Partition table generation failed: (%s)", ptName, bpName, err)

			rootPath := disk.EntityPath(mpt, "/")
			if rootPath == nil {
				panic(fmt.Sprintf("PT %q BP %q: no root mountpoint", ptName, bpName))
			}

			bootPath := disk.EntityPath(mpt, "/boot")
			if tbp != nil && bootPath == nil {
				panic(fmt.Sprintf("PT %q BP %q: no boot mountpoint", ptName, bpName))
			}

			if tbp != nil {
				parent := rootPath[1]
				_, ok := parent.(*disk.Btrfs)
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
		for ptName := range testdisk.TestPartitionTables {
			pt := testdisk.TestPartitionTables[ptName]

			if ptName == "btrfs" || ptName == "luks" {
				_, err := disk.NewPartitionTable(&pt, tbp, uint64(13*MiB), disk.LVMPartitioningMode, nil, rng)
				assert.Error(err, "PT %q BP %q: should return an error with LVMPartitioningMode", ptName, bpName)
				continue
			}

			mpt, err := disk.NewPartitionTable(&pt, tbp, uint64(13*MiB), disk.LVMPartitioningMode, nil, rng)
			require.NoError(t, err, "PT %q BP %q: Partition table generation failed: (%s)", ptName, bpName, err)

			rootPath := disk.EntityPath(mpt, "/")
			if rootPath == nil {
				panic(fmt.Sprintf("PT %q BP %q: no root mountpoint", ptName, bpName))
			}

			bootPath := disk.EntityPath(mpt, "/boot")
			if tbp != nil && bootPath == nil {
				panic(fmt.Sprintf("PT %q BP %q: no boot mountpoint", ptName, bpName))
			}

			// root should always be on a LVM
			rootParent := rootPath[1]
			{
				_, ok := rootParent.(*disk.LVMLogicalVolume)
				assert.True(ok, "PT %q BP %q: root's parent (%+v) is not an LVM logical volume", ptName, bpName, rootParent)
			}

			// check logical volume sizes against blueprint
			var lvsum uint64
			for _, mnt := range tbp {
				if mnt.Mountpoint == "/boot" {
					// not on LVM; skipping
					continue
				}
				mntPath := disk.EntityPath(mpt, mnt.Mountpoint)
				mntParent := mntPath[1]
				mntLV, ok := mntParent.(*disk.LVMLogicalVolume) // the partition's parent should be the logical volume
				assert.True(ok, "PT %q BP %q: %s's parent (%+v) is not an LVM logical volume", ptName, bpName, mnt.Mountpoint, mntParent)
				assert.GreaterOrEqualf(mntLV.Size, mnt.MinSize, "PT %q BP %q: %s's size (%d) is smaller than the requested minsize (%d)", ptName, bpName, mnt.Mountpoint, mntLV.Size, mnt.MinSize)
				lvsum += mntLV.Size
			}

			// root LV's parent should be the VG
			lvParent := rootPath[2]
			{
				_, ok := lvParent.(*disk.LVMVolumeGroup)
				assert.True(ok, "PT %q BP %q: root LV's parent (%+v) is not an LVM volume group", ptName, bpName, lvParent)
			}

			// the root partition is the second to last entity in the path (the last is the partition table itself)
			rootTop := rootPath[len(rootPath)-2]
			vgPart, ok := rootTop.(*disk.Partition)

			// if the VG is in a LUKS container, check that there's enough space for the header too
			{
				vgParent := rootPath[3]
				luksContainer, ok := vgParent.(*disk.LUKSContainer)
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
	pt := testdisk.TestPartitionTables["plain"]

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
			mpt, err := disk.NewPartitionTable(&pt, tc.Blueprint, uint64(3*GiB), disk.RawPartitioningMode, nil, rng)
			assert.NoError(err)
			for mnt, minSize := range tc.ExpectedMinSizes {
				path := disk.EntityPath(mpt, mnt)
				assert.NotNil(path, "[%d] mountpoint %q not found", idx, mnt)
				parent := path[1]
				part, ok := parent.(*disk.Partition)
				assert.True(ok, "%q parent (%v) is not a partition", mnt, parent)
				assert.GreaterOrEqual(part.GetSize(), minSize,
					"[%d] %q size %d should be greater or equal to %d", idx, mnt, part.GetSize(), minSize)
			}
		}

		{ // with LVM
			mpt, err := disk.NewPartitionTable(&pt, tc.Blueprint, uint64(3*GiB), disk.AutoLVMPartitioningMode, nil, rng)
			assert.NoError(err)
			for mnt, minSize := range tc.ExpectedMinSizes {
				path := disk.EntityPath(mpt, mnt)
				assert.NotNil(path, "[%d] mountpoint %q not found", idx, mnt)
				parent := path[1]
				part, ok := parent.(*disk.LVMLogicalVolume)
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
	pt := testdisk.TestPartitionTables["plain"]

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
				"/var": 1*GiB + disk.LVMDefaultExtentSize,
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
		mpt, err := disk.NewPartitionTable(&pt, tc.Blueprint, uint64(3*GiB), disk.AutoLVMPartitioningMode, nil, rng)
		assert.NoError(err)
		for mnt, expSize := range tc.ExpectedSizes {
			path := disk.EntityPath(mpt, mnt)
			assert.NotNil(path, "[%d] mountpoint %q not found", idx, mnt)
			parent := path[1]
			part, ok := parent.(*disk.LVMLogicalVolume)
			assert.True(ok, "[%d] %q parent (%v) is not an LVM logical volume", idx, mnt, parent)
			assert.Equal(part.GetSize(), expSize,
				"[%d] %q size %d should be equal to %d", idx, mnt, part.GetSize(), expSize)
		}
	}
}

func TestNewBootWithSizeLVMify(t *testing.T) {
	pt := testdisk.TestPartitionTables["plain-noboot"]
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

	mpt, err := disk.NewPartitionTable(&pt, custom, uint64(3*GiB), disk.AutoLVMPartitioningMode, nil, rng)
	assert.NoError(err)

	for idx, c := range custom {
		mnt, minSize := c.Mountpoint, c.MinSize
		path := disk.EntityPath(mpt, mnt)
		assert.NotNil(path, "[%d] mountpoint %q not found", idx, mnt)
		parent := path[1]
		part, ok := parent.(*disk.Partition)
		assert.True(ok, "%q parent (%v) is not a partition", mnt, parent)
		assert.GreaterOrEqual(part.GetSize(), minSize,
			"[%d] %q size %d should be greater or equal to %d", idx, mnt, part.GetSize(), minSize)
	}
}

func collectEntities(pt *disk.PartitionTable) []disk.Entity {
	entities := make([]disk.Entity, 0)
	collector := func(ent disk.Entity, path []disk.Entity) error {
		entities = append(entities, ent)
		return nil
	}
	_ = pt.ForEachEntity(collector)
	return entities
}

func TestClone(t *testing.T) {
	for name := range testdisk.TestPartitionTables {
		basePT := testdisk.TestPartitionTables[name]
		baseEntities := collectEntities(&basePT)

		clonePT := basePT.Clone().(*disk.PartitionTable)
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
	usr := disk.Partition{
		Type: disk.FilesystemDataGUID,
		UUID: disk.RootPartitionUUID,
		Payload: &disk.Filesystem{
			Type:         "xfs",
			Label:        "root",
			Mountpoint:   "/usr",
			FSTabOptions: "defaults",
			FSTabFreq:    0,
			FSTabPassNo:  0,
		},
	}

	{
		pt := testdisk.TestPartitionTables["plain"]
		assert.Equal("/", disk.FindDirectoryEntityPath(&pt, "/opt")[0].(disk.Mountable).GetMountpoint())
		assert.Equal("/boot/efi", disk.FindDirectoryEntityPath(&pt, "/boot/efi/Linux")[0].(disk.Mountable).GetMountpoint())
		assert.Equal("/boot", disk.FindDirectoryEntityPath(&pt, "/boot/loader")[0].(disk.Mountable).GetMountpoint())
		assert.Equal("/boot", disk.FindDirectoryEntityPath(&pt, "/boot")[0].(disk.Mountable).GetMountpoint())

		ptMod := pt.Clone().(*disk.PartitionTable)
		ptMod.Partitions = append(ptMod.Partitions, usr)
		assert.Equal("/", disk.FindDirectoryEntityPath(ptMod, "/opt")[0].(disk.Mountable).GetMountpoint())
		assert.Equal("/usr", disk.FindDirectoryEntityPath(ptMod, "/usr")[0].(disk.Mountable).GetMountpoint())
		assert.Equal("/usr", disk.FindDirectoryEntityPath(ptMod, "/usr/bin")[0].(disk.Mountable).GetMountpoint())

		// invalid dir should return nil
		assert.Nil(disk.FindDirectoryEntityPath(&pt, "invalid"))
	}

	{
		pt := testdisk.TestPartitionTables["plain-noboot"]
		assert.Equal("/", disk.FindDirectoryEntityPath(&pt, "/opt")[0].(disk.Mountable).GetMountpoint())
		assert.Equal("/", disk.FindDirectoryEntityPath(&pt, "/boot")[0].(disk.Mountable).GetMountpoint())
		assert.Equal("/", disk.FindDirectoryEntityPath(&pt, "/boot/loader")[0].(disk.Mountable).GetMountpoint())

		ptMod := pt.Clone().(*disk.PartitionTable)
		ptMod.Partitions = append(ptMod.Partitions, usr)
		assert.Equal("/", disk.FindDirectoryEntityPath(ptMod, "/opt")[0].(disk.Mountable).GetMountpoint())
		assert.Equal("/usr", disk.FindDirectoryEntityPath(ptMod, "/usr")[0].(disk.Mountable).GetMountpoint())
		assert.Equal("/usr", disk.FindDirectoryEntityPath(ptMod, "/usr/bin")[0].(disk.Mountable).GetMountpoint())

		// invalid dir should return nil
		assert.Nil(disk.FindDirectoryEntityPath(&pt, "invalid"))
	}

	{
		pt := testdisk.TestPartitionTables["luks"]
		assert.Equal("/", disk.FindDirectoryEntityPath(&pt, "/opt")[0].(disk.Mountable).GetMountpoint())
		assert.Equal("/boot", disk.FindDirectoryEntityPath(&pt, "/boot")[0].(disk.Mountable).GetMountpoint())
		assert.Equal("/boot", disk.FindDirectoryEntityPath(&pt, "/boot/loader")[0].(disk.Mountable).GetMountpoint())

		ptMod := pt.Clone().(*disk.PartitionTable)
		ptMod.Partitions = append(ptMod.Partitions, usr)
		assert.Equal("/", disk.FindDirectoryEntityPath(ptMod, "/opt")[0].(disk.Mountable).GetMountpoint())
		assert.Equal("/usr", disk.FindDirectoryEntityPath(ptMod, "/usr")[0].(disk.Mountable).GetMountpoint())
		assert.Equal("/usr", disk.FindDirectoryEntityPath(ptMod, "/usr/bin")[0].(disk.Mountable).GetMountpoint())

		// invalid dir should return nil
		assert.Nil(disk.FindDirectoryEntityPath(&pt, "invalid"))
	}

	{
		pt := testdisk.TestPartitionTables["luks+lvm"]
		assert.Equal("/", disk.FindDirectoryEntityPath(&pt, "/opt")[0].(disk.Mountable).GetMountpoint())
		assert.Equal("/boot", disk.FindDirectoryEntityPath(&pt, "/boot")[0].(disk.Mountable).GetMountpoint())
		assert.Equal("/boot", disk.FindDirectoryEntityPath(&pt, "/boot/loader")[0].(disk.Mountable).GetMountpoint())

		ptMod := pt.Clone().(*disk.PartitionTable)
		ptMod.Partitions = append(ptMod.Partitions, usr)
		assert.Equal("/", disk.FindDirectoryEntityPath(ptMod, "/opt")[0].(disk.Mountable).GetMountpoint())
		assert.Equal("/usr", disk.FindDirectoryEntityPath(ptMod, "/usr")[0].(disk.Mountable).GetMountpoint())
		assert.Equal("/usr", disk.FindDirectoryEntityPath(ptMod, "/usr/bin")[0].(disk.Mountable).GetMountpoint())

		// invalid dir should return nil
		assert.Nil(disk.FindDirectoryEntityPath(&pt, "invalid"))
	}

	{
		pt := testdisk.TestPartitionTables["btrfs"]
		assert.Equal("/", disk.FindDirectoryEntityPath(&pt, "/opt")[0].(disk.Mountable).GetMountpoint())
		assert.Equal("/boot", disk.FindDirectoryEntityPath(&pt, "/boot")[0].(disk.Mountable).GetMountpoint())
		assert.Equal("/boot", disk.FindDirectoryEntityPath(&pt, "/boot/loader")[0].(disk.Mountable).GetMountpoint())

		ptMod := pt.Clone().(*disk.PartitionTable)
		ptMod.Partitions = append(ptMod.Partitions, usr)
		assert.Equal("/", disk.FindDirectoryEntityPath(ptMod, "/opt")[0].(disk.Mountable).GetMountpoint())
		assert.Equal("/usr", disk.FindDirectoryEntityPath(ptMod, "/usr")[0].(disk.Mountable).GetMountpoint())
		assert.Equal("/usr", disk.FindDirectoryEntityPath(ptMod, "/usr/bin")[0].(disk.Mountable).GetMountpoint())

		// invalid dir should return nil
		assert.Nil(disk.FindDirectoryEntityPath(&pt, "invalid"))
	}

	{
		pt := disk.PartitionTable{} // pt with no root should return nil
		assert.Nil(disk.FindDirectoryEntityPath(&pt, "/var"))
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
		pt := testdisk.TestPartitionTables["plain"]
		pt = *pt.Clone().(*disk.PartitionTable) // don't modify the original test data

		{
			// make sure we have the correct volume
			// guard against changes in the test pt
			rootPart := pt.Partitions[3]
			rootPayload := rootPart.Payload.(*disk.Filesystem)

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
		pt := testdisk.TestPartitionTables["luks+lvm"]
		pt = *pt.Clone().(*disk.PartitionTable) // don't modify the original test data

		{
			// make sure we have the correct volume
			// guard against changes in the test pt
			rootPart := pt.Partitions[3]
			rootLUKS := rootPart.Payload.(*disk.LUKSContainer)
			rootVG := rootLUKS.Payload.(*disk.LVMVolumeGroup)
			rootLV := rootVG.LogicalVolumes[0]
			rootFS := rootLV.Payload.(*disk.Filesystem)
			homeLV := rootVG.LogicalVolumes[1]
			homeFS := homeLV.Payload.(*disk.Filesystem)

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
			rootLUKS := rootPart.Payload.(*disk.LUKSContainer)
			rootVG := rootLUKS.Payload.(*disk.LVMVolumeGroup)
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
		pt := testdisk.TestPartitionTables["btrfs"]
		pt = *pt.Clone().(*disk.PartitionTable) // don't modify the original test data

		{
			// make sure we have the correct volume
			// guard against changes in the test pt
			rootPart := pt.Partitions[3]
			rootPayload := rootPart.Payload.(*disk.Btrfs)
			assert.Equal("/", rootPayload.Subvolumes[0].Mountpoint)
			assert.Equal(uint64(0), rootPayload.Subvolumes[0].Size)
			assert.Equal("/var", rootPayload.Subvolumes[1].Mountpoint)
			assert.Equal(uint64(5*GiB), rootPayload.Subvolumes[1].Size)
		}

		{
			// add requirements for /var subdirs that are > 5 GiB
			pt.EnsureDirectorySizes(varSizes)
			rootPart := pt.Partitions[3]
			rootPayload := rootPart.Payload.(*disk.Btrfs)
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
	pt := testdisk.TestPartitionTables["plain"]

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
			mpt, err := disk.NewPartitionTable(&pt, tc.Blueprint, uint64(3*GiB), disk.RawPartitioningMode, map[string]uint64{"/": 1 * GiB, "/usr": 3 * GiB}, rng)
			assert.NoError(err)
			for mnt, minSize := range tc.ExpectedMinSizes {
				path := disk.EntityPath(mpt, mnt)
				assert.NotNil(path, "[%d] mountpoint %q not found", idx, mnt)
				parent := path[1]
				part, ok := parent.(*disk.Partition)
				assert.True(ok, "%q parent (%v) is not a partition", mnt, parent)
				assert.GreaterOrEqual(part.GetSize(), minSize,
					"[%d] %q size %d should be greater or equal to %d", idx, mnt, part.GetSize(), minSize)
			}
		}

		{ // with LVM
			mpt, err := disk.NewPartitionTable(&pt, tc.Blueprint, uint64(3*GiB), disk.AutoLVMPartitioningMode, map[string]uint64{"/": 1 * GiB, "/usr": 3 * GiB}, rng)
			assert.NoError(err)
			for mnt, minSize := range tc.ExpectedMinSizes {
				path := disk.EntityPath(mpt, mnt)
				assert.NotNil(path, "[%d] mountpoint %q not found", idx, mnt)
				parent := path[1]
				part, ok := parent.(*disk.LVMLogicalVolume)
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
			options := disk.FSTabOptions{MntOps: c.options}
			assert.Equal(t, c.expectedRO, options.ReadOnly())
		})
	}
}

func TestForEachFSTabEntity(t *testing.T) {
	// Use the test partition tables and check that fstab entities are all
	// visited by collecting their target fields.
	// The names must match the ones in testdisk.TestPartitionTables.
	expectedEntityPaths := map[string][]string{
		"plain":        {"/", "/boot", "/boot/efi"},
		"plain-swap":   {"/", "/boot", "none", "/boot/efi"},
		"plain-noboot": {"/", "/boot/efi"},
		"luks":         {"/", "/boot", "/boot/efi"},
		"luks+lvm":     {"/", "/boot", "/home", "/boot/efi"},
		"btrfs":        {"/", "/boot", "/var", "/boot/efi"},
	}

	for name := range testdisk.TestPartitionTables {
		// use a different name for the internal testing argument so we can
		// refer to the global test by t.Name() in the error message
		t.Run(name, func(ts *testing.T) {
			var targets []string
			targetCollectorCB := func(ent disk.FSTabEntity, _ []disk.Entity) error {
				targets = append(targets, ent.GetFSFile())
				return nil
			}

			require := require.New(ts)
			pt := testdisk.TestPartitionTables[name]

			// print an informative failure message if a new test partition
			// table is added and this test is not updated (instead of failing
			// at the final Equal() check)
			exp, ok := expectedEntityPaths[name]
			require.True(ok, "expected test result not defined for test partition table %q: please update the %s test", name, t.Name())

			err := pt.ForEachFSTabEntity(targetCollectorCB)
			// the callback never returns an error, but let's check it anyway
			// in case the foreach function ever changes to return other errors
			require.NoError(err)

			require.NotEmpty(targets)

			// we don't care about the order
			require.ElementsMatch(exp, targets)
		})
	}
}

func TestForEachMountable(t *testing.T) {
	// Use the test partition tables and check that Mountables are all
	// visited by collecting their mountpoints.
	// The names must match the ones in testdisk.TestPartitionTables.
	expectedMountpoints := map[string][]string{
		"plain":        {"/", "/boot", "/boot/efi"},
		"plain-swap":   {"/", "/boot", "/boot/efi"},
		"plain-noboot": {"/", "/boot/efi"},
		"luks":         {"/", "/boot", "/boot/efi"},
		"luks+lvm":     {"/", "/boot", "/home", "/boot/efi"},
		"btrfs":        {"/", "/boot", "/var", "/boot/efi"},
	}

	for name := range testdisk.TestPartitionTables {
		t.Run(name, func(t *testing.T) {
			var mountpoints []string
			mountpointCollectorCB := func(ent disk.Mountable, _ []disk.Entity) error {
				mountpoints = append(mountpoints, ent.GetMountpoint())
				return nil
			}

			require := require.New(t)
			pt := testdisk.TestPartitionTables[name]

			// print an informative failure message if a new test partition
			// table is added and this test is not updated (instead of failing
			// at the final Equal() check)
			exp, ok := expectedMountpoints[name]
			require.True(ok, "expected options not defined for test partition table %q: please update the TestNewFSTabStageOptions test", name)

			err := pt.ForEachMountable(mountpointCollectorCB)
			// the callback never returns an error, but let's check it anyway
			// in case the foreach function ever changes to return other errors
			require.NoError(err)

			require.NotEmpty(mountpoints)

			// we don't care about the order
			require.ElementsMatch(exp, mountpoints)
		})
	}
}
