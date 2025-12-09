package main

import (
	"sort"

	"github.com/osbuild/images/cmd/check-host-config/check"
)

// getAllChecks returns all checks discovered from the check package.
// Checks are automatically registered via their init() functions.
func getAllChecks() []check.RegisteredCheck {
	checks := check.GetAllChecks()
	// Sort checks by name for consistent ordering
	sort.Slice(checks, func(i, j int) bool {
		return checks[i].Meta.Name < checks[j].Meta.Name
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
