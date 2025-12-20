package distro

// ISOConfig represents configuration for the ISO part of images that are packed
// into ISOs.
type ISOConfig struct {
}

// InheritFrom inherits unset values from the provided parent configuration and
// returns a new structure instance, which is a result of the inheritance.
func (c *ISOConfig) InheritFrom(parentConfig *ISOConfig) *ISOConfig {
	return shallowMerge(c, parentConfig)
}
