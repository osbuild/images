package blueprint

// TODO: validate input:
// - Duplicate mountpoints
// - No mixing of btrfs and LVM
// - Only one swap partition or file

type PartitioningCustomization struct {
	Plain *PlainFilesystemCustomization `json:"plain,omitempty" toml:"plain,omitempty"`
	LVM   *LVMCustomization             `json:"lvm,omitempty" toml:"lvm,omitempty"`
	Btrfs *BtrfsCustomization           `json:"btrfs,omitempty" toml:"btrfs,omitempty"`
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
