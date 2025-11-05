package bootc

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsOCIArchive(t *testing.T) {
	testCases := []struct {
		name         string
		ref          string
		expected     bool
		expectedPath string
	}{
		{
			name:         "oci-archive with single slash",
			ref:          "oci-archive:/path/to/file.tar",
			expected:     true,
			expectedPath: "/path/to/file.tar",
		},
		{
			name:         "oci-archive with triple slash",
			ref:          "oci-archive:///path/to/file.tar",
			expected:     true,
			expectedPath: "/path/to/file.tar",
		},
		{
			name:         "oci-archive with relative path",
			ref:          "oci-archive:./file.tar",
			expected:     true,
			expectedPath: "./file.tar",
		},
		{
			name:         "oci-archive with relative path without prefix",
			ref:          "oci-archive:file.tar",
			expected:     true,
			expectedPath: "./file.tar",
		},
		{
			name:         "regular container image",
			ref:          "quay.io/example/image:latest",
			expected:     false,
			expectedPath: "",
		},
		{
			name:         "docker image reference",
			ref:          "docker://quay.io/example/image:latest",
			expected:     false,
			expectedPath: "",
		},
		{
			name:         "plain file path without prefix",
			ref:          "/path/to/file.tar",
			expected:     false,
			expectedPath: "",
		},
		{
			name:         "relative file path without prefix",
			ref:          "./file.tar",
			expected:     false,
			expectedPath: "",
		},
		{
			name:         "empty string",
			ref:          "",
			expected:     false,
			expectedPath: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ok, path := isOCIArchive(tc.ref)
			assert.Equal(t, tc.expected, ok, "Expected ok=%v, got ok=%v", tc.expected, ok)
			assert.Equal(t, tc.expectedPath, path, "Expected path=%q, got path=%q", tc.expectedPath, path)
		})
	}
}

func TestGetContainerSize(t *testing.T) {
	// Save original functions to restore after tests
	originalPodmanInspect := podmanInspect
	originalFileStat := fileStat
	defer func() {
		podmanInspect = originalPodmanInspect
		fileStat = originalFileStat
	}()

	t.Run("OCI archive success", func(t *testing.T) {
		// Mock fileStat to return a file with known size
		fileStat = func(name string) (os.FileInfo, error) {
			return mockFileInfo{size: 1000}, nil
		}

		size, err := getContainerSize("oci-archive:/path/to/file.tar")
		assert.NoError(t, err)
		// Should return 2x the file size (compression ratio)
		assert.Equal(t, uint64(2000), size)
	})

	t.Run("OCI archive with triple slash", func(t *testing.T) {
		fileStat = func(name string) (os.FileInfo, error) {
			return mockFileInfo{size: 500}, nil
		}

		size, err := getContainerSize("oci-archive:///path/to/file.tar")
		assert.NoError(t, err)
		assert.Equal(t, uint64(1000), size)
	})

	t.Run("OCI archive stat error", func(t *testing.T) {
		fileStat = func(name string) (os.FileInfo, error) {
			return nil, errors.New("file not found")
		}

		size, err := getContainerSize("oci-archive:/path/to/file.tar")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to stat OCI archive")
		assert.Equal(t, uint64(0), size)
	})

	t.Run("OCI archive relative path", func(t *testing.T) {
		fileStat = func(name string) (os.FileInfo, error) {
			assert.Equal(t, "./file.tar", name)
			return mockFileInfo{size: 750}, nil
		}

		size, err := getContainerSize("oci-archive:file.tar")
		assert.NoError(t, err)
		assert.Equal(t, uint64(1500), size)
	})

	t.Run("regular container image success", func(t *testing.T) {
		// Mock podmanInspect to return a valid size
		podmanInspect = func(name string, args ...string) ([]byte, error) {
			assert.Equal(t, "podman", name)
			return []byte("1073741824\n"), nil // 1GB
		}

		size, err := getContainerSize("quay.io/example/image:latest")
		assert.NoError(t, err)
		assert.Equal(t, uint64(1073741824), size)
	})

	t.Run("regular container image with whitespace", func(t *testing.T) {
		podmanInspect = func(name string, args ...string) ([]byte, error) {
			return []byte("  2048  \n"), nil
		}

		size, err := getContainerSize("quay.io/example/image:latest")
		assert.NoError(t, err)
		assert.Equal(t, uint64(2048), size)
	})

	t.Run("regular container image inspect error", func(t *testing.T) {
		podmanInspect = func(name string, args ...string) ([]byte, error) {
			return []byte("error output"), errors.New("podman command failed")
		}

		size, err := getContainerSize("quay.io/example/image:latest")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed inspect image")
		assert.Equal(t, uint64(0), size)
	})

	t.Run("regular container image parse error", func(t *testing.T) {
		podmanInspect = func(name string, args ...string) ([]byte, error) {
			return []byte("not a number"), nil
		}

		size, err := getContainerSize("quay.io/example/image:latest")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot parse image size")
		assert.Equal(t, uint64(0), size)
	})

	t.Run("regular container image empty output", func(t *testing.T) {
		podmanInspect = func(name string, args ...string) ([]byte, error) {
			return []byte(""), nil
		}

		size, err := getContainerSize("quay.io/example/image:latest")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot parse image size")
		assert.Equal(t, uint64(0), size)
	})
}

// mockFileInfo is a mock implementation of os.FileInfo for testing
type mockFileInfo struct {
	size int64
}

func (m mockFileInfo) Name() string       { return "mock" }
func (m mockFileInfo) Size() int64        { return m.size }
func (m mockFileInfo) Mode() os.FileMode  { return 0 }
func (m mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m mockFileInfo) IsDir() bool        { return false }
func (m mockFileInfo) Sys() interface{}   { return nil }
