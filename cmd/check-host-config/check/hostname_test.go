package check_test

import (
	"errors"
	"testing"

	"github.com/osbuild/blueprint/pkg/blueprint"
	check "github.com/osbuild/images/cmd/check-host-config/check"
	"github.com/osbuild/images/internal/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostnameCheck(t *testing.T) {
	tests := []struct {
		name     string
		config   *string // hostname customization
		mockExec map[string]ExecResult
		wantErr  error
	}{
		{
			name:   "pass when hostname matches",
			config: common.ToPtr("test-hostname"),
			mockExec: map[string]ExecResult{
				"hostname ": {Stdout: []byte("test-hostname\n")},
			},
		},
		{
			name:   "warning when hostname does not match",
			config: common.ToPtr("test-hostname"),
			mockExec: map[string]ExecResult{
				"hostname ": {Stdout: []byte("changed-by-cloud-init\n")},
			},
			wantErr: check.ErrCheckWarning,
		},
		{
			name:    "skip when no hostname customization",
			config:  nil,
			wantErr: check.ErrCheckSkipped,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installMockExec(t, tt.mockExec)

			chk, found := check.FindCheckByName("hostname")
			require.True(t, found, "hostname check not found")
			config := buildConfig(&blueprint.Customizations{
				Hostname: tt.config,
			})

			err := chk.Func(chk.Meta, config)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr))
			} else {
				require.NoError(t, err)
			}
		})
	}
}
