// Package versionlock defines structures for creating a dnf versionlock
// configuration file as described in dnf-versionlock(8) and
// dnf5-versionlock(8).
package versionlock

type Config struct {
	Version  string    `toml:"version"`
	Packages []Package `toml:"packages"`
}

type Condition struct {
	// One of 'epoch', 'evr', or 'arch'
	Key string `toml:"key"`

	// One of <, <=, =, >=, >, or !=
	Comparator string `toml:"comparator"`

	Value string `toml:"value"`
}

type Package struct {
	Name       string      `toml:"name"`
	Comment    string      `toml:"comment,omitempty"`
	Conditions []Condition `toml:"conditions"`
}
