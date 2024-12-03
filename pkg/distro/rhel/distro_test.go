package rhel_test

import (
	"math/rand"
	"testing"

	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/distro_test_common"
	"github.com/osbuild/images/pkg/distro/rhel"
	"github.com/osbuild/images/pkg/distrofactory"
)

// math/rand is good enough in this case
/* #nosec G404 */
var rng = rand.New(rand.NewSource(0))

func TestESP(t *testing.T) {
	var distros []distro.Distro
	distroFactory := distrofactory.NewDefault()
	for _, distroName := range []string{"rhel-7.9", "rhel-8.8", "rhel-8.9", "rhel-8.10", "centos-8", "rhel-9.0", "rhel-9.2", "rhel-9.4", "centos-9", "rhel-10.0", "centos-10"} {
		distros = append(distros, distroFactory.GetDistro(distroName))
	}

	distro_test_common.TestESP(t, distros, func(i distro.ImageType) (*disk.PartitionTable, error) {
		it := i.(*rhel.ImageType)
		return it.GetPartitionTable([]blueprint.FilesystemCustomization{}, distro.ImageOptions{}, rng)
	})
}
