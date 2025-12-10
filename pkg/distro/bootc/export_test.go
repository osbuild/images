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
	UpdateFilesystemSizes         = updateFilesystemSizes
	CalcRequiredDirectorySizes    = calcRequiredDirectorySizes

	TestDiskContainers = diskContainers
)

func NewTestBootcDistro() *BootcDistro {
	return NewTestBootcDistroWithDefaultFs("xfs")
}

func NewTestBootcDistroWithDefaultFs(defaultFs string) *BootcDistro {
	info := &osinfo.Info{
		OSRelease: osinfo.OSRelease{
			ID:        "bootc-test",
			VersionID: "1",
		},
		KernelInfo: &osinfo.KernelInfo{
			Version: "5.14.0-611.4.1.el9_7.x86_64",
		},
	}
	return common.Must(newBootcDistroAfterIntrospect("x86_64", info, "quay.io/example/example:ref", defaultFs, 0))
}

func NewTestBootcImageType(imageType string) *BootcImageType {
	d := NewTestBootcDistro()
	it, err := d.arches["x86_64"].GetImageType(imageType)
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
