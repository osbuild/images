package kickstart

import "github.com/osbuild/images/pkg/customizations/users"

type File struct {
	Contents string
}

type OSTree struct {
	OSName string
	Remote string
}

type Options struct {
	// Path where the kickstart file will be created
	Path string

	// Add kickstart options to make the installation fully unattended
	Unattended bool

	// Create a sudoers drop-in file for each user or group to enable the
	// NOPASSWD option
	SudoNopasswd []string

	// Kernel options that will be appended to the installed system
	// (not the iso)
	KernelOptionsAppend []string

	// Enable networking on on boot in the installed system
	NetworkOnBoot bool

	Language *string
	Keyboard *string
	Timezone *string

	// Users to create during installation
	Users []users.User

	// Groups to create during installation
	Groups []users.Group

	// ostree-related kickstart options
	OSTree *OSTree

	// User-defined kickstart files that will be added to the ISO
	UserFile *File
}
