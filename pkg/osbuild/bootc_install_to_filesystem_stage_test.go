package osbuild_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/pkg/osbuild"
)

func TestBootcInstallToFilesystemStageNewHappy(t *testing.T) {
	devices := makeOsbuildDevices("dev-for-/", "dev-for-/boot", "dev-for-/boot/efi")
	mounts := makeOsbuildMounts("/", "/boot", "/boot/efi")

	expectedStage := &osbuild.Stage{
		Type:    "org.osbuild.bootc.install-to-filesystem",
		Devices: devices,
		Mounts:  mounts,
	}
	stage, err := osbuild.NewBootcInstallToFilesystemStage(devices, mounts)
	require.Nil(t, err)
	assert.Equal(t, stage, expectedStage)
}

func TestBootcInstallToFilesystemStageMissingMounts(t *testing.T) {
	devices := makeOsbuildDevices("dev-for-/")
	mounts := makeOsbuildMounts("/")

	stage, err := osbuild.NewBootcInstallToFilesystemStage(devices, mounts)
	// XXX: rename error
	assert.ErrorContains(t, err, "required mounts for bootupd stage [/boot /boot/efi] missing")
	require.Nil(t, stage)
}

func TestBootcInstallToFilesystemStageJsonHappy(t *testing.T) {
	devices := makeOsbuildDevices("disk", "dev-for-/", "dev-for-/boot", "dev-for-/boot/efi")
	mounts := makeOsbuildMounts("/", "/boot", "/boot/efi")

	stage, err := osbuild.NewBootcInstallToFilesystemStage(devices, mounts)
	require.Nil(t, err)
	stageJson, err := json.MarshalIndent(stage, "", "  ")
	require.Nil(t, err)
	assert.Equal(t, string(stageJson), `{
  "type": "org.osbuild.bootc.install-to-filesystem",
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
