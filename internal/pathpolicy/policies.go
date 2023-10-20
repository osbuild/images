package pathpolicy

// MountpointPolicies is a set of default mountpoint policies used for filesystem customizations
var MountpointPolicies = NewPathPolicies(map[string]PathPolicy{
	"/":     {Exact: true},
	"/boot": {Exact: true},
	"/var":  {},
	"/opt":  {},
	"/srv":  {},
	// NB: any mountpoints under /usr are not supported by systemd fstab
	// generator in initram before the switch-root, so we don't allow them.
	"/usr":  {Exact: true},
	"/app":  {},
	"/data": {},
	"/home": {},
	"/tmp":  {},
})

// CustomDirectoriesPolicies is a set of default policies for custom directories
var CustomDirectoriesPolicies = NewPathPolicies(map[string]PathPolicy{
	"/":    {Deny: true},
	"/etc": {},
})

// CustomFilesPolicies is a set of default policies for custom files
var CustomFilesPolicies = NewPathPolicies(map[string]PathPolicy{
	"/":           {Deny: true},
	"/etc":        {},
	"/root":       {},
	"/etc/fstab":  {Deny: true},
	"/etc/shadow": {Deny: true},
	"/etc/passwd": {Deny: true},
	"/etc/group":  {Deny: true},
})
