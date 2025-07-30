package generic

import (
	"math/rand"

	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/distro"
)

// math/rand is good enough in this case
/* #nosec G404 */
var rng = rand.New(rand.NewSource(0))

func GetPartitionTable(it distro.ImageType) (*disk.PartitionTable, error) {
	return it.(*imageType).getPartitionTable(&blueprint.Customizations{}, distro.ImageOptions{}, rng)
}

func BootstrapContainerFor(it distro.ImageType) string {
	return bootstrapContainerFor(it.(*imageType))
}

type ImageType = imageType

func (t *imageType) GetDefaultImageConfig() *distro.ImageConfig {
	return t.getDefaultImageConfig()
}

var (
	CheckOptionsFedora = checkOptionsFedora
	CheckOptionsRhel7  = checkOptionsRhel7
	CheckOptionsRhel8  = checkOptionsRhel8
	CheckOptionsRhel9  = checkOptionsRhel9
	CheckOptionsRhel10 = checkOptionsRhel10
)
