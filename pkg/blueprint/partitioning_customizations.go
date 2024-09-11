package blueprint

import (
	"errors"
	"fmt"

	"github.com/osbuild/images/pkg/pathpolicy"
)

// TODO: validate input:
// - Duplicate mountpoints
// - No mixing of btrfs and LVM
// - Only one swap partition or file

type PartitioningCustomization struct {
	MinSize uint64                        `json:"minsize,omitempty" toml:"minsize,omitempty"`
	Plain   *PlainFilesystemCustomization `json:"plain,omitempty" toml:"plain,omitempty"`
	LVM     *LVMCustomization             `json:"lvm,omitempty" toml:"lvm,omitempty"`
	Btrfs   *BtrfsCustomization           `json:"btrfs,omitempty" toml:"btrfs,omitempty"`
}

type PlainFilesystemCustomization struct {
	Filesystems []FilesystemCustomization `json:"filesystems,omitempty" toml:"filesystems,omitempty"`
}

type LVMCustomization struct {
	VolumeGroups []VGCustomization `json:"volume_groups,omitempty" toml:"volume_groups,omitempty"`
}

type VGCustomization struct {
	// Volume group name
	Name string `json:"name" toml:"name"`
	// Size of the partition that contains the volume group
	MinSize        uint64            `json:"minsize" toml:"minsize"`
	LogicalVolumes []LVCustomization `json:"logical_volumes,omitempty" toml:"logical_volumes,omitempty"`
}

type LVCustomization struct {
	// Logical volume name
	Name string `json:"name,omitempty" toml:"name,omitempty"`
	FilesystemCustomization
}

type BtrfsCustomization struct {
	Volumes []BtrfsVolumeCustomization
}

type BtrfsVolumeCustomization struct {
	// Size of the btrfs partition/volume.
	MinSize    uint64 `json:"minsize" toml:"minsize"`
	Subvolumes []BtrfsSubvolumeCustomization
}

type BtrfsSubvolumeCustomization struct {
	Name       string `json:"name" toml:"name"`
	Mountpoint string `json:"mountpoint" toml:"mountpoint"`
}

// CheckMountpointsPolicy checks if the mountpoints are allowed by the policy
func CheckPartitioningPolicy(partitioning *PartitioningCustomization, mountpointAllowList *pathpolicy.PathPolicies) error {
	if partitioning == nil {
		return nil
	}

	// collect all mountpoints
	var mountpoints []string
	if partitioning.Plain != nil {
		for _, part := range partitioning.Plain.Filesystems {
			mountpoints = append(mountpoints, part.Mountpoint)
		}
	}
	if partitioning.LVM != nil {
		for _, vg := range partitioning.LVM.VolumeGroups {
			for _, lv := range vg.LogicalVolumes {
				mountpoints = append(mountpoints, lv.Mountpoint)
			}
		}
	}
	if partitioning.Btrfs != nil {
		for _, vol := range partitioning.Btrfs.Volumes {
			for _, subvol := range vol.Subvolumes {
				mountpoints = append(mountpoints, subvol.Mountpoint)
			}
		}
	}

	var errs []error
	for _, mp := range mountpoints {
		if err := mountpointAllowList.Check(mp); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("The following errors occurred while setting up custom mountpoints:\n%w", errors.Join(errs...))
	}

	return nil
}
