package blueprint

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/osbuild/images/pkg/pathpolicy"
)

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

func validateMountpoint(path string) error {
	if path == "" {
		return fmt.Errorf("mountpoint is empty")
	}

	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("mountpoint %q is not an absolute path", path)
	}

	if cleanPath := filepath.Clean(path); path != cleanPath {
		return fmt.Errorf("mountpoint %q is not a canonical path (did you mean %q?)", path, cleanPath)
	}

	return nil
}

func validateFilesystemType(path, fstype string) error {
	// Check that the fs type is valid for the mountpoint.
	// Empty strings are allowed for fstype to set the type automatically based
	// on the distro defaults.
	badfsMsg := "unsupported filesystem type for %q: %s"
	switch path {
	case "/boot":
		switch fstype {
		case "xfs", "ext4", "":
		default:
			return fmt.Errorf(badfsMsg, path, fstype)
		}
	case "/boot/efi":
		switch fstype {
		case "vfat", "":
		default:
			return fmt.Errorf(badfsMsg, path, fstype)
		}
	}
	return nil
}

// ValidateLayoutConstraints checks that at most one LVM Volume Group or btrfs
// volume is defined. Returns an error if both LVM and btrfs are set and if
// either has more than one element.
func (p *PartitioningCustomization) ValidateLayoutConstraints() error {
	if p == nil {
		return nil
	}

	if p.Btrfs != nil && p.LVM != nil {
		return fmt.Errorf("btrfs and lvm partitioning cannot be combined")
	}

	if p.Btrfs != nil && len(p.Btrfs.Volumes) > 1 {
		return fmt.Errorf("multiple btrfs volumes are not yet supported")
	}

	if p.LVM != nil && len(p.LVM.VolumeGroups) > 1 {
		return fmt.Errorf("multiple LVM volume groups are not yet supported")
	}

	return nil
}

// Validate checks for customization combinations that are generally not
// supported or can create conflicts, regardless of specific distro or image
// type policies.
func (p *PartitioningCustomization) Validate() error {
	if p == nil {
		return nil
	}

	// iterate through everything and look for:
	// - invalid mountpoints (global)
	// - duplicate mountpoints (global)
	// - duplicate volume group and logical volume names (lvm)
	// - duplicate subvolume names (btrfs)
	// - empty subvolume names (btrfs)
	// - invalid filesystem for mountpoint (e.g. /boot)
	// - special mountpoints on lvm or btrfs
	// - btrfs as filesystem on plain

	plainOnlyMountpoints := []string{
		"/boot",
		"/boot/efi", // not allowed by our global policies, but that might change
	}

	mountpoints := make(map[string]bool)
	if p.Plain != nil {
		for _, fs := range p.Plain.Filesystems {
			if err := validateMountpoint(fs.Mountpoint); err != nil {
				return fmt.Errorf("invalid plain filesystem customization: %w", err)
			}
			if mountpoints[fs.Mountpoint] {
				return fmt.Errorf("duplicate mountpoint %q in partitioning customizations", fs.Mountpoint)
			}
			if err := validateFilesystemType(fs.Mountpoint, fs.Type); err != nil {
				return fmt.Errorf("invalid plain filesystem customization: %w", err)
			}
			if fs.Type == "btrfs" {
				return fmt.Errorf("btrfs filesystem defined under plain partitioning customization: please use the \"btrfs\" customization to define btrfs volumes and subvolumes")
			}

			mountpoints[fs.Mountpoint] = true
		}
	}

	if p.LVM != nil {
		vgnames := make(map[string]bool)
		for _, vg := range p.LVM.VolumeGroups { // there can be only one VG currently, but keep the check for when we change the rule
			if vg.Name != "" && vgnames[vg.Name] { // VGs with no name get autogenerated names
				return fmt.Errorf("duplicate volume group name %q in partitioning customizations", vg.Name)
			}
			vgnames[vg.Name] = true
			lvnames := make(map[string]bool)
			for _, lv := range vg.LogicalVolumes {
				if lv.Name != "" && lvnames[lv.Name] { // LVs with no name get autogenerated names
					return fmt.Errorf("duplicate lvm logical volume name %q in volume group %q in partitioning customizations", lv.Name, vg.Name)
				}
				lvnames[lv.Name] = true

				if err := validateMountpoint(lv.Mountpoint); err != nil {
					return fmt.Errorf("invalid logical volume customization: %w", err)
				}
				if mountpoints[lv.Mountpoint] {
					return fmt.Errorf("duplicate mountpoint %q in partitioning customizations", lv.Mountpoint)
				}
				mountpoints[lv.Mountpoint] = true

				if slices.Contains(plainOnlyMountpoints, lv.Mountpoint) {
					return fmt.Errorf("invalid mountpoint %q for logical volume", lv.Mountpoint)
				}
			}

		}
	}

	if p.Btrfs != nil {
		for _, vol := range p.Btrfs.Volumes {
			subvolnames := make(map[string]bool)
			for _, subvol := range vol.Subvolumes {
				if subvol.Name == "" {
					return fmt.Errorf("btrfs subvolume with empty name in partitioning customizations")
				}
				if subvolnames[subvol.Name] {
					return fmt.Errorf("duplicate btrfs subvolume name %q in partitioning customizations", subvol.Name)
				}
				subvolnames[subvol.Name] = true

				if err := validateMountpoint(subvol.Mountpoint); err != nil {
					return fmt.Errorf("invalid btrfs subvolume customization: %w", err)
				}
				if mountpoints[subvol.Mountpoint] {
					return fmt.Errorf("duplicate mountpoint %q in partitioning customizations", subvol.Mountpoint)
				}
				if slices.Contains(plainOnlyMountpoints, subvol.Mountpoint) {
					return fmt.Errorf("invalid mountpoint %q for btrfs subvolume", subvol.Mountpoint)
				}
				mountpoints[subvol.Mountpoint] = true
			}
		}
	}

	return nil
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