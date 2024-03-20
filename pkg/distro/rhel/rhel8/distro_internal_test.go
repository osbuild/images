package rhel8

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/osbuild/images/internal/common"

	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/distro"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testBasicImageType = imageType{
	name:                "test",
	basePartitionTables: defaultBasePartitionTables,
}
var mountpoints = []blueprint.FilesystemCustomization{
	{
		MinSize:    1024,
		Mountpoint: "/usr",
	},
}

type rhelFamilyDistro struct {
	name   string
	distro distro.Distro
}

var rhelFamilyDistros = []rhelFamilyDistro{
	{
		name:   "rhel-810",
		distro: DistroFactory("rhel-810"),
	},
}

// math/rand is good enough in this case
/* #nosec G404 */
var rng = rand.New(rand.NewSource(0))

func TestDistro_UnsupportedArch(t *testing.T) {
	testBasicImageType.arch = &architecture{
		name: "unsupported_arch",
	}
	_, err := testBasicImageType.getPartitionTable(mountpoints, distro.ImageOptions{}, rng)
	require.EqualError(t, err, fmt.Sprintf("no partition table defined for architecture %q for image type %q", testBasicImageType.arch.name, testBasicImageType.name))
}

func TestDistro_DefaultPartitionTables(t *testing.T) {
	rhel8distro := rhelFamilyDistros[0].distro
	for _, archName := range rhel8distro.ListArches() {
		testBasicImageType.arch = &architecture{
			name: archName,
		}
		pt, err := testBasicImageType.getPartitionTable(mountpoints, distro.ImageOptions{}, rng)
		require.Nil(t, err)
		for _, m := range mountpoints {
			assert.True(t, pt.ContainsMountpoint(m.Mountpoint))
		}
	}
}

func TestEC2Partitioning(t *testing.T) {
	testCases := []struct {
		distro             string
		aarch64bootSizeMiB uint64
	}{
		{
			distro:             "rhel-8.8",
			aarch64bootSizeMiB: 512,
		},
		{
			distro:             "rhel-8.9",
			aarch64bootSizeMiB: 512,
		},
		{
			distro:             "rhel-8.10",
			aarch64bootSizeMiB: 1024,
		},
		{
			distro:             "centos-8",
			aarch64bootSizeMiB: 1024,
		},
	}

	for _, tt := range testCases {
		for _, arch := range []string{"x86_64", "aarch64"} {
			for _, it := range []string{"ami", "ec2", "ec2-ha", "ec2-sap"} {
				// skip non-existing combos
				if strings.HasPrefix(it, "ec2") && strings.HasPrefix(tt.distro, "centos") {
					continue
				}
				if arch == "aarch64" && (it == "ec2-ha" || it == "ec2-sap") {
					continue
				}
				t.Run(fmt.Sprintf("%s/%s/%s", tt.distro, arch, it), func(t *testing.T) {
					a, err := DistroFactory(tt.distro).GetArch(arch)
					require.NoError(t, err)
					i, err := a.GetImageType(it)
					require.NoError(t, err)

					it := i.(*imageType)
					pt, err := it.getPartitionTable([]blueprint.FilesystemCustomization{}, distro.ImageOptions{}, rng)
					require.NoError(t, err)

					// x86_64 is /boot-less, check that
					if arch == "x86_64" {
						require.Nil(t, err, pt.FindMountable("/boot"))
						return
					}

					bootSize, err := pt.GetMountpointSize("/boot")
					require.NoError(t, err)
					require.Equal(t, tt.aarch64bootSizeMiB*common.MiB, bootSize)
				})

			}
		}

	}
}

func TestDistroFactory(t *testing.T) {
	type testCase struct {
		strID    string
		expected distro.Distro
	}

	testCases := []testCase{
		{
			strID:    "rhel-8.0",
			expected: newDistro("rhel", 0),
		},
		{
			strID:    "rhel-80",
			expected: newDistro("rhel", 0),
		},
		{
			strID:    "rhel-8.4",
			expected: newDistro("rhel", 4),
		},
		{
			strID:    "rhel-84",
			expected: newDistro("rhel", 4),
		},
		{
			strID:    "rhel-8.10",
			expected: newDistro("rhel", 10),
		},
		{
			strID:    "rhel-810",
			expected: newDistro("rhel", 10),
		},
		{
			strID:    "centos-8",
			expected: newDistro("centos", -1),
		},
		{
			strID:    "centos-8.4",
			expected: nil,
		},
		{
			strID:    "rhel-8",
			expected: nil,
		},
		{
			strID:    "rhel-8.4.1",
			expected: nil,
		},
		{
			strID:    "rhel-7",
			expected: nil,
		},
		{
			strID:    "rhel-79",
			expected: nil,
		},
		{
			strID:    "rhel-7.9",
			expected: nil,
		},
		{
			strID:    "fedora-8",
			expected: nil,
		},
		{
			strID:    "fedora-38",
			expected: nil,
		},
		{
			strID:    "fedora-38.1",
			expected: nil,
		},
		{
			strID:    "fedora",
			expected: nil,
		},
		{
			strID:    "rhel-9",
			expected: nil,
		},
		{
			strID:    "rhel-910",
			expected: nil,
		},
		{
			strID:    "rhel-9.10",
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.strID, func(t *testing.T) {
			d := DistroFactory(tc.strID)
			if tc.expected == nil {
				assert.Nil(t, d)
			} else {
				assert.NotNil(t, d)
				assert.Equal(t, tc.expected.Name(), d.Name())
			}
		})
	}
}
