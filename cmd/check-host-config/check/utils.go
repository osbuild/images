package check

import (
	"fmt"
	"io/fs"
	"log"
	"os/user"
	"strconv"
	"strings"
	"syscall"

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
var ParseOSRelease = func(osReleasePath string) (*OSRelease, error) {
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

// FileInfo returns UNIX file information (mode, uid, gid).
func FileInfo(name string) (fs.FileMode, uint32, uint32, error) {
	info, err := Stat(name)
	if err != nil {
		return 0, 0, 0, err
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, 0, fmt.Errorf("Host checks only work on UNIX-like")
	}

	return info.Mode(), stat.Uid, stat.Gid, nil
}

// LookupUID is a mockable function that looks up a user by name and returns the UID.
// The default implementation uses os/user.Lookup.
var LookupUID = func(username string) (uint32, error) {
	u, err := user.Lookup(username)
	if err != nil {
		return 0, err
	}
	uid, err := strconv.ParseUint(u.Uid, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse UID: %w", err)
	}
	return uint32(uid), nil
}

// LookupGID is a mockable function that looks up a group by name and returns the GID.
// The default implementation uses os/user.LookupGroup.
var LookupGID = func(groupname string) (uint32, error) {
	g, err := user.LookupGroup(groupname)
	if err != nil {
		return 0, err
	}
	gid, err := strconv.ParseUint(g.Gid, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse GID: %w", err)
	}
	return uint32(gid), nil
}

// resolveUser converts an any value (string, int, or int64) to a uint32 UID.
// If the value is a string, it looks up the user name. If it's numeric, it converts directly.
//
//nolint:gosec // G115: caller guarantees UID is in uint32 range
func resolveUser(value any) (uint32, error) {
	switch v := value.(type) {
	case string:
		return LookupUID(v)
	case int:
		return uint32(v), nil
	case int64:
		return uint32(v), nil
	default:
		return 0, fmt.Errorf("unsupported type for user: %T (expected string, int, or int64)", value)
	}
}

// resolveGroup converts an any value (string, int, or int64) to a uint32 GID.
// If the value is a string, it looks up the group name. If it's numeric, it converts directly.
//
//nolint:gosec // G115: caller guarantees GID is in uint32 range
func resolveGroup(value any) (uint32, error) {
	switch v := value.(type) {
	case string:
		return LookupGID(v)
	case int:
		return uint32(v), nil
	case int64:
		return uint32(v), nil
	default:
		return 0, fmt.Errorf("unsupported type for group: %T (expected string, int, or int64)", value)
	}
}
