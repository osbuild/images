package platform

import (
	"github.com/osbuild/images/pkg/arch"
)

// Data is a platform configured from YAML inputs
// that implements the "Platform" interface
type Data struct {
	Arch         arch.Arch   `yaml:"arch"`
	ImageFormat  ImageFormat `yaml:"image_format"`
	QCOW2Compat  string      `yaml:"qcow2_compat"`
	BIOSPlatform string      `yaml:"bios_platform"`
	UEFIVendor   string      `yaml:"uefi_vendor"`
	ZiplSupport  bool        `yaml:"zipl_support"`
	// packages are index by an arbitrary string key to
	// make them YAML mergable, a good key is e.g. "bios"
	// to indicate that these packages are needed for
	// bios support
	Packages      map[string][]string `yaml:"packages"`
	BuildPackages map[string][]string `yaml:"build_packages"`
	BootFiles     [][2]string         `yaml:"boot_files"`

	Bootloader Bootloader `yaml:"bootloader"`
	FIPSMenu   bool       `yaml:"fips_menu"` // Add FIPS entry to iso bootloader menu
}

// ensure platform.Data implements the Platform interface
var _ = Platform(&Data{})

func (pc *Data) GetArch() arch.Arch {
	return pc.Arch
}
func (pc *Data) GetImageFormat() ImageFormat {
	return pc.ImageFormat
}
func (pc *Data) GetQCOW2Compat() string {
	return pc.QCOW2Compat
}
func (pc *Data) GetBIOSPlatform() string {
	return pc.BIOSPlatform
}
func (pc *Data) GetUEFIVendor() string {
	return pc.UEFIVendor
}
func (pc *Data) GetZiplSupport() bool {
	return pc.ZiplSupport
}
func (pc *Data) GetPackages() []string {
	var merged []string
	for _, pkgList := range pc.Packages {
		merged = append(merged, pkgList...)
	}
	return merged
}
func (pc *Data) GetBuildPackages() []string {
	var merged []string
	for _, pkgList := range pc.BuildPackages {
		merged = append(merged, pkgList...)
	}
	return merged
}
func (pc *Data) GetBootFiles() [][2]string {
	return pc.BootFiles
}

func (pc *Data) GetBootloader() Bootloader {
	return pc.Bootloader
}

// GetFIPSMenu is used to add the FIPS entry to the iso bootloader menu
func (pc *PlatformConf) GetFIPSMenu() bool {
	return pc.FIPSMenu
}
