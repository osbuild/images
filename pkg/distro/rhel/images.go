package rhel

import (
	"fmt"
	"math/rand"

	"github.com/osbuild/images/internal/workload"
	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/customizations/anaconda"
	"github.com/osbuild/images/pkg/customizations/fdo"
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

func ostreeDeploymentCustomizations(
	t *distro.ImageTypeConfig,
	c *blueprint.Customizations) (manifest.OSTreeDeploymentCustomizations, error) {

	if !t.RpmOstree || !t.Bootable {
		return manifest.OSTreeDeploymentCustomizations{}, fmt.Errorf("ostree deployment customizations are only supported for bootable rpm-ostree images")
	}

	imageConfig := t.DefaultImageConfig
	deploymentConf := manifest.OSTreeDeploymentCustomizations{}

	var kernelOptions []string
	if len(t.KernelOptions) > 0 {
		kernelOptions = append(kernelOptions, t.KernelOptions...)
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

	if imageConfig.MountUnits != nil {
		deploymentConf.MountUnits = *imageConfig.MountUnits
	}

	return deploymentConf, nil
}

func DiskImage(workload workload.Workload,
	t *ImageType,
	customizations *blueprint.Customizations,
	options distro.ImageOptions,
	packageSets map[string]rpmmd.PackageSet,
	containers []container.SourceSpec,
	rng *rand.Rand) (image.ImageKind, error) {

	img := image.NewDiskImage()
	img.Platform = t.platform

	var err error
	img.OSCustomizations, err = distro.OsCustomizations(&t.DistroConfig, packageSets[OSPkgsKey], options, containers, customizations)
	if err != nil {
		return nil, err
	}

	img.Environment = t.Environment
	img.Workload = workload
	img.Compression = t.Compression
	// TODO: move generation into LiveImage
	pt, err := t.GetPartitionTable(customizations, options, rng)
	if err != nil {
		return nil, err
	}
	img.PartitionTable = pt

	img.Filename = t.Filename()

	img.VPCForceSize = t.DiskImageVPCForceSize

	if img.OSCustomizations.NoBLS {
		img.OSProduct = t.Arch().Distro().Product()
		img.OSVersion = t.Arch().Distro().OsVersion()
		img.OSNick = t.Arch().Distro().Codename()
	}

	if t.DiskImagePartTool != nil {
		img.PartTool = *t.DiskImagePartTool
	}

	return img, nil
}

func EdgeCommitImage(workload workload.Workload,
	t *ImageType,
	customizations *blueprint.Customizations,
	options distro.ImageOptions,
	packageSets map[string]rpmmd.PackageSet,
	containers []container.SourceSpec,
	rng *rand.Rand) (image.ImageKind, error) {

	parentCommit, commitRef := makeOSTreeParentCommit(options.OSTree, t.OSTreeRef())
	img := image.NewOSTreeArchive(commitRef)

	img.Platform = t.platform

	var err error
	img.OSCustomizations, err = distro.OsCustomizations(&t.DistroConfig, packageSets[OSPkgsKey], options, containers, customizations)
	if err != nil {
		return nil, err
	}

	img.Environment = t.Environment
	img.Workload = workload
	img.OSTreeParent = parentCommit
	img.OSVersion = t.Arch().Distro().OsVersion()
	img.Filename = t.Filename()

	return img, nil
}

func EdgeContainerImage(workload workload.Workload,
	t *ImageType,
	customizations *blueprint.Customizations,
	options distro.ImageOptions,
	packageSets map[string]rpmmd.PackageSet,
	containers []container.SourceSpec,
	rng *rand.Rand) (image.ImageKind, error) {

	parentCommit, commitRef := makeOSTreeParentCommit(options.OSTree, t.OSTreeRef())
	img := image.NewOSTreeContainer(commitRef)

	img.Platform = t.platform

	var err error
	img.OSCustomizations, err = distro.OsCustomizations(&t.DistroConfig, packageSets[OSPkgsKey], options, containers, customizations)
	if err != nil {
		return nil, err
	}

	img.ContainerLanguage = img.OSCustomizations.Language
	img.Environment = t.Environment
	img.Workload = workload
	img.OSTreeParent = parentCommit
	img.OSVersion = t.Arch().Distro().OsVersion()
	img.ExtraContainerPackages = packageSets[ContainerPkgsKey]
	img.Filename = t.Filename()

	return img, nil
}

func EdgeInstallerImage(workload workload.Workload,
	t *ImageType,
	customizations *blueprint.Customizations,
	options distro.ImageOptions,
	packageSets map[string]rpmmd.PackageSet,
	containers []container.SourceSpec,
	rng *rand.Rand) (image.ImageKind, error) {

	commit, err := makeOSTreePayloadCommit(options.OSTree, t.OSTreeRef())
	if err != nil {
		return nil, fmt.Errorf("%s: %s", t.Name(), err.Error())
	}

	img := image.NewAnacondaOSTreeInstaller(commit)

	img.Platform = t.platform
	img.ExtraBasePackages = packageSets[InstallerPkgsKey]
	img.Subscription = options.Subscription

	if t.Arch().Distro().Releasever() == "8" {
		// NOTE: RHEL 8 only supports the older Anaconda configs
		img.UseLegacyAnacondaConfig = true
	}

	img.Kickstart, err = kickstart.New(customizations)
	if err != nil {
		return nil, err
	}
	img.Kickstart.OSTree = &kickstart.OSTree{
		OSName: "rhel-edge",
	}
	img.Kickstart.Path = osbuild.KickstartPathOSBuild
	img.Kickstart.Language, img.Kickstart.Keyboard = customizations.GetPrimaryLocale()
	// ignore ntp servers - we don't currently support setting these in the
	// kickstart though kickstart does support setting them
	img.Kickstart.Timezone, _ = customizations.GetTimezoneSettings()

	img.RootfsCompression = "xz"
	if t.Arch().Distro().Releasever() == "10" {
		img.RootfsType = manifest.SquashfsRootfs
	}

	// Enable BIOS iso on x86_64 only
	// Use grub2 on RHEL10, otherwise use syslinux
	// NOTE: Will need to be updated for RHEL11 and later
	if img.Platform.GetArch() == arch.ARCH_X86_64 {
		if t.Arch().Distro().Releasever() == "10" {
			img.ISOBoot = manifest.Grub2ISOBoot
		} else {
			img.ISOBoot = manifest.SyslinuxISOBoot
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

	instCust, err := customizations.GetInstaller()
	if err != nil {
		return nil, err
	}
	if instCust != nil && instCust.Modules != nil {
		img.AdditionalAnacondaModules = append(img.AdditionalAnacondaModules, instCust.Modules.Enable...)
		img.DisabledAnacondaModules = append(img.DisabledAnacondaModules, instCust.Modules.Disable...)
	}

	if len(img.Kickstart.Users)+len(img.Kickstart.Groups) > 0 {
		// only enable the users module if needed
		img.AdditionalAnacondaModules = append(img.AdditionalAnacondaModules, anaconda.ModuleUsers)
	}

	img.ISOLabel, err = t.ISOLabel()
	if err != nil {
		return nil, err
	}

	img.Product = t.Arch().Distro().Product()
	img.Variant = "edge"
	img.OSVersion = t.Arch().Distro().OsVersion()
	img.Release = fmt.Sprintf("%s %s", t.Arch().Distro().Product(), t.Arch().Distro().OsVersion())
	img.FIPS = customizations.GetFIPS()

	img.Filename = t.Filename()

	if locale := t.getDefaultImageConfig().Locale; locale != nil {
		img.Locale = *locale
	}

	return img, nil
}

func EdgeRawImage(workload workload.Workload,
	t *ImageType,
	customizations *blueprint.Customizations,
	options distro.ImageOptions,
	packageSets map[string]rpmmd.PackageSet,
	containers []container.SourceSpec,
	rng *rand.Rand) (image.ImageKind, error) {

	commit, err := makeOSTreePayloadCommit(options.OSTree, t.OSTreeRef())
	if err != nil {
		return nil, fmt.Errorf("%s: %s", t.Name(), err.Error())
	}
	img := image.NewOSTreeDiskImageFromCommit(commit)

	deploymentConfig, err := ostreeDeploymentCustomizations(&t.DistroConfig, customizations)
	if err != nil {
		return nil, err
	}
	img.OSTreeDeploymentCustomizations = deploymentConfig

	img.Platform = t.platform
	img.Workload = workload
	img.Remote = ostree.Remote{
		Name:       "rhel-edge",
		URL:        options.OSTree.URL,
		ContentURL: options.OSTree.ContentURL,
	}
	img.OSName = "rhel-edge"

	// TODO: move generation into LiveImage
	pt, err := t.GetPartitionTable(customizations, options, rng)
	if err != nil {
		return nil, err
	}
	img.PartitionTable = pt

	img.Filename = t.Filename()
	img.Compression = t.Compression

	return img, nil
}

func EdgeSimplifiedInstallerImage(workload workload.Workload,
	t *ImageType,
	customizations *blueprint.Customizations,
	options distro.ImageOptions,
	packageSets map[string]rpmmd.PackageSet,
	containers []container.SourceSpec,
	rng *rand.Rand) (image.ImageKind, error) {

	commit, err := makeOSTreePayloadCommit(options.OSTree, t.OSTreeRef())
	if err != nil {
		return nil, fmt.Errorf("%s: %s", t.Name(), err.Error())
	}
	rawImg := image.NewOSTreeDiskImageFromCommit(commit)

	deploymentConfig, err := ostreeDeploymentCustomizations(&t.DistroConfig, customizations)
	if err != nil {
		return nil, err
	}
	rawImg.OSTreeDeploymentCustomizations = deploymentConfig

	rawImg.Platform = t.platform
	rawImg.Workload = workload
	rawImg.Remote = ostree.Remote{
		Name:       "rhel-edge",
		URL:        options.OSTree.URL,
		ContentURL: options.OSTree.ContentURL,
	}
	rawImg.OSName = "rhel-edge"

	// TODO: move generation into LiveImage
	pt, err := t.GetPartitionTable(customizations, options, rng)
	if err != nil {
		return nil, err
	}
	rawImg.PartitionTable = pt

	rawImg.Filename = t.Filename()

	img := image.NewOSTreeSimplifiedInstaller(rawImg, customizations.InstallationDevice)
	img.ExtraBasePackages = packageSets[InstallerPkgsKey]
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

	img.ISOLabel, err = t.ISOLabel()
	if err != nil {
		return nil, err
	}

	d := t.arch.distro
	img.Product = d.product
	img.Variant = "edge"
	img.OSName = "rhel-edge"
	img.OSVersion = d.osVersion

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

func ImageInstallerImage(workload workload.Workload,
	t *ImageType,
	customizations *blueprint.Customizations,
	options distro.ImageOptions,
	packageSets map[string]rpmmd.PackageSet,
	containers []container.SourceSpec,
	rng *rand.Rand) (image.ImageKind, error) {

	img := image.NewAnacondaTarInstaller()

	img.Platform = t.platform
	img.Workload = workload

	var err error
	img.OSCustomizations, err = distro.OsCustomizations(&t.DistroConfig, packageSets[OSPkgsKey], options, containers, customizations)
	if err != nil {
		return nil, err
	}

	img.ExtraBasePackages = packageSets[InstallerPkgsKey]

	if t.Arch().Distro().Releasever() == "8" {
		// NOTE: RHEL 8 only supports the older Anaconda configs
		img.UseLegacyAnacondaConfig = true
	}

	img.Kickstart, err = kickstart.New(customizations)
	if err != nil {
		return nil, err
	}
	img.Kickstart.Language = &img.OSCustomizations.Language
	img.Kickstart.Keyboard = img.OSCustomizations.Keyboard
	img.Kickstart.Timezone = &img.OSCustomizations.Timezone

	installerConfig, err := t.getDefaultInstallerConfig()
	if err != nil {
		return nil, err
	}

	if installerConfig != nil {
		img.AdditionalDracutModules = append(img.AdditionalDracutModules, installerConfig.AdditionalDracutModules...)
		img.AdditionalDrivers = append(img.AdditionalDrivers, installerConfig.AdditionalDrivers...)
	}

	instCust, err := customizations.GetInstaller()
	if err != nil {
		return nil, err
	}
	if instCust != nil && instCust.Modules != nil {
		img.AdditionalAnacondaModules = append(img.AdditionalAnacondaModules, instCust.Modules.Enable...)
		img.DisabledAnacondaModules = append(img.DisabledAnacondaModules, instCust.Modules.Disable...)
	}
	img.AdditionalAnacondaModules = append(img.AdditionalAnacondaModules, anaconda.ModuleUsers)

	img.RootfsCompression = "xz"
	if t.Arch().Distro().Releasever() == "10" {
		img.RootfsType = manifest.SquashfsRootfs
	}

	// Enable BIOS iso on x86_64 only
	// Use grub2 on RHEL10, otherwise use syslinux
	// NOTE: Will need to be updated for RHEL11 and later
	if img.Platform.GetArch() == arch.ARCH_X86_64 {
		if t.Arch().Distro().Releasever() == "10" {
			img.ISOBoot = manifest.Grub2ISOBoot
		} else {
			img.ISOBoot = manifest.SyslinuxISOBoot
		}
	}

	// put the kickstart file in the root of the iso
	img.ISORootKickstart = true

	img.ISOLabel, err = t.ISOLabel()
	if err != nil {
		return nil, err
	}

	d := t.arch.distro
	img.Product = d.product
	img.OSVersion = d.osVersion
	img.Release = fmt.Sprintf("%s %s", d.product, d.osVersion)

	img.Filename = t.Filename()

	return img, nil
}

func TarImage(workload workload.Workload,
	t *ImageType,
	customizations *blueprint.Customizations,
	options distro.ImageOptions,
	packageSets map[string]rpmmd.PackageSet,
	containers []container.SourceSpec,
	rng *rand.Rand) (image.ImageKind, error) {

	img := image.NewArchive()
	img.Platform = t.platform

	var err error
	img.OSCustomizations, err = distro.OsCustomizations(&t.DistroConfig, packageSets[OSPkgsKey], options, containers, customizations)
	if err != nil {
		return nil, err
	}

	img.Environment = t.Environment
	img.Workload = workload

	img.Filename = t.Filename()

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
