package main

import (
	"cmp"
	"slices"

	"github.com/osbuild/images/cmd/check-host-config/check"
)

// getAllChecks returns all checks discovered from the check package.
// Checks are automatically registered via their init() functions.
func getAllChecks() []check.RegisteredCheck {
	checks := check.GetAllChecks()
	// Sort checks by name for consistent ordering
	slices.SortFunc(checks, func(a, b check.RegisteredCheck) int {
		return cmp.Compare(a.Meta.Name, b.Meta.Name)
	})
	return checks
}

var checks = getAllChecks()

// MaxShortCheckName is the length of the longest check short name. This is only used
// for formatting the log output in a nice and readable way.
var MaxShortCheckName int

func init() {
	for _, c := range checks {
		if nameLen := len(c.Meta.ShortName); nameLen > MaxShortCheckName {
			MaxShortCheckName = nameLen
		}
	}
}
