package main

import (
	"github.com/osbuild/images/cmd/check-host-config/check"
)

var checks = []check.Check{
	check.HostnameCheck{},                 // return code 10
	check.FilesCheck{},                    // return code 11
	check.UsersCheck{},                    // return code 12
	check.ServicesEnabledCheck{},          // return code 13
	check.ServicesDisabledCheck{},         // return code 14
	check.ServicesMaskedCheck{},           // return code 15
	check.FirewallServicesEnabledCheck{},  // return code 16
	check.FirewallServicesDisabledCheck{}, // return code 17
	check.FirewallPortsCheck{},            // return code 18
	check.CACertsCheck{},                  // return code 19
	check.ModularityCheck{},               // return code 20
	check.OpenSCAPCheck{},                 // return code 21
}

var MaxShortCheckName int

func init() {
	for _, c := range checks {
		if nameLen := len(c.Metadata().ShortName); nameLen > MaxShortCheckName {
			MaxShortCheckName = nameLen
		}
	}
}
