package check

import (
	"bufio"
	"context"
	"strconv"
	"strings"

	"github.com/osbuild/images/cmd/check-host-config/mockos"
)

// OSRelease contains parsed fields from /etc/os-release
type OSRelease struct {
	ID           string
	VersionID    string
	Version      string
	MajorVersion int // Extracted major version from VersionID (e.g., 9 from "9.0")
}

// ParseOSRelease reads and parses /etc/os-release file, extracting
// the id, version_id, and version fields.
func ParseOSRelease(ctx context.Context, log Logger, osReleasePath string) (*OSRelease, error) {
	data, err := mockos.ReadFileContext(ctx, log, osReleasePath)
	if err != nil {
		return nil, err
	}

	release := &OSRelease{}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))

	for scanner.Scan() {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		// Remove quotes if present
		value = strings.Trim(value, "\"")

		switch key {
		case "ID":
			release.ID = value
		case "VERSION_ID":
			release.VersionID = value
		case "VERSION":
			release.Version = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
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
