package rhel10

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/distro/rhel"
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
			distro:      "rhel-10.0",
			bootSizeMiB: 1024,
		},
		{
			distro:      "centos-10",
			bootSizeMiB: 1024,
		},
	}

	for _, tt := range testCases {
		for _, arch := range []string{"x86_64", "aarch64"} {
			for _, it := range []string{"ami"} {
				// skip non-existing combos
				if strings.HasPrefix(it, "ec2") && strings.HasPrefix(tt.distro, "centos") {
					continue
				}
				t.Run(fmt.Sprintf("%s/%s/%s", tt.distro, arch, it), func(t *testing.T) {
					a, err := DistroFactory(tt.distro).GetArch(arch)
					require.NoError(t, err)
					require.NotNil(t, a)
					i, err := a.GetImageType(it)
					require.NoError(t, err)

					it := i.(*rhel.ImageType)
					pt, err := it.GetPartitionTable([]blueprint.FilesystemCustomization{}, distro.ImageOptions{}, rng)
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
			strID:    "rhel-100",
			expected: nil,
		},
		{
			strID:    "rhel-10.0",
			expected: newDistro("rhel", 10, 0),
		},
		{
			strID:    "rhel-103",
			expected: nil,
		},
		{
			strID:    "rhel-10.3",
			expected: newDistro("rhel", 10, 3),
		},
		{
			strID:    "rhel-1010",
			expected: nil,
		},
		{
			strID:    "rhel-10.10",
			expected: newDistro("rhel", 10, 10),
		},
		{
			strID:    "centos-10",
			expected: newDistro("centos", 10, -1),
		},

		{
			strID:    "rhel-90",
			expected: nil,
		},
		{
			strID:    "rhel-9.0",
			expected: nil,
		},
		{
			strID:    "rhel-93",
			expected: nil,
		},
		{
			strID:    "rhel-9.3",
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
		{
			strID:    "centos-9",
			expected: nil,
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
