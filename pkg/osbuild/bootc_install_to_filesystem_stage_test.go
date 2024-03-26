package osbuild_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/osbuild"
)

func makeFakeContainerInputs() osbuild.ContainerDeployInputs {
	return osbuild.ContainerDeployInputs{
		Images: osbuild.NewContainersInputForSources([]container.Spec{
			{
				ImageID:   "id-0",
				Source:    "registry.example.org/reg/img",
				LocalName: "local-name",
			},
		},
		),
	}
}

func TestBootcInstallToFilesystemStageNewHappy(t *testing.T) {
	devices := makeOsbuildDevices("dev-for-/", "dev-for-/boot", "dev-for-/boot/efi")
	mounts := makeOsbuildMounts("/", "/boot", "/boot/efi")
	inputs := makeFakeContainerInputs()

	expectedStage := &osbuild.Stage{
		Type:    "org.osbuild.bootc.install-to-filesystem",
		Inputs:  inputs,
		Devices: devices,
		Mounts:  mounts,
	}
	stage, err := osbuild.NewBootcInstallToFilesystemStage(inputs, devices, mounts)
	require.Nil(t, err)
	assert.Equal(t, stage, expectedStage)
}

func TestBootcInstallToFilesystemStageNewNoContainers(t *testing.T) {
	devices := makeOsbuildDevices("dev-for-/", "dev-for-/boot", "dev-for-/boot/efi")
	mounts := makeOsbuildMounts("/", "/boot", "/boot/efi")
	inputs := osbuild.ContainerDeployInputs{}

	_, err := osbuild.NewBootcInstallToFilesystemStage(inputs, devices, mounts)
	assert.EqualError(t, err, "expected exactly one container input but got: 0 (map[])")
}

func TestBootcInstallToFilesystemStageNewTwoContainers(t *testing.T) {
	devices := makeOsbuildDevices("dev-for-/", "dev-for-/boot", "dev-for-/boot/efi")
	mounts := makeOsbuildMounts("/", "/boot", "/boot/efi")
	inputs := osbuild.ContainerDeployInputs{
		Images: osbuild.ContainersInput{
			References: map[string]osbuild.ContainersInputSourceRef{
				"1": {},
				"2": {},
			},
		},
	}

	_, err := osbuild.NewBootcInstallToFilesystemStage(inputs, devices, mounts)
	assert.EqualError(t, err, "expected exactly one container input but got: 2 (map[1:{} 2:{}])")
}

func TestBootcInstallToFilesystemStageMissingMounts(t *testing.T) {
	devices := makeOsbuildDevices("dev-for-/")
	mounts := makeOsbuildMounts("/")
	inputs := makeFakeContainerInputs()

	stage, err := osbuild.NewBootcInstallToFilesystemStage(inputs, devices, mounts)
	// XXX: rename error
	assert.ErrorContains(t, err, "required mounts for bootupd stage [/boot /boot/efi] missing")
	require.Nil(t, stage)
}

func TestBootcInstallToFilesystemStageJsonHappy(t *testing.T) {
	devices := makeOsbuildDevices("disk", "dev-for-/", "dev-for-/boot", "dev-for-/boot/efi")
	mounts := makeOsbuildMounts("/", "/boot", "/boot/efi")
	inputs := makeFakeContainerInputs()

	stage, err := osbuild.NewBootcInstallToFilesystemStage(inputs, devices, mounts)
	require.Nil(t, err)
	stageJson, err := json.MarshalIndent(stage, "", "  ")
	require.Nil(t, err)
	assert.Equal(t, string(stageJson), `{
  "type": "org.osbuild.bootc.install-to-filesystem",
  "inputs": {
    "images": {
      "type": "org.osbuild.containers",
      "origin": "org.osbuild.source",
      "references": {
        "id-0": {
          "name": "local-name"
        }
      }
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
