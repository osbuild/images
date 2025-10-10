package bootc

import (
	"math/rand"

	"github.com/osbuild/blueprint/pkg/blueprint"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/bib/osinfo"
	"github.com/osbuild/images/pkg/disk"
)

var (
	CheckFilesystemCustomizations = checkFilesystemCustomizations
	PartitionTables               = partitionTables
	UpdateFilesystemSizes         = updateFilesystemSizes
	CalcRequiredDirectorySizes    = calcRequiredDirectorySizes

	TestDiskContainers = diskContainers
)

func NewTestBootcDistro() *BootcDistro {
	info := &osinfo.Info{
		OSRelease: osinfo.OSRelease{
			ID: "bootc-test",
		},
	}
	return common.Must(newBootcDistroAfterIntrospect("x86_64", info, "quay.io/example/example:ref", "xfs", 0))
}

func NewTestBootcImageType() *BootcImageType {
	d := NewTestBootcDistro()
	it, err := d.arches["x86_64"].GetImageType("qcow2")
	if err != nil {
		panic(err)
	}
	return it.(*BootcImageType)
}

func (t *BootcImageType) SetSourceInfoPartitionTable(basept *disk.PartitionTable) {
	t.arch.distro.sourceInfo.PartitionTable = basept
}

func (t *BootcImageType) GenPartitionTable(customizations *blueprint.Customizations, rootfsMinSize uint64, rng *rand.Rand) (*disk.PartitionTable, error) {
	return t.genPartitionTable(customizations, rootfsMinSize, rng)
}
