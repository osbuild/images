package manifest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestISOBoot(t *testing.T) {
	options := xorrisofsStageOptions("boot.iso", ISOCustomizations{Label: "test-iso-1", BootType: Grub2UEFIOnlyISOBoot}, false)
	assert.Nil(t, options.Boot)
	assert.Equal(t, "", options.IsohybridMBR)
	assert.Equal(t, "", options.Grub2MBR)

	options = xorrisofsStageOptions("boot.iso", ISOCustomizations{Label: "test-iso-1", BootType: SyslinuxISOBoot}, false)
	require.NotNil(t, options.Boot)
	assert.Equal(t, "isolinux/isolinux.bin", options.Boot.Image)
	assert.Equal(t, "/usr/share/syslinux/isohdpfx.bin", options.IsohybridMBR)
	assert.Equal(t, "", options.Grub2MBR)

	options = xorrisofsStageOptions("boot.iso", ISOCustomizations{Label: "test-iso-1", BootType: Grub2ISOBoot}, false)
	require.NotNil(t, options.Boot)
	assert.Equal(t, "images/eltorito.img", options.Boot.Image)
	assert.Equal(t, "/usr/lib/grub/i386-pc/boot_hybrid.img", options.Grub2MBR)
	assert.Equal(t, "", options.IsohybridMBR)

	options = xorrisofsStageOptions("boot.iso", ISOCustomizations{Label: "test-iso-1", BootType: Grub2ISOBoot, Preparer: "Test", Publisher: "Tester"}, false)
	require.NotNil(t, options.Boot)
	assert.Equal(t, "images/eltorito.img", options.Boot.Image)
	assert.Equal(t, "/usr/lib/grub/i386-pc/boot_hybrid.img", options.Grub2MBR)
	assert.Equal(t, "", options.IsohybridMBR)
	assert.Equal(t, "Test", options.Prep)
	assert.Equal(t, "Tester", options.Pub)

	// XXX TODO expand with efiImage true tests
}
