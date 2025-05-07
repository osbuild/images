package platform_test

import (
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/platform"
)

func TestPlatformYamlSmoke(t *testing.T) {
	inputYAML := []byte(`
        arch: "x86_64"
        bios_platform: i386-pc
        uefi_vendor: "fedora"
        image_format: "qcow2"
        qcow2_compat: "1.1"
        packages:
          - grub2-pc
        build_packages:
          - grub2-pc-as-bp
        boot_files:
          - ["/usr/share/uboot/rpi_arm64/u-boot.bin", "/boot/efi/rpi-u-boot.bin"]
`)
	var pc platform.PlatformConf
	err := yaml.Unmarshal(inputYAML, &pc)
	assert.NoError(t, err)
	expected := platform.PlatformConf{
		Arch:          common.Must(arch.FromString("x86_64")),
		BIOSPlatform:  "i386-pc",
		UEFIVendor:    "fedora",
		ImageFormat:   platform.FORMAT_QCOW2,
		QCOW2Compat:   "1.1",
		Packages:      []string{"grub2-pc"},
		BuildPackages: []string{"grub2-pc-as-bp"},
		BootFiles: [][2]string{
			{"/usr/share/uboot/rpi_arm64/u-boot.bin", "/boot/efi/rpi-u-boot.bin"},
		},
	}
	assert.Equal(t, expected, pc)
}
