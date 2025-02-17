package common

import (
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
//
// Also assumes that rawhide is *always* greater than anything else to deal
// with Fedora specific versioning
func VersionLessThan(a, b string) bool {
	if a == "rawhide" {
		return false
	}

	if b == "rawhide" {
		return true
	}

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
