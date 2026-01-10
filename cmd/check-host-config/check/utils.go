package check

import (
	"log"
	"strconv"
	"strings"

	"github.com/osbuild/images/pkg/distro"
)

// OSRelease contains parsed fields from /etc/os-release
type OSRelease struct {
	ID           string
	VersionID    string
	Version      string
	MajorVersion int // Extracted major version from VersionID (e.g., 9 from "9.0")
}

// ParseOSRelease is a mockable function that reads and parses /etc/os-release file.
// The default implementation calls distro.ReadOSReleaseFromTree("/") to read from
// the system root, which automatically tries /etc/os-release and /usr/lib/os-release.
// The osReleasePath parameter is kept for API compatibility but ignored in the default implementation.
var ParseOSRelease func(osReleasePath string) (*OSRelease, error) = func(osReleasePath string) (*OSRelease, error) {
	log.Printf("ParseOSRelease: reading from system root\n")
	osrelease, err := distro.ReadOSReleaseFromTree("/")
	if err != nil {
		log.Printf("ParseOSRelease failed: %v\n", err)
		return nil, err
	}

	release := &OSRelease{
		ID:        osrelease["ID"],
		VersionID: osrelease["VERSION_ID"],
		Version:   osrelease["VERSION"],
	}

	// Extract major version from VersionID (e.g., "9.0" -> 9)
	if release.VersionID != "" {
		majorVersionStr := release.VersionID
		if idx := strings.Index(majorVersionStr, "."); idx != -1 {
			majorVersionStr = majorVersionStr[:idx]
		}
		majorVersion, err := strconv.Atoi(majorVersionStr)
		if err != nil {
			// If parsing fails, leave MajorVersion as 0 (zero value)
			// This allows callers to check for 0 to detect invalid versions
			release.MajorVersion = 0
		} else {
			release.MajorVersion = majorVersion
		}
	}

	return release, nil
}
