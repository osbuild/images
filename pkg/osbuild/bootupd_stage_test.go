package osbuild_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/osbuild"
)

func makeOsbuildMounts(targets ...string) []osbuild.Mount {
	var mnts []osbuild.Mount
	for _, target := range targets {
		mnts = append(mnts, osbuild.Mount{
			Type:   "org.osbuild.ext4",
			Name:   "mnt-for-" + target,
			Source: "dev-for-" + target,
			Target: target,
		})
	}
	return mnts
}

func makeOsbuildDevices(devnames ...string) map[string]osbuild.Device {
	devices := make(map[string]osbuild.Device)
	for _, devname := range devnames {
		devices[devname] = osbuild.Device{
			Type: "org.osbuild.loopback",
		}
	}
	return devices
}

func TestBootupdStageNewHappy(t *testing.T) {
	opts := &osbuild.BootupdStageOptions{
		StaticConfigs: true,
	}
	devices := makeOsbuildDevices("dev-for-/", "dev-for-/boot", "dev-for-/boot/efi")
	mounts := makeOsbuildMounts("/", "/boot", "/boot/efi")

	expectedStage := &osbuild.Stage{
		Type:    "org.osbuild.bootupd",
		Options: opts,
		Devices: devices,
		Mounts:  mounts,
	}
	stage, err := osbuild.NewBootupdStage(opts, devices, mounts)
	require.Nil(t, err)
	assert.Equal(t, stage, expectedStage)
}

func TestBootupdStageMissingMounts(t *testing.T) {
	opts := &osbuild.BootupdStageOptions{
		StaticConfigs: true,
	}
	devices := makeOsbuildDevices("dev-for-/")
	mounts := makeOsbuildMounts("/")

	stage, err := osbuild.NewBootupdStage(opts, devices, mounts)
	assert.ErrorContains(t, err, "required mounts for bootupd stage [/boot /boot/efi] missing")
	require.Nil(t, stage)
}

func TestBootupdStageMissingDevice(t *testing.T) {
	opts := &osbuild.BootupdStageOptions{
		Bios: &osbuild.BootupdStageOptionsBios{
			Device: "disk",
		},
	}
	devices := makeOsbuildDevices("dev-for-/", "dev-for-/boot", "dev-for-/boot/efi")
	mounts := makeOsbuildMounts("/", "/boot", "/boot/efi")

	stage, err := osbuild.NewBootupdStage(opts, devices, mounts)
	assert.ErrorContains(t, err, `cannot find expected device "disk" for bootupd bios option in [dev-for-/ dev-for-/boot dev-for-/boot/efi]`)
	require.Nil(t, stage)
}

func TestBootupdStageJsonHappy(t *testing.T) {
	opts := &osbuild.BootupdStageOptions{
		Deployment: &osbuild.OSTreeDeployment{
			OSName: "default",
			Ref:    "ostree/1/1/0",
		},
		StaticConfigs: true,
		Bios: &osbuild.BootupdStageOptionsBios{
			Device: "disk",
		},
	}
	devices := makeOsbuildDevices("disk", "dev-for-/", "dev-for-/boot", "dev-for-/boot/efi")
	mounts := makeOsbuildMounts("/", "/boot", "/boot/efi")

	stage, err := osbuild.NewBootupdStage(opts, devices, mounts)
	require.Nil(t, err)
	stageJson, err := json.MarshalIndent(stage, "", "  ")
	require.Nil(t, err)
	assert.Equal(t, string(stageJson), `{
  "type": "org.osbuild.bootupd",
  "options": {
    "deployment": {
      "osname": "default",
      "ref": "ostree/1/1/0"
    },
    "static-configs": true,
    "bios": {
      "device": "disk"
    }
  },
  "devices": {
    "dev-for-/": {
      "type": "org.osbuild.loopback"
    },
    "dev-for-/boot": {
      "type": "org.osbuild.loopback"
    },
    "dev-for-/boot/efi": {
      "type": "org.osbuild.loopback"
    },
    "disk": {
      "type": "org.osbuild.loopback"
    }
  },
  "mounts": [
    {
      "name": "mnt-for-/",
      "type": "org.osbuild.ext4",
      "source": "dev-for-/",
      "target": "/"
    },
    {
      "name": "mnt-for-/boot",
      "type": "org.osbuild.ext4",
      "source": "dev-for-/boot",
      "target": "/boot"
    },
    {
      "name": "mnt-for-/boot/efi",
      "type": "org.osbuild.ext4",
      "source": "dev-for-/boot/efi",
      "target": "/boot/efi"
    }
  ]
}`)
}

func TestGenBootupdDevicesMountsMissingRoot(t *testing.T) {
	filename := "fake-disk.img"
	pt := &disk.PartitionTable{}
	_, _, err := osbuild.GenBootupdDevicesMounts(filename, pt)
	assert.EqualError(t, err, "required mounts for bootupd stage [/ /boot /boot/efi] missing")
}

var fakePt = &disk.PartitionTable{
	UUID: "D209C89E-EA5E-4FBD-B161-B461CCE297E0",
	Type: "gpt",
	Partitions: []disk.Partition{
		{
			Size:     1 * common.MebiByte,
			Bootable: true,
			Type:     disk.BIOSBootPartitionGUID,
			UUID:     disk.BIOSBootPartitionUUID,
		},
		{
			Size: 501 * common.MebiByte,
			Type: disk.EFISystemPartitionGUID,
			UUID: disk.EFISystemPartitionUUID,
			Payload: &disk.Filesystem{
				Type:         "vfat",
				UUID:         disk.EFIFilesystemUUID,
				Mountpoint:   "/boot/efi",
				Label:        "EFI-SYSTEM",
				FSTabOptions: "umask=0077,shortname=winnt",
				FSTabFreq:    0,
				FSTabPassNo:  2,
			},
		},
		{
			Size: 1 * common.GibiByte,
			Type: disk.FilesystemDataGUID,
			UUID: disk.FilesystemDataUUID,
			Payload: &disk.Filesystem{
				Type:         "ext4",
				Mountpoint:   "/boot",
				Label:        "boot",
				FSTabOptions: "defaults",
				FSTabFreq:    1,
				FSTabPassNo:  2,
			},
		},
		{
			Size: 2 * common.GibiByte,
			Type: disk.FilesystemDataGUID,
			UUID: disk.RootPartitionUUID,
			Payload: &disk.Filesystem{
				Type:         "ext4",
				Label:        "root",
				Mountpoint:   "/",
				FSTabOptions: "defaults",
				FSTabFreq:    1,
				FSTabPassNo:  1,
			},
		},
	},
}

func TestGenBootupdDevicesMountsHappy(t *testing.T) {
	filename := "fake-disk.img"

	devices, mounts, err := osbuild.GenBootupdDevicesMounts(filename, fakePt)
	require.Nil(t, err)
	assert.Equal(t, devices, map[string]osbuild.Device{
		"disk": {
			Type: "org.osbuild.loopback",
			Options: &osbuild.LoopbackDeviceOptions{
				Filename: "fake-disk.img",
				Partscan: true,
			},
		},
	})
	assert.Equal(t, mounts, []osbuild.Mount{
		{
			Name:      "part4",
			Type:      "org.osbuild.ext4",
			Source:    "disk",
			Target:    "/",
			Partition: common.ToPtr(4),
		},
		{
			Name:      "part3",
			Type:      "org.osbuild.ext4",
			Source:    "disk",
			Target:    "/boot",
			Partition: common.ToPtr(3),
		},
		{
			Name:      "part2",
			Type:      "org.osbuild.fat",
			Source:    "disk",
			Target:    "/boot/efi",
			Partition: common.ToPtr(2),
		},
	})
}
