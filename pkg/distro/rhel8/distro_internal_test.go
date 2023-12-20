package rhel8

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/distro"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testBasicImageType = imageType{
	name:                "test",
	basePartitionTables: defaultBasePartitionTables,
}

var testEc2ImageType = imageType{
	name:                "test_ec2",
	basePartitionTables: ec2BasePartitionTables,
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

func TestDistro_Ec2PartitionTables(t *testing.T) {
	rhel8distro := rhelFamilyDistros[0].distro
	for _, archName := range rhel8distro.ListArches() {
		testEc2ImageType.arch = &architecture{
			name: archName,
		}
		pt, err := testEc2ImageType.getPartitionTable(mountpoints, distro.ImageOptions{}, rng)
		if _, exists := testEc2ImageType.basePartitionTables[archName]; exists {
			require.Nil(t, err)
			for _, m := range mountpoints {
				assert.True(t, pt.ContainsMountpoint(m.Mountpoint))
			}
		} else {
			require.EqualError(t, err, fmt.Sprintf("no partition table defined for architecture %q for image type %q", testEc2ImageType.arch.name, testEc2ImageType.name))
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
			strID:    "fedora-37",
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
