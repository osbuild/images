package distro

type ImageTypeConfig struct {
	Bootable  bool
	RpmOstree bool
	BootISO   bool

	DefaultImageConfig *ImageConfig
	KernelOptions      []string

	// XXX: replace with something better
	IsRHEL bool
}
