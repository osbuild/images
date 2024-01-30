package osbuild

// NewBootcInstallToFilesystem creates a new stage for the
// org.osbuild.bootc.install-to-filesystem stage.
//
// It requires a mount setup so that bootupd can be run by bootc. I.e
// "/", "/boot" and "/boot/efi" need to be set up so that
// bootc/bootupd find and install all required bootloader bits.
//
// The mounts input should be generated with GenBootupdDevicesMounts.
func NewBootcInstallToFilesystemStage(devices map[string]Device, mounts []Mount) (*Stage, error) {
	if err := validateBootupdMounts(mounts); err != nil {
		return nil, err
	}

	return &Stage{
		Type:    "org.osbuild.bootc.install-to-filesystem",
		Devices: devices,
		Mounts:  mounts,
	}, nil
}
