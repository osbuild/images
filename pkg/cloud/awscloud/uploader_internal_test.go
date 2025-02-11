package awscloud

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/platform"
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
			expected: common.ToPtr(ec2.BootModeValuesLegacyBios),
		},
		{
			name: "boot-mode-uefi",
			opts: &UploaderOptions{
				BootMode: common.ToPtr(platform.BOOT_UEFI),
			},
			expected: common.ToPtr(ec2.BootModeValuesUefi),
		},
		{
			name: "boot-mode-hybrid",
			opts: &UploaderOptions{
				BootMode: common.ToPtr(platform.BOOT_HYBRID),
			},
			expected: common.ToPtr(ec2.BootModeValuesUefiPreferred),
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
