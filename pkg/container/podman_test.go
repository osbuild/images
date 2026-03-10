package container_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/pkg/container"
)

func TestGenDefaultNetworkBackendFile(t *testing.T) {
	tests := []struct {
		name            string
		backend         container.NetworkBackend
		expectedContent string
	}{
		{
			name:            "netavark backend",
			backend:         container.NetworkBackendNetavark,
			expectedContent: "netavark",
		},
		{
			name:            "cni backend",
			backend:         container.NetworkBackendCNI,
			expectedContent: "cni",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := container.GenDefaultNetworkBackendFile(tt.backend)
			require.NoError(t, err)
			require.NotNil(t, file)
			assert.Equal(t, "/var/lib/containers/storage/defaultNetworkBackend", file.Path())
			assert.Equal(t, []byte(tt.expectedContent), file.Data())
		})
	}
}
