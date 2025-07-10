package generic_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/generic"
)

func TestGetPartitionTable_FedoraIoTPartitioningModeSpecialCase(t *testing.T) {
	// witness what partition mode was used
	var partMode disk.PartitioningMode
	restore := generic.MockDiskNewPartitionTable(func(basePT *disk.PartitionTable, mountpoints []blueprint.FilesystemCustomization, imageSize uint64, mode disk.PartitioningMode, architecture arch.Arch, requiredSizes map[string]uint64, rng *rand.Rand) (*disk.PartitionTable, error) {
		partMode = mode
		return nil, nil
	})
	defer restore()

	// TODO: ideally we would just construct a test distro
	// based on YAML that contains "rpm_ostree: true" and
	// "distro_like: fedora,rhel8" etc and test more targeted
	// but currently this is (quite) cumbersome so we use this
	// method.
	type testCase struct {
		distroNameVer    string
		imageTypeName    string
		partitioningMode disk.PartitioningMode
		expectError      string
		expectPartMode   disk.PartitioningMode
	}
	testCases := []testCase{
		// fedora IoT error for raw mode (but not btfs :(
		{
			distroNameVer:    "fedora-43",
			imageTypeName:    "iot-raw-xz",
			partitioningMode: disk.RawPartitioningMode,
			expectError:      "partitioning mode raw not supported for iot-raw-xz",
		},
		{
			distroNameVer:    "fedora-43",
			imageTypeName:    "iot-qcow2",
			partitioningMode: disk.RawPartitioningMode,
			expectError:      "partitioning mode raw not supported for iot-qcow2",
		},
		// fedora IoT *mutates* the partitioning mode
		{
			distroNameVer:    "fedora-43",
			imageTypeName:    "iot-raw-xz",
			partitioningMode: disk.LVMPartitioningMode,
			expectError:      "",
			expectPartMode:   disk.AutoLVMPartitioningMode,
		},
		// no modifications/mutations/errors for non-rpmostree
		{
			distroNameVer:    "fedora-43",
			imageTypeName:    "server-qcow2",
			partitioningMode: disk.RawPartitioningMode,
			expectError:      "",
			expectPartMode:   disk.RawPartitioningMode,
		},
		{
			distroNameVer:    "fedora-43",
			imageTypeName:    "server-qcow2",
			partitioningMode: disk.LVMPartitioningMode,
			expectError:      "",
			expectPartMode:   disk.LVMPartitioningMode,
		},
		// no edge/iot for rhel7,10 so we only test normal
		// images only
		{
			distroNameVer:    "rhel-10.1",
			imageTypeName:    "qcow2",
			partitioningMode: disk.RawPartitioningMode,
			expectError:      "",
			expectPartMode:   disk.RawPartitioningMode,
		},
		{
			distroNameVer:    "rhel-7.10",
			imageTypeName:    "ec2",
			partitioningMode: disk.LVMPartitioningMode,
			expectError:      "",
			expectPartMode:   disk.LVMPartitioningMode,
		},

		// XXX: add rhel8,rhel9 once it becomes a generic
		// distro
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s-%s-%v", tc.distroNameVer, tc.imageTypeName, tc.partitioningMode), func(t *testing.T) {
			d := generic.DistroFactory(tc.distroNameVer)
			require.NotNil(t, d)
			a, err := d.GetArch("x86_64")
			require.NoError(t, err)
			imgType, err := a.GetImageType(tc.imageTypeName)
			require.NoError(t, err)

			opts := &distro.ImageOptions{
				PartitioningMode: tc.partitioningMode,
			}
			_, err = generic.GetPartitionTable(imgType, opts)
			if tc.expectError != "" {
				assert.ErrorContains(t, err, tc.expectError)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expectPartMode, partMode)
		})
	}
}
