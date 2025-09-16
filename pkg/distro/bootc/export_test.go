package bootc

import (
	"math/rand"

	"github.com/osbuild/blueprint/pkg/blueprint"

	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/bib/osinfo"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/distro"
)

var (
	CheckFilesystemCustomizations = checkFilesystemCustomizations
	PartitionTables               = partitionTables
	UpdateFilesystemSizes         = updateFilesystemSizes
	CalcRequiredDirectorySizes    = calcRequiredDirectorySizes

	TestDiskContainers = diskContainers
)

func NewTestBootcImageType() *BootcImageType {
	d := &BootcDistro{
		sourceInfo: &osinfo.Info{
			OSRelease: osinfo.OSRelease{
				ID: "bootc-test",
			},
		},
		imgref:    "quay.io/example/example:ref",
		defaultFs: "xfs",
	}
	a := &BootcArch{distro: d, arch: arch.ARCH_X86_64}
	d.arches = map[string]distro.Arch{
		"x86_64": a,
	}

	imgType := &BootcImageType{
		arch:   a,
		name:   "qcow2",
		export: "qcow2",
		ext:    "qcow2",
	}
	a.addImageTypes(*imgType)

	return imgType
}

func (t *BootcImageType) SetSourceInfoPartitionTable(basept *disk.PartitionTable) {
	t.arch.distro.sourceInfo.PartitionTable = basept
}

func (t *BootcImageType) GenPartitionTable(customizations *blueprint.Customizations, rootfsMinSize uint64, rng *rand.Rand) (*disk.PartitionTable, error) {
	return t.genPartitionTable(customizations, rootfsMinSize, rng)
}
