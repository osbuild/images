package fedora

import (
	"math/rand"
	"testing"

	"github.com/osbuild/blueprint/pkg/blueprint"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/distro_test_common"
)

// math/rand is good enough in this case
/* #nosec G404 */
var rng = rand.New(rand.NewSource(0))

func TestESP(t *testing.T) {
	var distros []distro.Distro
	for _, distroName := range []string{"fedora-40", "fedora-41", "fedora-42"} {
		d := DistroFactory(distroName)
		distros = append(distros, d)
	}

	distro_test_common.TestESP(t, distros, func(i distro.ImageType) (*disk.PartitionTable, error) {
		it := i.(*imageType)
		return it.getPartitionTable(&blueprint.Customizations{}, distro.ImageOptions{}, rng)
	})
}
