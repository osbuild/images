package fedora

import (
	"fmt"
	"math/rand"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/internal/workload"
	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/customizations/anaconda"
	"github.com/osbuild/images/pkg/customizations/bootc"
	"github.com/osbuild/images/pkg/customizations/fdo"
	"github.com/osbuild/images/pkg/customizations/fsnode"
	"github.com/osbuild/images/pkg/customizations/ignition"
	"github.com/osbuild/images/pkg/customizations/kickstart"
	"github.com/osbuild/images/pkg/customizations/users"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/image"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/ostree"
	"github.com/osbuild/images/pkg/rpmmd"
)

// HELPERS

func ostreeDeploymentCustomizations(
	t *imageType,
	c *blueprint.Customizations) (manifest.OSTreeDeploymentCustomizations, error) {

	if !t.distroConfig.RpmOstree || !t.distroConfig.Bootable {
		return manifest.OSTreeDeploymentCustomizations{}, fmt.Errorf("ostree deployment customizations are only supported for bootable rpm-ostree images")
	}

	imageConfig := t.getDefaultImageConfig()
	deploymentConf := manifest.OSTreeDeploymentCustomizations{}

	var kernelOptions []string
	if len(t.distroConfig.KernelOptions) > 0 {
		kernelOptions = append(kernelOptions, t.distroConfig.KernelOptions...)
	}
	if bpKernel := c.GetKernel(); bpKernel != nil && bpKernel.Append != "" {
		kernelOptions = append(kernelOptions, bpKernel.Append)
	}

	if imageConfig.IgnitionPlatform != nil {
		deploymentConf.IgnitionPlatform = *imageConfig.IgnitionPlatform
	}

	switch deploymentConf.IgnitionPlatform {
	case "metal":
		if bpIgnition := c.GetIgnition(); bpIgnition != nil && bpIgnition.FirstBoot != nil && bpIgnition.FirstBoot.ProvisioningURL != "" {
			kernelOptions = append(kernelOptions, "ignition.config.url="+bpIgnition.FirstBoot.ProvisioningURL)
		}
	}
	deploymentConf.KernelOptionsAppend = kernelOptions

	deploymentConf.FIPS = c.GetFIPS()

	deploymentConf.Users = users.UsersFromBP(c.GetUsers())
	deploymentConf.Groups = users.GroupsFromBP(c.GetGroups())

	var err error
	deploymentConf.Directories, err = blueprint.DirectoryCustomizationsToFsNodeDirectories(c.GetDirectories())
	if err != nil {
		return manifest.OSTreeDeploymentCustomizations{}, err
	}
	deploymentConf.Files, err = blueprint.FileCustomizationsToFsNodeFiles(c.GetFiles())
	if err != nil {
		return manifest.OSTreeDeploymentCustomizations{}, err
	}

	language, keyboard := c.GetPrimaryLocale()
	if language != nil {
		deploymentConf.Locale = *language
	} else if imageConfig.Locale != nil {
		deploymentConf.Locale = *imageConfig.Locale
	}
	if keyboard != nil {
		deploymentConf.Keyboard = *keyboard
	} else if imageConfig.Keyboard != nil {
		deploymentConf.Keyboard = imageConfig.Keyboard.Keymap
	}

	if imageConfig.OSTreeConfSysrootReadOnly != nil {
		deploymentConf.SysrootReadOnly = *imageConfig.OSTreeConfSysrootReadOnly
	}

	if imageConfig.LockRootUser != nil {
		deploymentConf.LockRoot = *imageConfig.LockRootUser
	}

	for _, fs := range c.GetFilesystems() {
		deploymentConf.CustomFileSystems = append(deploymentConf.CustomFileSystems, fs.Mountpoint)
	}

	return deploymentConf, nil
}

// IMAGES

func diskImage(workload workload.Workload,
	t *imageType,
	bp *blueprint.Blueprint,
	options distro.ImageOptions,
	packageSets map[string]rpmmd.PackageSet,
	containers []container.SourceSpec,
	rng *rand.Rand) (image.ImageKind, error) {

	img := image.NewDiskImage()
	img.Platform = t.platform

	var err error
	img.OSCustomizations, err = distro.OsCustomizations(&t.distroConfig, packageSets[osPkgsKey], distro.ImageOptions{}, containers, bp.Customizations)
	if err != nil {
		return nil, err
	}

	img.Environment = t.environment
	img.Workload = workload
	img.Compression = t.compression
	if bp.Minimal {
		// Disable weak dependencies if the 'minimal' option is enabled
		img.OSCustomizations.InstallWeakDeps = false
	}
	// TODO: move generation into LiveImage
	pt, err := t.getPartitionTable(bp.Customizations, options, rng)
	if err != nil {
		return nil, err
	}
	img.PartitionTable = pt

	img.Filename = t.Filename()

	return img, nil
}

func containerImage(workload workload.Workload,
	t *imageType,
	bp *blueprint.Blueprint,
	options distro.ImageOptions,
	packageSets map[string]rpmmd.PackageSet,
	containers []container.SourceSpec,
	rng *rand.Rand) (image.ImageKind, error) {
	img := image.NewBaseContainer()

	img.Platform = t.platform

	var err error
	img.OSCustomizations, err = distro.OsCustomizations(&t.distroConfig, packageSets[osPkgsKey], distro.ImageOptions{}, containers, bp.Customizations)
	if err != nil {
		return nil, err
	}

	img.Environment = t.environment
	img.Workload = workload

	img.Filename = t.Filename()

	return img, nil
}

func liveInstallerImage(workload workload.Workload,
	t *imageType,
	bp *blueprint.Blueprint,
	options distro.ImageOptions,
	packageSets map[string]rpmmd.PackageSet,
	containers []container.SourceSpec,
	rng *rand.Rand) (image.ImageKind, error) {

	img := image.NewAnacondaLiveInstaller()

	img.Platform = t.platform
	img.Workload = workload
	img.ExtraBasePackages = packageSets[installerPkgsKey]

	d := t.arch.distro

	img.Product = d.product
	img.Variant = "Workstation"
	img.OSVersion = d.osVersion
	img.Release = fmt.Sprintf("%s %s", d.product, d.osVersion)
	img.Preview = common.VersionGreaterThanOrEqual(img.OSVersion, VERSION_BRANCHED)

	var err error
	img.ISOLabel, err = t.ISOLabel()
	if err != nil {
		return nil, err
	}

	img.Filename = t.Filename()

	if common.VersionGreaterThanOrEqual(img.OSVersion, VERSION_ROOTFS_SQUASHFS) {
		img.RootfsType = manifest.SquashfsRootfs
	}

	// Enable grub2 BIOS iso on x86_64 only
	if img.Platform.GetArch() == arch.ARCH_X86_64 {
		img.ISOBoot = manifest.Grub2ISOBoot
	}

	if locale := t.getDefaultImageConfig().Locale; locale != nil {
		img.Locale = *locale
	}

	installerConfig, err := t.getDefaultInstallerConfig()
	if err != nil {
		return nil, err
	}

	if installerConfig != nil {
		img.AdditionalDracutModules = append(img.AdditionalDracutModules, installerConfig.AdditionalDracutModules...)
		img.AdditionalDrivers = append(img.AdditionalDrivers, installerConfig.AdditionalDrivers...)
	}

	return img, nil
}

func imageInstallerImage(workload workload.Workload,
	t *imageType,
	bp *blueprint.Blueprint,
	options distro.ImageOptions,
	packageSets map[string]rpmmd.PackageSet,
	containers []container.SourceSpec,
	rng *rand.Rand) (image.ImageKind, error) {

	customizations := bp.Customizations

	img := image.NewAnacondaTarInstaller()

	var err error
	img.OSCustomizations, err = distro.OsCustomizations(&t.distroConfig, packageSets[osPkgsKey], distro.ImageOptions{}, containers, bp.Customizations)
	if err != nil {
		return nil, err
	}

	img.Kickstart, err = kickstart.New(customizations)
	if err != nil {
		return nil, err
	}
	img.Kickstart.Language = &img.OSCustomizations.Language
	img.Kickstart.Keyboard = img.OSCustomizations.Keyboard
	img.Kickstart.Timezone = &img.OSCustomizations.Timezone

	if img.Kickstart.Unattended {
		// NOTE: this is not supported right now because the
		// image-installer on Fedora isn't working when unattended.
		// These options are probably necessary but could change.
		// Unattended/non-interactive installations are better set to text
		// time since they might be running headless and a UI is
		// unnecessary.
		img.AdditionalKernelOpts = []string{"inst.text", "inst.noninteractive"}
	}

	instCust, err := customizations.GetInstaller()
	if err != nil {
		return nil, err
	}
	if instCust != nil && instCust.Modules != nil {
		img.AdditionalAnacondaModules = append(img.AdditionalAnacondaModules, instCust.Modules.Enable...)
		img.DisabledAnacondaModules = append(img.DisabledAnacondaModules, instCust.Modules.Disable...)
	}

	img.Platform = t.platform
	img.Workload = workload

	img.ExtraBasePackages = packageSets[installerPkgsKey]

	installerConfig, err := t.getDefaultInstallerConfig()
	if err != nil {
		return nil, err
	}

	if installerConfig != nil {
		img.AdditionalDracutModules = append(img.AdditionalDracutModules, installerConfig.AdditionalDracutModules...)
		img.AdditionalDrivers = append(img.AdditionalDrivers, installerConfig.AdditionalDrivers...)
	}

	d := t.arch.distro

	img.Product = d.product

	// We don't know the variant that goes into the OS pipeline that gets installed
	img.Variant = "Unknown"

	img.OSVersion = d.osVersion
	img.Release = fmt.Sprintf("%s %s", d.product, d.osVersion)

	img.Preview = common.VersionGreaterThanOrEqual(img.OSVersion, VERSION_BRANCHED)

	img.ISOLabel, err = t.ISOLabel()
	if err != nil {
		return nil, err
	}

	img.Filename = t.Filename()

	img.RootfsCompression = "lz4"
	if common.VersionGreaterThanOrEqual(img.OSVersion, VERSION_ROOTFS_SQUASHFS) {
		img.RootfsType = manifest.SquashfsRootfs
	}

	// Enable grub2 BIOS iso on x86_64 only
	if img.Platform.GetArch() == arch.ARCH_X86_64 {
		img.ISOBoot = manifest.Grub2ISOBoot
	}

	return img, nil
}

func iotCommitImage(workload workload.Workload,
	t *imageType,
	bp *blueprint.Blueprint,
	options distro.ImageOptions,
	packageSets map[string]rpmmd.PackageSet,
	containers []container.SourceSpec,
	rng *rand.Rand) (image.ImageKind, error) {

	parentCommit, commitRef := makeOSTreeParentCommit(options.OSTree, t.OSTreeRef())
	img := image.NewOSTreeArchive(commitRef)

	d := t.arch.distro

	img.Platform = t.platform

	var err error
	img.OSCustomizations, err = distro.OsCustomizations(&t.distroConfig, packageSets[osPkgsKey], distro.ImageOptions{}, containers, bp.Customizations)
	if err != nil {
		return nil, err
	}

	// see https://github.com/ostreedev/ostree/issues/2840
	img.OSCustomizations.Presets = []osbuild.Preset{
		{
			Name:  "ignition-firstboot-complete.service",
			State: osbuild.StateEnable,
		},
		{
			Name:  "coreos-ignition-write-issues.service",
			State: osbuild.StateEnable,
		},
		{
			Name:  "fdo-client-linuxapp.service",
			State: osbuild.StateEnable,
		},
	}

	img.Environment = t.environment
	img.Workload = workload
	img.OSTreeParent = parentCommit
	img.OSVersion = d.osVersion
	img.Filename = t.Filename()
	img.InstallWeakDeps = false

	return img, nil
}

func bootableContainerImage(workload workload.Workload,
	t *imageType,
	bp *blueprint.Blueprint,
	options distro.ImageOptions,
	packageSets map[string]rpmmd.PackageSet,
	containers []container.SourceSpec,
	rng *rand.Rand) (image.ImageKind, error) {

	parentCommit, commitRef := makeOSTreeParentCommit(options.OSTree, t.OSTreeRef())
	img := image.NewOSTreeArchive(commitRef)

	d := t.arch.distro

	img.Platform = t.platform

	var err error
	img.OSCustomizations, err = distro.OsCustomizations(&t.distroConfig, packageSets[osPkgsKey], distro.ImageOptions{}, containers, bp.Customizations)
	if err != nil {
		return nil, err
	}

	img.Environment = t.environment
	img.Workload = workload
	img.OSTreeParent = parentCommit
	img.OSVersion = d.osVersion
	img.Filename = t.Filename()
	img.InstallWeakDeps = false
	img.BootContainer = true
	img.BootcConfig = &bootc.Config{
		Filename:           "20-fedora.toml",
		RootFilesystemType: "ext4",
	}

	return img, nil
}

func iotContainerImage(workload workload.Workload,
	t *imageType,
	bp *blueprint.Blueprint,
	options distro.ImageOptions,
	packageSets map[string]rpmmd.PackageSet,
	containers []container.SourceSpec,
	rng *rand.Rand) (image.ImageKind, error) {

	parentCommit, commitRef := makeOSTreeParentCommit(options.OSTree, t.OSTreeRef())
	img := image.NewOSTreeContainer(commitRef)
	d := t.arch.distro
	img.Platform = t.platform

	var err error
	img.OSCustomizations, err = distro.OsCustomizations(&t.distroConfig, packageSets[osPkgsKey], distro.ImageOptions{}, containers, bp.Customizations)
	if err != nil {
		return nil, err
	}

	// see https://github.com/ostreedev/ostree/issues/2840
	img.OSCustomizations.Presets = []osbuild.Preset{
		{
			Name:  "ignition-firstboot-complete.service",
			State: osbuild.StateEnable,
		},
		{
			Name:  "coreos-ignition-write-issues.service",
			State: osbuild.StateEnable,
		},
		{
			Name:  "fdo-client-linuxapp.service",
			State: osbuild.StateEnable,
		},
	}

	img.ContainerLanguage = img.OSCustomizations.Language
	img.Environment = t.environment
	img.Workload = workload
	img.OSTreeParent = parentCommit
	img.OSVersion = d.osVersion
	img.ExtraContainerPackages = packageSets[containerPkgsKey]
	img.Filename = t.Filename()

	return img, nil
}

func iotInstallerImage(workload workload.Workload,
	t *imageType,
	bp *blueprint.Blueprint,
	options distro.ImageOptions,
	packageSets map[string]rpmmd.PackageSet,
	containers []container.SourceSpec,
	rng *rand.Rand) (image.ImageKind, error) {

	d := t.arch.distro

	commit, err := makeOSTreePayloadCommit(options.OSTree, t.OSTreeRef())
	if err != nil {
		return nil, fmt.Errorf("%s: %s", t.Name(), err.Error())
	}

	img := image.NewAnacondaOSTreeInstaller(commit)

	customizations := bp.Customizations
	img.FIPS = customizations.GetFIPS()
	img.Platform = t.platform
	img.ExtraBasePackages = packageSets[installerPkgsKey]

	img.Kickstart, err = kickstart.New(customizations)
	if err != nil {
		return nil, err
	}
	img.Kickstart.OSTree = &kickstart.OSTree{
		OSName: "fedora-iot",
		Remote: "fedora-iot",
	}
	img.Kickstart.Path = osbuild.KickstartPathOSBuild
	img.Kickstart.Language, img.Kickstart.Keyboard = customizations.GetPrimaryLocale()
	// ignore ntp servers - we don't currently support setting these in the
	// kickstart though kickstart does support setting them
	img.Kickstart.Timezone, _ = customizations.GetTimezoneSettings()

	instCust, err := customizations.GetInstaller()
	if err != nil {
		return nil, err
	}
	if instCust != nil && instCust.Modules != nil {
		img.AdditionalAnacondaModules = append(img.AdditionalAnacondaModules, instCust.Modules.Enable...)
		img.DisabledAnacondaModules = append(img.DisabledAnacondaModules, instCust.Modules.Disable...)
	}

	img.AdditionalAnacondaModules = append(img.AdditionalAnacondaModules, []string{
		anaconda.ModuleTimezone,
		anaconda.ModuleLocalization,
		anaconda.ModuleUsers,
	}...)

	installerConfig, err := t.getDefaultInstallerConfig()
	if err != nil {
		return nil, err
	}

	if installerConfig != nil {
		img.AdditionalDracutModules = append(img.AdditionalDracutModules, installerConfig.AdditionalDracutModules...)
		img.AdditionalDrivers = append(img.AdditionalDrivers, installerConfig.AdditionalDrivers...)
	}

	img.Product = d.product
	img.Variant = "IoT"
	img.OSVersion = d.osVersion
	img.Release = fmt.Sprintf("%s %s", d.product, d.osVersion)
	img.Preview = common.VersionGreaterThanOrEqual(img.OSVersion, VERSION_BRANCHED)

	img.ISOLabel, err = t.ISOLabel()
	if err != nil {
		return nil, err
	}

	img.Filename = t.Filename()

	img.RootfsCompression = "lz4"
	if common.VersionGreaterThanOrEqual(img.OSVersion, VERSION_ROOTFS_SQUASHFS) {
		img.RootfsType = manifest.SquashfsRootfs
	}

	// Enable grub2 BIOS iso on x86_64 only
	if img.Platform.GetArch() == arch.ARCH_X86_64 {
		img.ISOBoot = manifest.Grub2ISOBoot
	}

	if locale := t.getDefaultImageConfig().Locale; locale != nil {
		img.Locale = *locale
	}

	return img, nil
}

func iotImage(workload workload.Workload,
	t *imageType,
	bp *blueprint.Blueprint,
	options distro.ImageOptions,
	packageSets map[string]rpmmd.PackageSet,
	containers []container.SourceSpec,
	rng *rand.Rand) (image.ImageKind, error) {

	commit, err := makeOSTreePayloadCommit(options.OSTree, t.OSTreeRef())
	if err != nil {
		return nil, fmt.Errorf("%s: %s", t.Name(), err.Error())
	}
	img := image.NewOSTreeDiskImageFromCommit(commit)

	customizations := bp.Customizations
	deploymentConfig, err := ostreeDeploymentCustomizations(t, customizations)
	if err != nil {
		return nil, err
	}
	img.OSTreeDeploymentCustomizations = deploymentConfig

	img.Platform = t.platform
	img.Workload = workload

	img.Remote = ostree.Remote{
		Name: "fedora-iot",
	}
	img.OSName = "fedora-iot"

	// TODO: move generation into LiveImage
	pt, err := t.getPartitionTable(customizations, options, rng)
	if err != nil {
		return nil, err
	}
	img.PartitionTable = pt

	img.Filename = t.Filename()
	img.Compression = t.compression

	return img, nil
}

func iotSimplifiedInstallerImage(workload workload.Workload,
	t *imageType,
	bp *blueprint.Blueprint,
	options distro.ImageOptions,
	packageSets map[string]rpmmd.PackageSet,
	containers []container.SourceSpec,
	rng *rand.Rand) (image.ImageKind, error) {

	commit, err := makeOSTreePayloadCommit(options.OSTree, t.OSTreeRef())
	if err != nil {
		return nil, fmt.Errorf("%s: %s", t.Name(), err.Error())
	}
	rawImg := image.NewOSTreeDiskImageFromCommit(commit)

	customizations := bp.Customizations
	deploymentConfig, err := ostreeDeploymentCustomizations(t, customizations)
	if err != nil {
		return nil, err
	}
	rawImg.OSTreeDeploymentCustomizations = deploymentConfig

	rawImg.Platform = t.platform
	rawImg.Workload = workload
	rawImg.Remote = ostree.Remote{
		Name: "fedora-iot",
	}
	rawImg.OSName = "fedora"

	// TODO: move generation into LiveImage
	pt, err := t.getPartitionTable(customizations, options, rng)
	if err != nil {
		return nil, err
	}
	rawImg.PartitionTable = pt

	rawImg.Filename = t.Filename()

	img := image.NewOSTreeSimplifiedInstaller(rawImg, customizations.InstallationDevice)
	img.ExtraBasePackages = packageSets[installerPkgsKey]
	// img.Workload = workload
	img.Platform = t.platform
	img.Filename = t.Filename()
	if bpFDO := customizations.GetFDO(); bpFDO != nil {
		img.FDO = fdo.FromBP(*bpFDO)
	}
	// ignition configs from blueprint
	if bpIgnition := customizations.GetIgnition(); bpIgnition != nil {
		if bpIgnition.Embedded != nil {
			var err error
			img.IgnitionEmbedded, err = ignition.EmbeddedOptionsFromBP(*bpIgnition.Embedded)
			if err != nil {
				return nil, err
			}
		}
	}

	installerConfig, err := t.getDefaultInstallerConfig()
	if err != nil {
		return nil, err
	}

	if installerConfig != nil {
		img.AdditionalDracutModules = append(img.AdditionalDracutModules, installerConfig.AdditionalDracutModules...)
		img.AdditionalDrivers = append(img.AdditionalDrivers, installerConfig.AdditionalDrivers...)
	}

	img.AdditionalDracutModules = append(img.AdditionalDracutModules, "dbus-broker")

	d := t.arch.distro
	img.Product = d.product
	img.Variant = "IoT"
	img.OSName = "fedora"
	img.OSVersion = d.osVersion

	img.ISOLabel, err = t.ISOLabel()
	if err != nil {
		return nil, err
	}

	return img, nil
}

// Create an ostree SourceSpec to define an ostree parent commit using the user
// options and the default ref for the image type.  Additionally returns the
// ref to be used for the new commit to be created.
func makeOSTreeParentCommit(options *ostree.ImageOptions, defaultRef string) (*ostree.SourceSpec, string) {
	commitRef := defaultRef
	if options == nil {
		// nothing to do
		return nil, commitRef
	}
	if options.ImageRef != "" {
		// user option overrides default commit ref
		commitRef = options.ImageRef
	}

	var parentCommit *ostree.SourceSpec
	if options.URL == "" {
		// no parent
		return nil, commitRef
	}

	// ostree URL specified: set source spec for parent commit
	parentRef := options.ParentRef
	if parentRef == "" {
		// parent ref not set: use image ref
		parentRef = commitRef

	}
	parentCommit = &ostree.SourceSpec{
		URL:  options.URL,
		Ref:  parentRef,
		RHSM: options.RHSM,
	}
	return parentCommit, commitRef
}

// Create an ostree SourceSpec to define an ostree payload using the user options and the default ref for the image type.
func makeOSTreePayloadCommit(options *ostree.ImageOptions, defaultRef string) (ostree.SourceSpec, error) {
	if options == nil || options.URL == "" {
		// this should be caught by checkOptions() in distro, but it's good
		// to guard against it here as well
		return ostree.SourceSpec{}, fmt.Errorf("ostree commit URL required")
	}

	commitRef := defaultRef
	if options.ImageRef != "" {
		// user option overrides default commit ref
		commitRef = options.ImageRef
	}

	return ostree.SourceSpec{
		URL:  options.URL,
		Ref:  commitRef,
		RHSM: options.RHSM,
	}, nil
}

// initialSetupKickstart returns the File configuration for a kickstart file
// that's required to enable initial-setup to run on first boot.
func initialSetupKickstart() *fsnode.File {
	file, err := fsnode.NewFile("/root/anaconda-ks.cfg", nil, "root", "root", []byte("# Run initial-setup on first boot\n# Created by osbuild\nfirstboot --reconfig\n"))
	if err != nil {
		panic(err)
	}
	return file
}
