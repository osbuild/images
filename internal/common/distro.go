package common

import (
	"strings"

	"github.com/hashicorp/go-version"
)

// Returns true if the version represented by the first argument is
// semantically older than the second.
//
// Meant to be used for comparing distro versions for differences between minor
// releases.
//
// Provided version strings are of any characters which are not
// digits or periods, and then split on periods.
// Assumes any missing components are 0, so 8 < 8.1.
// Evaluates to false if a and b are equal.
func VersionLessThan(a, b string) bool {
	aV, err := version.NewVersion(a)
	if err != nil {
		panic(err)
	}
	bV, err := version.NewVersion(b)
	if err != nil {
		panic(err)
	}

	return aV.LessThan(bV)
}

func VersionGreaterThanOrEqual(a, b string) bool {
	return !VersionLessThan(a, b)
}

// SplitDistroNameVer splits the given distro nameVer string
// into the distro and version part. This assuem the pattern
// "$distro-$version" and that version does not contain a "-".
//
// E.g. "centos-stream-9" will return ("centos-stream", "9")
func SplitDistroNameVer(distroNameVer string) (string, string) {
	// we need to split from the right for "centos-stream-10" like
	// distro names, sadly go has no rsplit() so we do it manually
	idx := strings.LastIndex(distroNameVer, "-")
	if idx < 0 {
		return distroNameVer, ""
	}
	return distroNameVer[:idx], distroNameVer[idx+1:]
}
