package bootc

import (
	"fmt"
	"math/rand"
	"slices"
	"strconv"
	"strings"

	"github.com/osbuild/blueprint/pkg/blueprint"
	"github.com/osbuild/images/internal/cmdutil"
	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/bib/osinfo"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/customizations/anaconda"
	"github.com/osbuild/images/pkg/customizations/kickstart"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/image"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/olog"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/platform"
	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/osbuild/images/pkg/runner"
)

var _ = distro.ImageType(&BootcAnacondaInstaller{})

// XXX: rename to BootcAnacondaInstaller
type BootcAnacondaInstaller struct {
	arch *BootcArch

	name   string
	export string
}

func (t *BootcAnacondaInstaller) Name() string {
	return t.name
}

func (t *BootcAnacondaInstaller) Aliases() []string {
	return nil
}

func (t *BootcAnacondaInstaller) Arch() distro.Arch {
	return t.arch
}

func (t *BootcAnacondaInstaller) Filename() string {
	return "installer.iso"
}

func (t *BootcAnacondaInstaller) MIMEType() string {
	return "application/x-iso9660-image"
}

func (t *BootcAnacondaInstaller) OSTreeRef() string {
	return ""
}

func (t *BootcAnacondaInstaller) ISOLabel() (string, error) {
	return "Unknown", nil
}

func (t *BootcAnacondaInstaller) Size(size uint64) uint64 {
	return size
}

func (t *BootcAnacondaInstaller) PartitionType() disk.PartitionTableType {
	return disk.PT_NONE
}

func (t *BootcAnacondaInstaller) BasePartitionTable() (*disk.PartitionTable, error) {
	return nil, nil
}

func (t *BootcAnacondaInstaller) BootMode() platform.BootMode {
	return platform.BOOT_HYBRID
}

func (t *BootcAnacondaInstaller) BuildPipelines() []string {
	return []string{"build"}
}

func (t *BootcAnacondaInstaller) PayloadPipelines() []string {
	return []string{""}
}

func (t *BootcAnacondaInstaller) PayloadPackageSets() []string {
	return nil
}

func (t *BootcAnacondaInstaller) Exports() []string {
	return []string{t.export}
}

func (t *BootcAnacondaInstaller) SupportedBlueprintOptions() []string {
	// XXX: ?
	return []string{
		"customizations.directories",
		"customizations.disk",
		"customizations.files",
		"customizations.filesystem",
		"customizations.group",
		"customizations.kernel",
		"customizations.user",
	}
}
func (t *BootcAnacondaInstaller) RequiredBlueprintOptions() []string {
	return nil
}

// XXX: duplication with BootcImageType
func (t *BootcAnacondaInstaller) Manifest(bp *blueprint.Blueprint, options distro.ImageOptions, repos []rpmmd.RepoConfig, seedp *int64) (*manifest.Manifest, []string, error) {
	if t.arch.distro.imgref == "" {
		return nil, nil, fmt.Errorf("internal error: no base image defined")
	}
	containerSource := container.SourceSpec{
		Source: t.arch.distro.imgref,
		Name:   t.arch.distro.imgref,
		Local:  true,
	}
	// XXX: keep it simple for now, we may allow this in the future
	if t.arch.distro.buildImgref != t.arch.distro.imgref {
		return nil, nil, fmt.Errorf("cannot use build-containers with anaconda installer images")
	}

	var customizations *blueprint.Customizations
	if bp != nil {
		customizations = bp.Customizations
	}
	seed, err := cmdutil.SeedArgFor(nil, t.Name(), t.arch.Name(), t.arch.distro.Name())
	if err != nil {
		return nil, nil, err
	}
	//nolint:gosec
	rng := rand.New(rand.NewSource(seed))

	archi := common.Must(arch.FromString(t.arch.Name()))
	platform := &platform.Data{
		Arch:        archi,
		UEFIVendor:  t.arch.distro.sourceInfo.UEFIVendor,
		QCOW2Compat: "1.1",
	}
	switch archi {
	case arch.ARCH_X86_64:
		platform.BIOSPlatform = "i386-pc"
	case arch.ARCH_PPC64LE:
		platform.BIOSPlatform = "powerpc-ieee1275"
	case arch.ARCH_S390X:
		platform.ZiplSupport = true
	}
	// XXX: tons of copied code from
	// bootc-image-builder:‎bib/cmd/bootc-image-builder/image.go
	filename := "install.iso"

	// The ref is not needed and will be removed from the ctor later
	// in time
	img := image.NewAnacondaContainerInstaller(platform, filename, containerSource, "")
	img.ContainerRemoveSignatures = true
	img.RootfsCompression = "zstd"
	// kernelVer is used by dracut
	img.KernelVer = t.arch.distro.sourceInfo.KernelInfo.Version
	img.KernelPath = fmt.Sprintf("lib/modules/%s/vmlinuz", t.arch.distro.sourceInfo.KernelInfo.Version)
	img.InitramfsPath = fmt.Sprintf("lib/modules/%s/initramfs.img", t.arch.distro.sourceInfo.KernelInfo.Version)
	img.InstallerHome = "/var/roothome"
	payloadSource := container.SourceSpec{
		Source: t.arch.distro.payloadRef,
		Name:   t.arch.distro.payloadRef,
		Local:  true,
	}
	img.InstallerPayload = payloadSource

	if t.arch.Name() == arch.ARCH_X86_64.String() {
		img.InstallerCustomizations.ISOBoot = manifest.Grub2ISOBoot
	}

	img.InstallerCustomizations.Product = t.arch.distro.sourceInfo.OSRelease.Name
	img.InstallerCustomizations.OSVersion = t.arch.distro.sourceInfo.OSRelease.VersionID
	img.InstallerCustomizations.ISOLabel = labelForISO(&t.arch.distro.sourceInfo.OSRelease, t.arch.Name())

	img.InstallerCustomizations.FIPS = customizations.GetFIPS()
	img.Kickstart, err = kickstart.New(customizations)
	if err != nil {
		return nil, nil, err
	}
	img.Kickstart.Path = osbuild.KickstartPathOSBuild
	if kopts := customizations.GetKernel(); kopts != nil && kopts.Append != "" {
		img.Kickstart.KernelOptionsAppend = append(img.Kickstart.KernelOptionsAppend, kopts.Append)
	}
	img.Kickstart.NetworkOnBoot = true

	instCust, err := customizations.GetInstaller()
	if err != nil {
		return nil, nil, err
	}
	if instCust != nil && instCust.Modules != nil {
		img.InstallerCustomizations.EnabledAnacondaModules = append(img.InstallerCustomizations.EnabledAnacondaModules, instCust.Modules.Enable...)
		img.InstallerCustomizations.DisabledAnacondaModules = append(img.InstallerCustomizations.DisabledAnacondaModules, instCust.Modules.Disable...)
	}
	img.InstallerCustomizations.EnabledAnacondaModules = append(img.InstallerCustomizations.EnabledAnacondaModules,
		anaconda.ModuleUsers,
		anaconda.ModuleServices,
		anaconda.ModuleSecurity,
		// XXX: get from the imagedefs
		anaconda.ModuleNetwork,
		anaconda.ModulePayloads,
		anaconda.ModuleRuntime,
		anaconda.ModuleStorage,
	)

	img.Kickstart.OSTree = &kickstart.OSTree{
		OSName: "default",
	}
	img.InstallerCustomizations.UseRHELLoraxTemplates = needsRHELLoraxTemplates(t.arch.distro.sourceInfo.OSRelease)

	// see https://github.com/osbuild/bootc-image-builder/issues/733
	img.InstallerCustomizations.ISORootfsType = manifest.SquashfsRootfs

	installRootfsType, err := disk.NewFSType(t.arch.distro.defaultFs)
	if err != nil {
		return nil, nil, err
	}
	img.InstallRootfsType = installRootfsType

	mf := manifest.New()

	foundDistro, foundRunner, err := getDistroAndRunner(t.arch.distro.sourceInfo.OSRelease)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to infer distro and runner: %w", err)
	}
	mf.Distro = foundDistro

	_, err = img.InstantiateManifestFromContainer(&mf, []container.SourceSpec{containerSource}, foundRunner, rng)
	return &mf, nil, err
}

func labelForISO(os *osinfo.OSRelease, arch string) string {
	switch os.ID {
	case "fedora":
		return fmt.Sprintf("Fedora-S-dvd-%s-%s", arch, os.VersionID)
	case "centos":
		labelTemplate := "CentOS-Stream-%s-BaseOS-%s"
		if os.VersionID == "8" {
			labelTemplate = "CentOS-Stream-%s-%s-dvd"
		}
		return fmt.Sprintf(labelTemplate, os.VersionID, arch)
	case "rhel":
		version := strings.ReplaceAll(os.VersionID, ".", "-")
		return fmt.Sprintf("RHEL-%s-BaseOS-%s", version, arch)
	default:
		return fmt.Sprintf("Container-Installer-%s", arch)
	}
}

func getDistroAndRunner(osRelease osinfo.OSRelease) (manifest.Distro, runner.Runner, error) {
	switch osRelease.ID {
	case "fedora":
		version, err := strconv.ParseUint(osRelease.VersionID, 10, 64)
		if err != nil {
			return manifest.DISTRO_NULL, nil, fmt.Errorf("cannot parse Fedora version (%s): %w", osRelease.VersionID, err)
		}

		return manifest.DISTRO_FEDORA, &runner.Fedora{
			Version: version,
		}, nil
	case "centos":
		version, err := strconv.ParseUint(osRelease.VersionID, 10, 64)
		if err != nil {
			return manifest.DISTRO_NULL, nil, fmt.Errorf("cannot parse CentOS version (%s): %w", osRelease.VersionID, err)
		}
		r := &runner.CentOS{
			Version: version,
		}
		switch version {
		case 9:
			return manifest.DISTRO_EL9, r, nil
		case 10:
			return manifest.DISTRO_EL10, r, nil
		default:
			olog.Printf("Unknown CentOS version %d, using default distro for manifest generation", version)
			return manifest.DISTRO_NULL, r, nil
		}

	case "rhel":
		versionParts := strings.Split(osRelease.VersionID, ".")
		if len(versionParts) != 2 {
			return manifest.DISTRO_NULL, nil, fmt.Errorf("invalid RHEL version format: %s", osRelease.VersionID)
		}
		major, err := strconv.ParseUint(versionParts[0], 10, 64)
		if err != nil {
			return manifest.DISTRO_NULL, nil, fmt.Errorf("cannot parse RHEL major version (%s): %w", versionParts[0], err)
		}
		minor, err := strconv.ParseUint(versionParts[1], 10, 64)
		if err != nil {
			return manifest.DISTRO_NULL, nil, fmt.Errorf("cannot parse RHEL minor version (%s): %w", versionParts[1], err)
		}
		r := &runner.RHEL{
			Major: major,
			Minor: minor,
		}
		switch major {
		case 9:
			return manifest.DISTRO_EL9, r, nil
		case 10:
			return manifest.DISTRO_EL10, r, nil
		default:
			olog.Printf("Unknown RHEL version %d, using default distro for manifest generation", major)
			return manifest.DISTRO_NULL, r, nil
		}
	}

	olog.Printf("Unknown distro %s, using default runner", osRelease.ID)
	return manifest.DISTRO_NULL, &runner.Linux{}, nil
}

func needsRHELLoraxTemplates(si osinfo.OSRelease) bool {
	return si.ID == "rhel" || slices.Contains(si.IDLike, "rhel") || si.VersionID == "eln"
}
