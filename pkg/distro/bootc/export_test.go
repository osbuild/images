package bootc

import (
	"math/rand"

	"github.com/osbuild/blueprint/pkg/blueprint"

	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/bib/osinfo"
	"github.com/osbuild/images/pkg/disk"
)

var (
	CheckFilesystemCustomizations = checkFilesystemCustomizations
	PartitionTables               = partitionTables
	UpdateFilesystemSizes         = updateFilesystemSizes
	CreateRand                    = createRand
	CalcRequiredDirectorySizes    = calcRequiredDirectorySizes
)

func NewTestBootcImageType() *BootcImageType {
	d := &BootcDistro{
		sourceInfo: &osinfo.Info{
			OSRelease: osinfo.OSRelease{
				ID: "bootc-test",
			},
		},
		defaultFs: "xfs",
	}
	a := &BootcArch{distro: d, arch: arch.ARCH_X86_64}
	imgType := &BootcImageType{
		arch:   a,
		name:   "qcow2",
		export: "qcow2",
	}
	a.addImageTypes(*imgType)

	return imgType
}

func (t *BootcImageType) GenPartitionTable(customizations *blueprint.Customizations, rootfsMinSize uint64, rng *rand.Rand) (*disk.PartitionTable, error) {
	return t.genPartitionTable(customizations, rootfsMinSize, rng)
}
