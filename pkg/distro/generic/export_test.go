package generic

import (
	"math/rand"

	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/distro"
)

// math/rand is good enough in this case
/* #nosec G404 */
var rng = rand.New(rand.NewSource(0))

func GetPartitionTable(it distro.ImageType, opts *distro.ImageOptions) (*disk.PartitionTable, error) {
	if opts == nil {
		opts = &distro.ImageOptions{}
	}
	return it.(*imageType).getPartitionTable(&blueprint.Customizations{}, *opts, rng)
}

func BootstrapContainerFor(it distro.ImageType) string {
	return bootstrapContainerFor(it.(*imageType))
}

type (
	ImageType = imageType
)

func (t *imageType) GetDefaultImageConfig() *distro.ImageConfig {
	return t.getDefaultImageConfig()
}

func MockDiskNewPartitionTable(f func(basePT *disk.PartitionTable, mountpoints []blueprint.FilesystemCustomization, imageSize uint64, mode disk.PartitioningMode, architecture arch.Arch, requiredSizes map[string]uint64, rng *rand.Rand) (*disk.PartitionTable, error)) (restore func()) {
	saved := diskNewPartitionTable
	diskNewPartitionTable = f
	return func() {
		diskNewPartitionTable = saved
	}
}
