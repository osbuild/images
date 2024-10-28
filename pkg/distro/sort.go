package distro

import (
	"errors"
	"regexp"
	"sort"

	"github.com/hashicorp/go-version"
)

var nameVerRe = regexp.MustCompile(`-[0-9]+`)

// SortNames sorts the given list of distro names by name, version
// taking version semantics into account (i.e. sorting 8.1 lower then
// 8.10). Full semantic versioning is supported (see semver.org).
//
// Invalid version numbers will create errors but the sorting continue
// and invalid numbers are sorted lower than anything else (so the
// result is still usable in a {G,T}UI).
func SortNames(distros []string) error {
	var errs []error

	sort.Slice(distros, func(i, j int) bool {
		var name1, ver1, name2, ver2 string

		nameVer1 := distros[i]
		nameVer2 := distros[j]
		sep1 := nameVerRe.FindStringIndex(nameVer1)
		if sep1 == nil {
			name1 = nameVer1
		} else {
			name1 = nameVer1[:sep1[0]]
			ver1 = nameVer1[sep1[0]+1:]
		}
		sep2 := nameVerRe.FindStringIndex(nameVer2)
		if sep2 == nil {
			name2 = nameVer2
		} else {
			name2 = nameVer2[:sep2[0]]
			ver2 = nameVer2[sep2[0]+1:]
		}

		if name1 != name2 {
			return name1 < name2
		}
		// similar to common/distro.go:VersionLessThan but without
		// the panic on invalid numbers
		aV, err := version.NewVersion(ver1)
		if err != nil {
			errs = append(errs, err)
			return true
		}
		bV, err := version.NewVersion(ver2)
		if err != nil {
			errs = append(errs, err)
			return true
		}
		return aV.LessThan(bV)
	})
	return errors.Join(errs...)
}
