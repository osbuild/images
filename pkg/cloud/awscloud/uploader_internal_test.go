package awscloud

import (
	"testing"

	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/platform"
	"github.com/stretchr/testify/assert"
)

func TestUploaderOptionsEc2BootMode(t *testing.T) {
	testCases := []struct {
		name     string
		opts     *UploaderOptions
		expected *string
		err      bool
	}{
		{
			name:     "nil",
			opts:     nil,
			expected: nil,
		},
		{
			name:     "boot-mode-not-set",
			opts:     &UploaderOptions{},
			expected: nil,
		},
		{
			name: "boot-mode-legacy",
			opts: &UploaderOptions{
				BootMode: common.ToPtr(platform.BOOT_LEGACY),
			},
			expected: common.ToPtr(string(ec2types.BootModeValuesLegacyBios)),
		},
		{
			name: "boot-mode-uefi",
			opts: &UploaderOptions{
				BootMode: common.ToPtr(platform.BOOT_UEFI),
			},
			expected: common.ToPtr(string(ec2types.BootModeValuesUefi)),
		},
		{
			name: "boot-mode-hybrid",
			opts: &UploaderOptions{
				BootMode: common.ToPtr(platform.BOOT_HYBRID),
			},
			expected: common.ToPtr(string(ec2types.BootModeValuesUefiPreferred)),
		},
		{
			name: "boot-mode-invalid",
			opts: &UploaderOptions{
				BootMode: common.ToPtr(platform.BootMode(1234)),
			},
			err: true,
		},
		{
			name: "boot-mode-none",
			opts: &UploaderOptions{
				BootMode: common.ToPtr(platform.BOOT_NONE),
			},
			err: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := tc.opts.ec2BootMode()
			if tc.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, actual)
			}
		})
	}
}
