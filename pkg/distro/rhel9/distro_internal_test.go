package rhel9

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/pkg/distro"
)

// math/rand is good enough in this case
/* #nosec G404 */
var rng = rand.New(rand.NewSource(0))

func TestEC2Partitioning(t *testing.T) {
	testCases := []struct {
		distro      string
		bootSizeMiB uint64
	}{
		// x86_64
		{
			distro:      "rhel-9.2",
			bootSizeMiB: 500,
		},
		{
			distro:      "rhel-9.3",
			bootSizeMiB: 600,
		},
		{
			distro:      "rhel-9.4",
			bootSizeMiB: 1024,
		},
		{
			distro:      "centos-9",
			bootSizeMiB: 1024,
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

					bootSize, err := pt.GetMountpointSize("/boot")
					require.NoError(t, err)
					require.Equal(t, tt.bootSizeMiB*common.MiB, bootSize)
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
			strID:    "rhel-90",
			expected: newDistro("rhel", 9, 0),
		},
		{
			strID:    "rhel-9.0",
			expected: newDistro("rhel", 9, 0),
		},
		{
			strID:    "rhel-93",
			expected: newDistro("rhel", 9, 3),
		},
		{
			strID:    "rhel-9.3",
			expected: newDistro("rhel", 9, 3),
		},
		{
			strID:    "rhel-910",
			expected: newDistro("rhel", 9, 10),
		},
		{
			strID:    "rhel-9.10",
			expected: newDistro("rhel", 9, 10),
		},
		{
			strID:    "centos-9",
			expected: newDistro("centos", 9, -1),
		},
		{
			strID:    "centos-9.0",
			expected: nil,
		},
		{
			strID:    "rhel-9",
			expected: nil,
		},
		{
			strID:    "rhel-8.0",
			expected: nil,
		},
		{
			strID:    "rhel-80",
			expected: nil,
		},
		{
			strID:    "rhel-8.4",
			expected: nil,
		},
		{
			strID:    "rhel-84",
			expected: nil,
		},
		{
			strID:    "rhel-8.10",
			expected: nil,
		},
		{
			strID:    "rhel-810",
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
			strID:    "fedora-9",
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
