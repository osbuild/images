package bootc

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// podmanInspectFunc is a function type for executing podman image inspect commands.
// This allows the function to be mocked in tests.
type podmanInspectFunc func(name string, args ...string) ([]byte, error)

// defaultPodmanInspect is the default implementation that executes podman commands.
var defaultPodmanInspect podmanInspectFunc = func(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

// podmanInspect is the function used to execute podman commands. It can be overridden in tests.
var podmanInspect = defaultPodmanInspect

// statFunc is a function type for getting file info.
// This allows the function to be mocked in tests.
type statFunc func(name string) (os.FileInfo, error)

// defaultStat is the default implementation that uses os.Stat.
var defaultStat statFunc = os.Stat

// fileStat is the function used to get file info. It can be overridden in tests.
var fileStat = defaultStat

// isOCIArchive checks if the container reference is an OCI archive and extracts the file path.
// It handles the following formats:
//   - oci-archive:/path/to/file.tar
//   - oci-archive:///path/to/file.tar
//
// Returns ok=true if the reference is an OCI archive along with the file path.
// Returns ok=false if the reference is not an OCI archive, with an empty path.
func isOCIArchive(ref string) (ok bool, path string) {
	if path, hadPrefix := strings.CutPrefix(ref, "oci-archive:"); hadPrefix {
		// Handle oci-archive:// prefix (three slashes)
		path = strings.TrimPrefix(path, "//")
		// If it's a relative path without ./ prefix, add it
		if !strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "./") {
			path = "./" + path
		}
		return true, path
	}
	// Not an OCI archive
	return false, ""
}

// getContainerSize returns the size of an already pulled container image in bytes
func getContainerSize(imgref string) (uint64, error) {
	// Podman inspect does not work for OCI archives, let's do estimation instead
	if ok, path := isOCIArchive(imgref); ok {
		// Get file size directly for OCI archive
		fileInfo, err := fileStat(path)
		if err != nil {
			return 0, fmt.Errorf("failed to stat OCI archive: %w", err)
		}

		// Typical compression ratio for OCI archives is 2x
		// #nosec G115
		return uint64(fileInfo.Size()) * 2, nil
	}

	// For regular images, use podman to inspect
	output, err := podmanInspect("podman", "image", "inspect", imgref, "--format", "{{.Size}}")
	if err != nil {
		return 0, fmt.Errorf("failed inspect image: %w, output\n%s", err, output)
	}

	size, err := strconv.ParseUint(strings.TrimSpace(string(output)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("cannot parse image size: %w", err)
	}

	return size, nil
}
