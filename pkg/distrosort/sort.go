package distrosort

import (
	"errors"
	"sort"

	"github.com/osbuild/images/pkg/distroidparser"
)

// Names sorts the given list of distro names by name, version
// taking version semantics into account (i.e. sorting 8.1 lower then
// 8.10).
//
// Invalid version numbers will create errors but the sorting continue
// and invalid numbers are sorted lower than anything else (so the
// result is still usable in a {G,T}UI).
//
// Note that full semantic versioning (see semver.org) is not
// supported today but it would be once the underlying distroid parser
// supports better spliting.
func Names(distros []string) error {
	var errs []error

	parser := distroidparser.NewDefaultParser()
	sort.Slice(distros, func(i, j int) bool {
		id1, err := parser.Parse(distros[i])
		if err != nil {
			errs = append(errs, err)
			return true
		}
		id2, err := parser.Parse(distros[j])
		if err != nil {
			errs = append(errs, err)
			return true
		}
		if id1.Name != id2.Name {
			return id1.Name < id2.Name
		}
		ver1, err := id1.Version()
		if err != nil {
			errs = append(errs, err)
			return true
		}
		ver2, err := id2.Version()
		if err != nil {
			errs = append(errs, err)
			return true
		}

		return ver1.LessThan(ver2)
	})
	return errors.Join(errs...)
}
