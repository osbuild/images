package fedora

import (
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/customizations/oscap"
	"github.com/osbuild/images/pkg/datasizes"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/defs"
	"github.com/osbuild/images/pkg/platform"
	"github.com/osbuild/images/pkg/runner"
)

const (
	// package set names

	// main/common os image package set name
	osPkgsKey = "os"

	// container package set name
	containerPkgsKey = "container"

	// installer package set name
	installerPkgsKey = "installer"

	// blueprint package set name
	blueprintPkgsKey = "blueprint"
)

var (
	oscapProfileAllowList = []oscap.Profile{
		oscap.Ospp,
		oscap.PciDss,
		oscap.Standard,
	}

	// Default directory size minimums for all image types.
	requiredDirectorySizes = map[string]uint64{
		"/":    1 * datasizes.GiB,
		"/usr": 2 * datasizes.GiB,
	}
)

// Image Definitions
func mkImageInstallerImgType(d distribution) imageType {
	return imageType{
		name:               "minimal-installer",
		nameAliases:        []string{"image-installer", "fedora-image-installer"},
		filename:           "installer.iso",
		mimeType:           "application/x-iso9660-image",
		packageSets:        packageSetLoader,
		defaultImageConfig: imageConfig(d, "minimal-installer"),
		bootable:           true,
		bootISO:            true,
		rpmOstree:          false,
		image:              imageInstallerImage,
		// We don't know the variant of the OS pipeline being installed
		isoLabel:               getISOLabelFunc("Unknown"),
		buildPipelines:         []string{"build"},
		payloadPipelines:       []string{"anaconda-tree", "efiboot-tree", "os", "bootiso-tree", "bootiso"},
		exports:                []string{"bootiso"},
		requiredPartitionSizes: requiredDirectorySizes,
	}
}

func mkLiveInstallerImgType(d distribution) imageType {
	return imageType{
		name:                   "workstation-live-installer",
		nameAliases:            []string{"live-installer"},
		filename:               "live-installer.iso",
		mimeType:               "application/x-iso9660-image",
		packageSets:            packageSetLoader,
		defaultImageConfig:     imageConfig(d, "workstation-live-installer"),
		bootable:               true,
		bootISO:                true,
		rpmOstree:              false,
		image:                  liveInstallerImage,
		isoLabel:               getISOLabelFunc("Workstation"),
		buildPipelines:         []string{"build"},
		payloadPipelines:       []string{"anaconda-tree", "efiboot-tree", "bootiso-tree", "bootiso"},
		exports:                []string{"bootiso"},
		requiredPartitionSizes: requiredDirectorySizes,
	}
}

func mkIotCommitImgType(d distribution) imageType {
	return imageType{
		name:                   "iot-commit",
		nameAliases:            []string{"fedora-iot-commit"},
		filename:               "commit.tar",
		mimeType:               "application/x-tar",
		packageSets:            packageSetLoader,
		defaultImageConfig:     imageConfig(d, "iot-commit"),
		rpmOstree:              true,
		image:                  iotCommitImage,
		buildPipelines:         []string{"build"},
		payloadPipelines:       []string{"os", "ostree-commit", "commit-archive"},
		exports:                []string{"commit-archive"},
		requiredPartitionSizes: requiredDirectorySizes,
	}
}

func mkIotBootableContainer(d distribution) imageType {
	return imageType{
		name:                   "iot-bootable-container",
		filename:               "iot-bootable-container.tar",
		mimeType:               "application/x-tar",
		packageSets:            packageSetLoader,
		defaultImageConfig:     imageConfig(d, "iot-bootable-container"),
		rpmOstree:              true,
		image:                  bootableContainerImage,
		buildPipelines:         []string{"build"},
		payloadPipelines:       []string{"os", "ostree-commit", "ostree-encapsulate"},
		exports:                []string{"ostree-encapsulate"},
		requiredPartitionSizes: requiredDirectorySizes,
	}
}

func mkIotOCIImgType(d distribution) imageType {
	return imageType{
		name:                   "iot-container",
		nameAliases:            []string{"fedora-iot-container"},
		filename:               "container.tar",
		mimeType:               "application/x-tar",
		packageSets:            packageSetLoader,
		defaultImageConfig:     imageConfig(d, "iot-container"),
		rpmOstree:              true,
		bootISO:                false,
		image:                  iotContainerImage,
		buildPipelines:         []string{"build"},
		payloadPipelines:       []string{"os", "ostree-commit", "container-tree", "container"},
		exports:                []string{"container"},
		requiredPartitionSizes: requiredDirectorySizes,
	}
}

func mkIotInstallerImgType(d distribution) imageType {
	return imageType{
		name:                   "iot-installer",
		nameAliases:            []string{"fedora-iot-installer"},
		filename:               "installer.iso",
		mimeType:               "application/x-iso9660-image",
		packageSets:            packageSetLoader,
		defaultImageConfig:     imageConfig(d, "iot-installer"),
		rpmOstree:              true,
		bootISO:                true,
		image:                  iotInstallerImage,
		isoLabel:               getISOLabelFunc("IoT"),
		buildPipelines:         []string{"build"},
		payloadPipelines:       []string{"anaconda-tree", "efiboot-tree", "bootiso-tree", "bootiso"},
		exports:                []string{"bootiso"},
		requiredPartitionSizes: requiredDirectorySizes,
	}
}

func mkIotSimplifiedInstallerImgType(d distribution) imageType {
	return imageType{
		name:                   "iot-simplified-installer",
		filename:               "simplified-installer.iso",
		mimeType:               "application/x-iso9660-image",
		packageSets:            packageSetLoader,
		defaultImageConfig:     imageConfig(d, "iot-simplified-installer"),
		defaultSize:            10 * datasizes.GibiByte,
		rpmOstree:              true,
		bootable:               true,
		bootISO:                true,
		image:                  iotSimplifiedInstallerImage,
		isoLabel:               getISOLabelFunc("IoT"),
		buildPipelines:         []string{"build"},
		payloadPipelines:       []string{"ostree-deployment", "image", "xz", "coi-tree", "efiboot-tree", "bootiso-tree", "bootiso"},
		exports:                []string{"bootiso"},
		requiredPartitionSizes: requiredDirectorySizes,
	}
}

func mkIotRawImgType(d distribution) imageType {
	return imageType{
		name:               "iot-raw-xz",
		nameAliases:        []string{"iot-raw-image", "fedora-iot-raw-image"},
		filename:           "image.raw.xz",
		compression:        "xz",
		mimeType:           "application/xz",
		packageSets:        nil,
		defaultSize:        4 * datasizes.GibiByte,
		rpmOstree:          true,
		bootable:           true,
		image:              iotImage,
		buildPipelines:     []string{"build"},
		payloadPipelines:   []string{"ostree-deployment", "image", "xz"},
		exports:            []string{"xz"},
		defaultImageConfig: imageConfig(d, "iot-raw-xz"),

		// Passing an empty map into the required partition sizes disables the
		// default partition sizes normally set so our `basePartitionTables` can
		// override them (and make them smaller, in this case).
		requiredPartitionSizes: map[string]uint64{},
	}
}

func mkIotQcow2ImgType(d distribution) imageType {
	return imageType{
		name:                   "iot-qcow2",
		nameAliases:            []string{"iot-qcow2-image"}, // kept for backwards compatibility
		filename:               "image.qcow2",
		mimeType:               "application/x-qemu-disk",
		packageSets:            nil,
		defaultImageConfig:     imageConfig(d, "iot-qcow2"),
		defaultSize:            10 * datasizes.GibiByte,
		rpmOstree:              true,
		bootable:               true,
		image:                  iotImage,
		buildPipelines:         []string{"build"},
		payloadPipelines:       []string{"ostree-deployment", "image", "qcow2"},
		exports:                []string{"qcow2"},
		requiredPartitionSizes: requiredDirectorySizes,
	}
}

func mkWslImgType(d distribution) imageType {
	return imageType{
		name:                   "wsl",
		nameAliases:            []string{"server-wsl"}, // this is the eventual name, and `wsl` the alias but we've been having issues with CI renaming it
		filename:               "wsl.tar",
		mimeType:               "application/x-tar",
		packageSets:            packageSetLoader,
		defaultImageConfig:     imageConfig(d, "wsl"),
		image:                  containerImage,
		bootable:               false,
		buildPipelines:         []string{"build"},
		payloadPipelines:       []string{"os", "container"},
		exports:                []string{"container"},
		requiredPartitionSizes: requiredDirectorySizes,
	}
}

type distribution struct {
	name               string
	product            string
	osVersion          string
	releaseVersion     string
	modulePlatformID   string
	ostreeRefTmpl      string
	runner             runner.Runner
	arches             map[string]distro.Arch
	defaultImageConfig *distro.ImageConfig
}

func defaultDistroInstallerConfig(d *distribution) *distro.InstallerConfig {
	config := distro.InstallerConfig{}
	// In Fedora 42 the ifcfg module was replaced by net-lib.
	if common.VersionLessThan(d.osVersion, "42") {
		config.AdditionalDracutModules = append(config.AdditionalDracutModules, "ifcfg")
	} else {
		config.AdditionalDracutModules = append(config.AdditionalDracutModules, "net-lib")
	}

	return &config
}

func getISOLabelFunc(variant string) isoLabelFunc {
	const ISO_LABEL = "%s-%s-%s-%s"

	return func(t *imageType) string {
		return fmt.Sprintf(ISO_LABEL, t.Arch().Distro().Product(), t.Arch().Distro().OsVersion(), variant, t.Arch().Name())
	}

}

func getDistro(version int) distribution {
	if version < 0 {
		panic("Invalid Fedora version (must be positive)")
	}
	nameVer := fmt.Sprintf("fedora-%d", version)
	return distribution{
		name:               nameVer,
		product:            "Fedora",
		osVersion:          strconv.Itoa(version),
		releaseVersion:     strconv.Itoa(version),
		modulePlatformID:   fmt.Sprintf("platform:f%d", version),
		ostreeRefTmpl:      fmt.Sprintf("fedora/%d/%%s/iot", version),
		runner:             &runner.Fedora{Version: uint64(version)},
		defaultImageConfig: common.Must(defs.DistroImageConfig(nameVer)),
	}
}

func (d *distribution) Name() string {
	return d.name
}

func (d *distribution) Codename() string {
	return "" // Fedora does not use distro codename
}

func (d *distribution) Releasever() string {
	return d.releaseVersion
}

func (d *distribution) OsVersion() string {
	return d.releaseVersion
}

func (d *distribution) Product() string {
	return d.product
}

func (d *distribution) ModulePlatformID() string {
	return d.modulePlatformID
}

func (d *distribution) OSTreeRef() string {
	return d.ostreeRefTmpl
}

func (d *distribution) ListArches() []string {
	archNames := make([]string, 0, len(d.arches))
	for name := range d.arches {
		archNames = append(archNames, name)
	}
	sort.Strings(archNames)
	return archNames
}

func (d *distribution) GetArch(name string) (distro.Arch, error) {
	arch, exists := d.arches[name]
	if !exists {
		return nil, errors.New("invalid architecture: " + name)
	}
	return arch, nil
}

func (d *distribution) addArches(arches ...architecture) {
	if d.arches == nil {
		d.arches = map[string]distro.Arch{}
	}

	// Do not make copies of architectures, as opposed to image types,
	// because architecture definitions are not used by more than a single
	// distro definition.
	for idx := range arches {
		d.arches[arches[idx].name] = &arches[idx]
	}
}

func (d *distribution) getDefaultImageConfig() *distro.ImageConfig {
	return d.defaultImageConfig
}

type architecture struct {
	distro           *distribution
	name             string
	imageTypes       map[string]distro.ImageType
	imageTypeAliases map[string]string
}

func (a *architecture) Name() string {
	return a.name
}

func (a *architecture) ListImageTypes() []string {
	itNames := make([]string, 0, len(a.imageTypes))
	for name := range a.imageTypes {
		itNames = append(itNames, name)
	}
	sort.Strings(itNames)
	return itNames
}

func (a *architecture) GetImageType(name string) (distro.ImageType, error) {
	t, exists := a.imageTypes[name]
	if !exists {
		aliasForName, exists := a.imageTypeAliases[name]
		if !exists {
			return nil, errors.New("invalid image type: " + name)
		}
		t, exists = a.imageTypes[aliasForName]
		if !exists {
			panic(fmt.Sprintf("image type '%s' is an alias to a non-existing image type '%s'", name, aliasForName))
		}
	}
	return t, nil
}

func (a *architecture) addImageTypes(platform platform.Platform, imageTypes ...imageType) {
	if a.imageTypes == nil {
		a.imageTypes = map[string]distro.ImageType{}
	}
	for idx := range imageTypes {
		it := imageTypes[idx]
		it.arch = a
		it.platform = platform
		a.imageTypes[it.name] = &it
		for _, alias := range it.nameAliases {
			if a.imageTypeAliases == nil {
				a.imageTypeAliases = map[string]string{}
			}
			if existingAliasFor, exists := a.imageTypeAliases[alias]; exists {
				panic(fmt.Sprintf("image type alias '%s' for '%s' is already defined for another image type '%s'", alias, it.name, existingAliasFor))
			}
			a.imageTypeAliases[alias] = it.name
		}
	}
}

func (a *architecture) Distro() distro.Distro {
	return a.distro
}

func newDistro(version int) distro.Distro {
	rd := getDistro(version)

	// XXX: generate architecture automatically from the imgType yaml
	x86_64 := architecture{
		name:   arch.ARCH_X86_64.String(),
		distro: &rd,
	}

	aarch64 := architecture{
		name:   arch.ARCH_AARCH64.String(),
		distro: &rd,
	}

	ppc64le := architecture{
		distro: &rd,
		name:   arch.ARCH_PPC64LE.String(),
	}

	s390x := architecture{
		distro: &rd,
		name:   arch.ARCH_S390X.String(),
	}

	riscv64 := architecture{
		name:   arch.ARCH_RISCV64.String(),
		distro: &rd,
	}

	// XXX: move all image types should to YAML
	its, err := defs.ImageTypes(rd.name)
	if err != nil {
		panic(err)
	}
	for _, imgTypeYAML := range its {
		// use as marker for images that are not converted to
		// YAML yet
		if imgTypeYAML.Filename == "" {
			continue
		}
		it := newImageTypeFrom(rd, imgTypeYAML)
		for _, pl := range imgTypeYAML.Platforms {
			switch pl.Arch {
			case arch.ARCH_X86_64:
				x86_64.addImageTypes(&pl, it)
			case arch.ARCH_AARCH64:
				aarch64.addImageTypes(&pl, it)
			case arch.ARCH_PPC64LE:
				ppc64le.addImageTypes(&pl, it)
			case arch.ARCH_S390X:
				s390x.addImageTypes(&pl, it)
			case arch.ARCH_RISCV64:
				riscv64.addImageTypes(&pl, it)
			default:
				err := fmt.Errorf("unsupported arch: %v", pl.Arch)
				panic(err)
			}
		}
	}

	x86_64.addImageTypes(
		&platform.X86{},
		mkWslImgType(rd),
	)

	// add distro installer configuration to all installer types
	distroInstallerConfig := defaultDistroInstallerConfig(&rd)

	liveInstallerImgType := mkLiveInstallerImgType(rd)
	liveInstallerImgType.defaultInstallerConfig = distroInstallerConfig

	imageInstallerImgType := mkImageInstallerImgType(rd)
	imageInstallerImgType.defaultInstallerConfig = distroInstallerConfig

	iotInstallerImgType := mkIotInstallerImgType(rd)
	iotInstallerImgType.defaultInstallerConfig = distroInstallerConfig

	x86_64.addImageTypes(
		&platform.X86{
			BasePlatform: platform.BasePlatform{
				FirmwarePackages: []string{
					"biosdevname",
					"iwlwifi-dvm-firmware",
					"iwlwifi-mvm-firmware",
					"microcode_ctl",
				},
			},
			BIOS:       true,
			UEFIVendor: "fedora",
		},
		mkIotOCIImgType(rd),
		mkIotCommitImgType(rd),
		iotInstallerImgType,
		imageInstallerImgType,
		liveInstallerImgType,
	)
	x86_64.addImageTypes(
		&platform.X86{
			BasePlatform: platform.BasePlatform{
				ImageFormat: platform.FORMAT_RAW,
			},
			BIOS:       false,
			UEFIVendor: "fedora",
		},
		mkIotRawImgType(rd),
	)
	x86_64.addImageTypes(
		&platform.X86{
			BasePlatform: platform.BasePlatform{
				ImageFormat: platform.FORMAT_QCOW2,
			},
			BIOS:       false,
			UEFIVendor: "fedora",
		},
		mkIotQcow2ImgType(rd),
	)
	aarch64.addImageTypes(
		&platform.Aarch64{
			UEFIVendor: "fedora",
			BasePlatform: platform.BasePlatform{
				ImageFormat: platform.FORMAT_QCOW2,
				QCOW2Compat: "1.1",
			},
		},
		mkIotQcow2ImgType(rd),
	)
	aarch64.addImageTypes(
		&platform.Aarch64{
			UEFIVendor: "fedora",
			BasePlatform: platform.BasePlatform{
				ImageFormat: platform.FORMAT_QCOW2,
			},
		},
	)
	aarch64.addImageTypes(
		&platform.Aarch64{
			BasePlatform: platform.BasePlatform{
				FirmwarePackages: []string{
					"arm-image-installer",
					"bcm283x-firmware",
					"brcmfmac-firmware",
					"iwlwifi-mvm-firmware",
					"realtek-firmware",
					"uboot-images-armv8",
				},
			},
			UEFIVendor: "fedora",
		},
		imageInstallerImgType,
		mkIotCommitImgType(rd),
		iotInstallerImgType,
		mkIotOCIImgType(rd),
		liveInstallerImgType,
	)
	aarch64.addImageTypes(
		&platform.Aarch64_Fedora{
			BasePlatform: platform.BasePlatform{
				ImageFormat: platform.FORMAT_RAW,
			},
			UEFIVendor: "fedora",
			BootFiles: [][2]string{
				{"/usr/lib/ostree-boot/efi/bcm2710-rpi-2-b.dtb", "/boot/efi/"},
				{"/usr/lib/ostree-boot/efi/bcm2710-rpi-3-b-plus.dtb", "/boot/efi/"},
				{"/usr/lib/ostree-boot/efi/bcm2710-rpi-3-b.dtb", "/boot/efi/"},
				{"/usr/lib/ostree-boot/efi/bcm2710-rpi-cm3.dtb", "/boot/efi/"},
				{"/usr/lib/ostree-boot/efi/bcm2710-rpi-zero-2-w.dtb", "/boot/efi/"},
				{"/usr/lib/ostree-boot/efi/bcm2710-rpi-zero-2.dtb", "/boot/efi/"},
				{"/usr/lib/ostree-boot/efi/bcm2711-rpi-4-b.dtb", "/boot/efi/"},
				{"/usr/lib/ostree-boot/efi/bcm2711-rpi-400.dtb", "/boot/efi/"},
				{"/usr/lib/ostree-boot/efi/bcm2711-rpi-cm4.dtb", "/boot/efi/"},
				{"/usr/lib/ostree-boot/efi/bcm2711-rpi-cm4s.dtb", "/boot/efi/"},
				{"/usr/lib/ostree-boot/efi/bootcode.bin", "/boot/efi/"},
				{"/usr/lib/ostree-boot/efi/config.txt", "/boot/efi/config.txt"},
				{"/usr/lib/ostree-boot/efi/fixup.dat", "/boot/efi/"},
				{"/usr/lib/ostree-boot/efi/fixup4.dat", "/boot/efi/"},
				{"/usr/lib/ostree-boot/efi/fixup4cd.dat", "/boot/efi/"},
				{"/usr/lib/ostree-boot/efi/fixup4db.dat", "/boot/efi/"},
				{"/usr/lib/ostree-boot/efi/fixup4x.dat", "/boot/efi/"},
				{"/usr/lib/ostree-boot/efi/fixup_cd.dat", "/boot/efi/"},
				{"/usr/lib/ostree-boot/efi/fixup_db.dat", "/boot/efi/"},
				{"/usr/lib/ostree-boot/efi/fixup_x.dat", "/boot/efi/"},
				{"/usr/lib/ostree-boot/efi/overlays", "/boot/efi/"},
				{"/usr/share/uboot/rpi_arm64/u-boot.bin", "/boot/efi/rpi-u-boot.bin"},
				{"/usr/lib/ostree-boot/efi/start.elf", "/boot/efi/"},
				{"/usr/lib/ostree-boot/efi/start4.elf", "/boot/efi/"},
				{"/usr/lib/ostree-boot/efi/start4cd.elf", "/boot/efi/"},
				{"/usr/lib/ostree-boot/efi/start4db.elf", "/boot/efi/"},
				{"/usr/lib/ostree-boot/efi/start4x.elf", "/boot/efi/"},
				{"/usr/lib/ostree-boot/efi/start_cd.elf", "/boot/efi/"},
				{"/usr/lib/ostree-boot/efi/start_db.elf", "/boot/efi/"},
				{"/usr/lib/ostree-boot/efi/start_x.elf", "/boot/efi/"},
			},
		},
		mkIotRawImgType(rd),
	)

	iotSimplifiedInstallerImgType := mkIotSimplifiedInstallerImgType(rd)
	iotSimplifiedInstallerImgType.defaultInstallerConfig = distroInstallerConfig

	x86_64.addImageTypes(
		&platform.X86{
			BasePlatform: platform.BasePlatform{
				ImageFormat: platform.FORMAT_RAW,
				FirmwarePackages: []string{
					"grub2-efi-x64",
					"grub2-efi-x64-cdboot",
					"grub2-tools",
					"grub2-tools-minimal",
					"efibootmgr",
					"shim-x64",
					"brcmfmac-firmware",
					"iwlwifi-dvm-firmware",
					"iwlwifi-mvm-firmware",
					"realtek-firmware",
					"microcode_ctl",
				},
			},
			BIOS:       false,
			UEFIVendor: "fedora",
		},
		iotSimplifiedInstallerImgType,
	)

	aarch64.addImageTypes(
		&platform.Aarch64{
			BasePlatform: platform.BasePlatform{
				FirmwarePackages: []string{
					"arm-image-installer",
					"bcm283x-firmware",
					"grub2-efi-aa64",
					"grub2-efi-aa64-cdboot",
					"grub2-tools",
					"grub2-tools-minimal",
					"efibootmgr",
					"shim-aa64",
					"brcmfmac-firmware",
					"iwlwifi-dvm-firmware",
					"iwlwifi-mvm-firmware",
					"realtek-firmware",
					"uboot-images-armv8",
				},
			},
			UEFIVendor: "fedora",
		},
		iotSimplifiedInstallerImgType,
	)

	x86_64.addImageTypes(
		&platform.X86{
			BasePlatform: platform.BasePlatform{
				FirmwarePackages: []string{
					"biosdevname",
					"iwlwifi-dvm-firmware",
					"iwlwifi-mvm-firmware",
					"microcode_ctl",
				},
			},
			BIOS:       true,
			UEFIVendor: "fedora",
		},
		mkIotBootableContainer(rd),
	)
	aarch64.addImageTypes(
		&platform.Aarch64{
			BasePlatform: platform.BasePlatform{
				FirmwarePackages: []string{
					"arm-image-installer",
					"bcm283x-firmware",
					"brcmfmac-firmware",
					"iwlwifi-mvm-firmware",
					"realtek-firmware",
					"uboot-images-armv8",
				},
			},
			UEFIVendor: "fedora",
		},
		mkIotBootableContainer(rd),
	)

	ppc64le.addImageTypes(
		&platform.PPC64LE{
			BIOS: true,
			BasePlatform: platform.BasePlatform{
				ImageFormat: platform.FORMAT_QCOW2,
				QCOW2Compat: "1.1",
			},
		},
		mkIotBootableContainer(rd),
	)

	s390x.addImageTypes(
		&platform.S390X{
			Zipl: true,
			BasePlatform: platform.BasePlatform{
				ImageFormat: platform.FORMAT_QCOW2,
				QCOW2Compat: "1.1",
			},
		},
		mkIotBootableContainer(rd),
	)

	rd.addArches(x86_64, aarch64, ppc64le, s390x, riscv64)
	return &rd
}

func ParseID(idStr string) (*distro.ID, error) {
	id, err := distro.ParseID(idStr)
	if err != nil {
		return nil, err
	}

	if id.Name != "fedora" {
		return nil, fmt.Errorf("invalid distro name: %s", id.Name)
	}

	if id.MinorVersion != -1 {
		return nil, fmt.Errorf("fedora distro does not support minor versions")
	}

	return id, nil
}

func DistroFactory(idStr string) distro.Distro {
	id, err := ParseID(idStr)
	if err != nil {
		return nil
	}

	return newDistro(id.MajorVersion)
}
