package rhel_test

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/osbuild/blueprint/pkg/blueprint"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/distro_test_common"
	"github.com/osbuild/images/pkg/distro/rhel"
	"github.com/osbuild/images/pkg/distrofactory"
	"github.com/osbuild/images/pkg/platform"
	"github.com/stretchr/testify/require"
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
		return it.GetPartitionTable(&blueprint.Customizations{}, distro.ImageOptions{}, rng)
	})
}

// TestAMIHybridBoot verifies that rhel 8.9 and 9.3 are the first RHEL versions
// that implemented hybrid boot for the ami and ec2* image types
func TestAMIHybridBoot(t *testing.T) {
	testCases := []struct {
		distro   string
		bootMode platform.BootMode
	}{
		{"rhel-8.8", platform.BOOT_LEGACY},
		{"rhel-8.9", platform.BOOT_HYBRID},
		{"rhel-8.10", platform.BOOT_HYBRID},
		{"centos-8", platform.BOOT_HYBRID},
		{"rhel-9.0", platform.BOOT_LEGACY},
		{"rhel-9.2", platform.BOOT_LEGACY},
		{"rhel-9.3", platform.BOOT_HYBRID},
		{"rhel-9.4", platform.BOOT_HYBRID},
		{"centos-9", platform.BOOT_HYBRID},
		{"rhel-10.0", platform.BOOT_HYBRID},
		{"centos-10", platform.BOOT_HYBRID},
	}

	distroFactory := distrofactory.NewDefault()

	for _, tc := range testCases {
		// test only x86_64. ami for aarch64 has always UEFI, other arches are not defined.
		a, err := distroFactory.GetDistro(tc.distro).GetArch("x86_64")
		require.NoError(t, err)

		for _, it := range a.ListImageTypes() {
			// test only ami and ec2* image types
			if it != "ami" && !strings.HasPrefix(it, "ec2") {
				continue
			}
			t.Run(fmt.Sprintf("%s/%s", tc.distro, it), func(t *testing.T) {
				it, err := a.GetImageType(it)
				require.NoError(t, err)

				require.Equal(t, tc.bootMode, it.BootMode())
			})
		}
	}

}
