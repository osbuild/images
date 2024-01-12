package osbuild_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/pkg/osbuild"
)

func makeOsbuildMounts(targets ...string) osbuild.Mounts {
	var mnts []osbuild.Mount
	for _, target := range targets {
		mnts = append(mnts, osbuild.Mount{
			Type:   "org.osbuild.ext4",
			Name:   "mnt-" + target,
			Source: "dev-" + target,
			Target: target,
		})
	}
	return mnts
}

func makeOsbuildDevices(devnames ...string) osbuild.Devices {
	devices := make(map[string]osbuild.Device)
	for _, devname := range devnames {
		devices[devname] = osbuild.Device{
			Type: "orgosbuild.loopback",
		}
	}
	return devices
}

func TestBootupdStageNewHappy(t *testing.T) {
	opts := &osbuild.BootupdStageOptions{
		StaticConfigs: true,
	}
	devices := makeOsbuildDevices("dev-/", "dev-/boot", "dev-/boot/efi")
	mounts := makeOsbuildMounts("/", "/boot", "/boot/efi")

	expectedStage := &osbuild.Stage{
		Type:    "org.osbuild.bootupd",
		Options: opts,
		Devices: devices,
		Mounts:  mounts,
	}
	stage, err := osbuild.NewBootupdStage(opts, &devices, &mounts)
	require.Nil(t, err)
	assert.Equal(t, stage, expectedStage)
}

func TestBootupdStageMissingMounts(t *testing.T) {
	opts := &osbuild.BootupdStageOptions{
		StaticConfigs: true,
	}
	devices := makeOsbuildDevices("dev-/")
	mounts := makeOsbuildMounts("/")

	stage, err := osbuild.NewBootupdStage(opts, &devices, &mounts)
	assert.ErrorContains(t, err, "required mounts for bootupd stage [/boot /boot/efi] missing")
	require.Nil(t, stage)
}

func TestBootupdStageMissingDevice(t *testing.T) {
	opts := &osbuild.BootupdStageOptions{
		Bios: &osbuild.BootupdStageOptionsBios{
			Device: "disk",
		},
	}
	devices := makeOsbuildDevices("dev-/", "dev-/boot", "dev-/boot/efi")
	mounts := makeOsbuildMounts("/", "/boot", "/boot/efi")

	stage, err := osbuild.NewBootupdStage(opts, &devices, &mounts)
	assert.ErrorContains(t, err, `cannot find expected device "disk" for bootupd bios option in [dev-/ dev-/boot dev-/boot/efi]`)
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
	devices := makeOsbuildDevices("disk", "dev-/", "dev-/boot", "dev-/boot/efi")
	mounts := makeOsbuildMounts("/", "/boot", "/boot/efi")

	stage, err := osbuild.NewBootupdStage(opts, &devices, &mounts)
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
    "dev-/": {
      "type": "orgosbuild.loopback"
    },
    "dev-/boot": {
      "type": "orgosbuild.loopback"
    },
    "dev-/boot/efi": {
      "type": "orgosbuild.loopback"
    },
    "disk": {
      "type": "orgosbuild.loopback"
    }
  },
  "mounts": [
    {
      "name": "mnt-/",
      "type": "org.osbuild.ext4",
      "source": "dev-/",
      "target": "/"
    },
    {
      "name": "mnt-/boot",
      "type": "org.osbuild.ext4",
      "source": "dev-/boot",
      "target": "/boot"
    },
    {
      "name": "mnt-/boot/efi",
      "type": "org.osbuild.ext4",
      "source": "dev-/boot/efi",
      "target": "/boot/efi"
    }
  ]
}`)
}
