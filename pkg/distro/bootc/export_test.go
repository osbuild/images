package bootc

import (
	"math/rand"

	"github.com/osbuild/blueprint/pkg/blueprint"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/bib/osinfo"
	"github.com/osbuild/images/pkg/bootc"
	"github.com/osbuild/images/pkg/disk"
)

var (
	CheckFilesystemCustomizations = checkFilesystemCustomizations
	UpdateFilesystemSizes         = updateFilesystemSizes
	CalcRequiredDirectorySizes    = calcRequiredDirectorySizes

	TestDiskContainers = diskContainers
)

type ImageType = imageType

func NewTestBootcDistro() *Distro {
	return NewTestBootcDistroWithDefaultFs("xfs")
}

func NewTestBootcDistroWithDefaultFs(defaultFs string) *Distro {
	os := &osinfo.Info{
		OSRelease: osinfo.OSRelease{
			ID:        "bootc-test",
			VersionID: "1",
			Name:      "Bootc Test OS",
		},
		KernelInfo: &osinfo.KernelInfo{
			Version: "6.17.7-300.fc43.x86_64",
		},
	}
	bootcInfo := &bootc.Info{
		Imgref:        "quay.io/example/example:ref",
		ImageID:       "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		OSInfo:        os,
		Arch:          "x86_64",
		DefaultRootFs: defaultFs,
		Size:          0,
	}
	return common.Must(newBootcDistroAfterIntrospect(bootcInfo))
}

func NewTestBootcImageType(imageType string) *ImageType {
	d := NewTestBootcDistro()
	it, err := d.arches["x86_64"].GetImageType(imageType)
	if err != nil {
		panic(err)
	}
	return it.(*ImageType)
}

func (t *ImageType) SetSourceInfoPartitionTable(basept *disk.PartitionTable) {
	t.arch.distro.sourceInfo.PartitionTable = basept
}

func (t *ImageType) GenPartitionTable(customizations *blueprint.Customizations, rootfsMinSize uint64, rng *rand.Rand) (*disk.PartitionTable, error) {
	return t.genPartitionTable(customizations, rootfsMinSize, rng)
}
