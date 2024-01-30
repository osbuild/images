package osbuild_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/pkg/osbuild"
)

func TestNewMounts(t *testing.T) {
	assert := assert.New(t)

	{ // btrfs
		actual := osbuild.NewBtrfsMount("btrfs", "/dev/sda1", "/mnt/btrfs")
		expected := &osbuild.Mount{
			Name:   "btrfs",
			Type:   "org.osbuild.btrfs",
			Source: "/dev/sda1",
			Target: "/mnt/btrfs",
		}
		assert.Equal(expected, actual)
	}

	{ // ext4
		actual := osbuild.NewExt4Mount("ext4", "/dev/sda2", "/mnt/ext4")
		expected := &osbuild.Mount{
			Name:   "ext4",
			Type:   "org.osbuild.ext4",
			Source: "/dev/sda2",
			Target: "/mnt/ext4",
		}
		assert.Equal(expected, actual)
	}

	{ // fat
		actual := osbuild.NewFATMount("fat", "/dev/sda3", "/mnt/fat")
		expected := &osbuild.Mount{
			Name:   "fat",
			Type:   "org.osbuild.fat",
			Source: "/dev/sda3",
			Target: "/mnt/fat",
		}
		assert.Equal(expected, actual)
	}

	{ // xfs
		actual := osbuild.NewXfsMount("xfs", "/dev/sda4", "/mnt/xfs")
		expected := &osbuild.Mount{
			Name:   "xfs",
			Type:   "org.osbuild.xfs",
			Source: "/dev/sda4",
			Target: "/mnt/xfs",
		}
		assert.Equal(expected, actual)
	}
}
