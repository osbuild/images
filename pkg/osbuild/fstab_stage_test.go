package osbuild

import (
	"testing"

	"github.com/osbuild/images/internal/testdisk"
	"github.com/osbuild/images/pkg/disk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFSTabStage(t *testing.T) {
	expectedStage := &Stage{
		Type:    "org.osbuild.fstab",
		Options: &FSTabStageOptions{},
	}
	actualStage := NewFSTabStage(&FSTabStageOptions{})
	assert.Equal(t, expectedStage, actualStage)
}

func TestAddFilesystem(t *testing.T) {
	options := &FSTabStageOptions{}
	filesystems := []*FSTabEntry{
		{
			UUID:    "76a22bf4-f153-4541-b6c7-0332c0dfaeac",
			VFSType: "ext4",
			Path:    "/",
			Options: "defaults",
			Freq:    1,
			PassNo:  1,
		},
		{
			UUID:    "bba22bf4-f153-4541-b6c7-0332c0dfaeac",
			VFSType: "xfs",
			Path:    "/home",
			Options: "defaults",
			Freq:    1,
			PassNo:  2,
		},
		{
			UUID:    "cca22bf4-f153-4541-b6c7-0332c0dfaeac",
			VFSType: "xfs",
			Path:    "/var",
			Options: "defaults",
			Freq:    1,
			PassNo:  1,
		},
	}

	for i, fs := range filesystems {
		options.AddFilesystem(fs.UUID, fs.VFSType, fs.Path, fs.Options, fs.Freq, fs.PassNo)
		assert.Equal(t, options.FileSystems[i], fs)
	}
	assert.Equal(t, len(filesystems), len(options.FileSystems))
}

func TestNewFSTabStageOptions(t *testing.T) {
	pt := testdisk.MakeFakePartitionTable("/", "/boot", "/boot/efi", "/home", "swap")

	opts, err := NewFSTabStageOptions(pt)
	require.NoError(t, err)

	assert.Equal(t, &FSTabStageOptions{
		FileSystems: []*FSTabEntry{
			{
				UUID:    disk.RootPartitionUUID,
				VFSType: "ext4",
				Path:    "/",
			},
			{
				UUID:    disk.FilesystemDataUUID,
				VFSType: "ext4",
				Path:    "/boot",
			},
			{
				UUID:    disk.EFIFilesystemUUID,
				VFSType: "vfat",
				Path:    "/boot/efi",
			},
			{
				UUID:    disk.FilesystemDataUUID,
				VFSType: "ext4",
				Path:    "/home",
			},
			{
				VFSType: "swap",
				Path:    "none",
			},
		},
	}, opts)
}
