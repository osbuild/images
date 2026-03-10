package check_test

import (
	"errors"
	"testing"

	"github.com/osbuild/blueprint/pkg/blueprint"
	check "github.com/osbuild/images/cmd/check-host-config/check"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainerEmbeddingCheck(t *testing.T) {
	tests := []struct {
		name       string
		containers []blueprint.Container
		mockExec   map[string]ExecResult
		wantErr    error
	}{
		{
			name:       "skip when no containers",
			containers: nil,
			wantErr:    check.ErrCheckSkipped,
		},
		{
			name: "pass when container is found",
			containers: []blueprint.Container{
				{Source: "registry.example.com/test"},
			},
			mockExec: map[string]ExecResult{
				"sudo podman images --format json": {
					Stdout: []byte(`[{"Names":["registry.example.com/test:latest"]}]`),
				},
			},
		},
		{
			name: "fail when container is not found",
			containers: []blueprint.Container{
				{Source: "registry.example.com/missing"},
			},
			mockExec: map[string]ExecResult{
				"sudo podman images --format json": {
					Stdout: []byte(`[{"Names":["registry.example.com/other:latest"]}]`),
				},
			},
			wantErr: check.ErrCheckFailed,
		},
		{
			name: "fail when podman command fails",
			containers: []blueprint.Container{
				{Source: "registry.example.com/test"},
			},
			mockExec: map[string]ExecResult{
				"sudo podman images --format json": {
					Err: errors.New("podman not found"),
				},
			},
			wantErr: check.ErrCheckFailed,
		},
		{
			name: "pass with multiple containers",
			containers: []blueprint.Container{
				{Source: "registry.example.com/first"},
				{Source: "registry.example.com/second"},
			},
			mockExec: map[string]ExecResult{
				"sudo podman images --format json": {
					Stdout: []byte(`[{"Names":["registry.example.com/first:latest"]},{"Names":["registry.example.com/second:v1"]}]`),
				},
			},
		},
		{
			name: "pass when custom name matches",
			containers: []blueprint.Container{
				{Source: "registry.example.com/source-image", Name: "custom-name:v1"},
			},
			mockExec: map[string]ExecResult{
				"sudo podman images --format json": {
					Stdout: []byte(`[{"Names":["custom-name:v1"]}]`),
				},
			},
		},
		{
			name: "fail when custom name does not match",
			containers: []blueprint.Container{
				{Source: "registry.example.com/source-image", Name: "custom-name:v1"},
			},
			mockExec: map[string]ExecResult{
				"sudo podman images --format json": {
					Stdout: []byte(`[{"Names":["registry.example.com/source-image:latest"]}]`),
				},
			},
			wantErr: check.ErrCheckFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installMockExec(t, tt.mockExec)

			chk, found := check.FindCheckByName("container-embedding")
			require.True(t, found, "container-embedding check not found")

			config := buildConfigWithBlueprint(func(bp *blueprint.Blueprint) {
				bp.Containers = tt.containers
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
