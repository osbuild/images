package osbuild

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/internal/testdisk"
	"github.com/osbuild/images/pkg/datasizes"
	"github.com/osbuild/images/pkg/disk"
)

func TestNewMkfsStage(t *testing.T) {
	devOpts := LoopbackDeviceOptions{
		Filename:   "file.img",
		Start:      0,
		Size:       1024,
		SectorSize: common.ToPtr(uint64(512)),
	}
	device := NewLoopbackDevice(&devOpts)

	devices := map[string]Device{
		"device": *device,
	}

	btrfsOptions := &MkfsBtrfsStageOptions{
		UUID:  uuid.New().String(),
		Label: "test",
	}
	mkbtrfs := NewMkfsBtrfsStage(btrfsOptions, devices)
	mkbtrfsExpected := &Stage{
		Type:    "org.osbuild.mkfs.btrfs",
		Options: btrfsOptions,
		Devices: map[string]Device{"device": *device},
	}
	assert.Equal(t, mkbtrfsExpected, mkbtrfs)

	ext4Options := &MkfsExt4StageOptions{
		UUID:  uuid.New().String(),
		Label: "test",
	}
	mkext4 := NewMkfsExt4Stage(ext4Options, devices)
	mkext4Expected := &Stage{
		Type:    "org.osbuild.mkfs.ext4",
		Options: ext4Options,
		Devices: map[string]Device{"device": *device},
	}
	assert.Equal(t, mkext4Expected, mkext4)

	fatOptions := &MkfsFATStageOptions{
		VolID:   "7B7795E7",
		Label:   "test",
		FATSize: common.ToPtr(12),
	}
	mkfat := NewMkfsFATStage(fatOptions, devices)
	mkfatExpected := &Stage{
		Type:    "org.osbuild.mkfs.fat",
		Options: fatOptions,
		Devices: map[string]Device{"device": *device},
	}
	assert.Equal(t, mkfatExpected, mkfat)

	xfsOptions := &MkfsXfsStageOptions{
		UUID:  uuid.New().String(),
		Label: "test",
	}
	mkxfs := NewMkfsXfsStage(xfsOptions, devices)
	mkxfsExpected := &Stage{
		Type:    "org.osbuild.mkfs.xfs",
		Options: xfsOptions,
		Devices: map[string]Device{"device": *device},
	}
	assert.Equal(t, mkxfsExpected, mkxfs)
}

func TestGenMkfsStages(t *testing.T) {
	pt := testdisk.MakeFakePartitionTable("/", "/boot", "/boot/efi")
	stages := GenMkfsStages(pt, "file.img")
	assert.Equal(t, []*Stage{
		{
			Type: "org.osbuild.mkfs.ext4",
			Options: &MkfsExt4StageOptions{
				UUID: disk.RootPartitionUUID,
			},
			Devices: map[string]Device{
				"device": {
					Type: "org.osbuild.loopback",
					Options: &LoopbackDeviceOptions{
						Filename: "file.img",
						Size:     testdisk.FakePartitionSize / disk.DefaultSectorSize,
						Lock:     true,
					},
				},
			},
		},
		{
			Type: "org.osbuild.mkfs.ext4",
			Options: &MkfsExt4StageOptions{
				UUID: disk.FilesystemDataUUID,
			},
			Devices: map[string]Device{
				"device": {
					Type: "org.osbuild.loopback",
					Options: &LoopbackDeviceOptions{
						Filename: "file.img",
						Size:     testdisk.FakePartitionSize / disk.DefaultSectorSize,
						Lock:     true,
					},
				},
			},
		},
		{
			Type: "org.osbuild.mkfs.fat",
			Options: &MkfsFATStageOptions{
				VolID: strings.ReplaceAll(disk.EFIFilesystemUUID, "-", ""),
			},
			Devices: map[string]Device{
				"device": {
					Type: "org.osbuild.loopback",
					Options: &LoopbackDeviceOptions{
						Filename: "file.img",
						Size:     testdisk.FakePartitionSize / disk.DefaultSectorSize,
						Lock:     true,
					},
				},
			},
		},
	}, stages)
}

func TestGenMkfsStagesBtrfs(t *testing.T) {
	// Let's put there /extra to make sure that / and /extra creates only one btrfs partition
	pt := testdisk.MakeFakeBtrfsPartitionTable("/", "/boot", "/boot/efi", "/extra")
	stages := GenMkfsStages(pt, "file.img")
	assert.Equal(t, []*Stage{
		{
			Type:    "org.osbuild.mkfs.ext4",
			Options: &MkfsExt4StageOptions{},
			Devices: map[string]Device{
				"device": {
					Type: "org.osbuild.loopback",
					Options: &LoopbackDeviceOptions{
						Filename: "file.img",
						Size:     datasizes.GiB / disk.DefaultSectorSize,
						Lock:     true,
					},
				},
			},
		},
		{
			Type: "org.osbuild.mkfs.fat",
			Options: &MkfsFATStageOptions{
				VolID: strings.ReplaceAll(disk.EFIFilesystemUUID, "-", ""),
			},
			Devices: map[string]Device{
				"device": {
					Type: "org.osbuild.loopback",
					Options: &LoopbackDeviceOptions{
						Filename: "file.img",
						Start:    datasizes.GiB / disk.DefaultSectorSize,
						Size:     100 * datasizes.MiB / disk.DefaultSectorSize,
						Lock:     true,
					},
				},
			},
		},
		{
			Type: "org.osbuild.mkfs.btrfs",
			Options: &MkfsBtrfsStageOptions{
				UUID: disk.RootPartitionUUID,
			},
			Devices: map[string]Device{
				"device": {
					Type: "org.osbuild.loopback",
					Options: &LoopbackDeviceOptions{
						Filename: "file.img",
						Start:    (datasizes.GiB + 100*datasizes.MiB) / disk.DefaultSectorSize,
						Size:     9 * datasizes.GiB / disk.DefaultSectorSize,
						Lock:     true,
					},
				},
			},
		},
	}, stages)
}

func TestGenMkfsStagesUnhappy(t *testing.T) {
	pt := &disk.PartitionTable{
		Type: disk.PT_GPT,
		Partitions: []disk.Partition{
			{
				Payload: &disk.Filesystem{
					Type: "ext2",
				},
			},
		},
	}

	assert.PanicsWithValue(t, "unknown fs type ext2", func() {
		GenMkfsStages(pt, "file.img")
	})
}
