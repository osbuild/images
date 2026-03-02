package manifest

import (
	"github.com/osbuild/images/pkg/osbuild"
)

type DiskCustomizations struct {
	// What type of mount configuration should we create, systemd units, fstab
	// or none
	MountConfiguration osbuild.MountConfiguration
}
