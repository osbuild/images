package image

import (
	"fmt"
	"math/rand"

	"github.com/osbuild/images/internal/environment"
	"github.com/osbuild/images/internal/workload"
	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/artifact"
	"github.com/osbuild/images/pkg/customizations/anaconda"
	"github.com/osbuild/images/pkg/datasizes"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/platform"
	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/osbuild/images/pkg/runner"
)

type AnacondaNetInstaller struct {
	Base
	Platform         platform.Platform
	OSCustomizations manifest.OSCustomizations
	Environment      environment.Environment
	Workload         workload.Workload

	ExtraBasePackages rpmmd.PackageSet

	RootfsCompression string
	RootfsType        manifest.RootfsType
	ISOBoot           manifest.ISOBootType

	ISOLabel  string
	Product   string
	Variant   string
	OSVersion string
	Release   string
	Preview   bool

	Filename string

	AdditionalKernelOpts    []string
	EnabledAnacondaModules  []string
	DisabledAnacondaModules []string

	AdditionalDracutModules []string
	AdditionalDrivers       []string

	// Uses the old, deprecated, Anaconda config option "kickstart-modules".
	// Only for RHEL 8.
	UseLegacyAnacondaConfig bool
}

func NewAnacondaNetInstaller() *AnacondaNetInstaller {
	return &AnacondaNetInstaller{
		Base: NewBase("netinst"),
	}
}

func (img *AnacondaNetInstaller) InstantiateManifest(m *manifest.Manifest,
	repos []rpmmd.RepoConfig,
	runner runner.Runner,
	rng *rand.Rand) (*artifact.Artifact, error) {
	buildPipeline := addBuildBootstrapPipelines(m, runner, repos, nil)
	buildPipeline.Checkpoint()

	installerType := manifest.AnacondaInstallerTypeNetinst
	anacondaPipeline := manifest.NewAnacondaInstaller(
		installerType,
		buildPipeline,
		img.Platform,
		repos,
		"kernel",
		img.Product,
		img.OSVersion,
		img.Preview,
	)

	anacondaPipeline.ExtraPackages = img.ExtraBasePackages.Include
	anacondaPipeline.ExcludePackages = img.ExtraBasePackages.Exclude
	anacondaPipeline.ExtraRepos = img.ExtraBasePackages.Repositories
	anacondaPipeline.Variant = img.Variant
	anacondaPipeline.Biosdevname = (img.Platform.GetArch() == arch.ARCH_X86_64)

	anacondaPipeline.UseLegacyAnacondaConfig = img.UseLegacyAnacondaConfig
	anacondaPipeline.EnabledAnacondaModules = img.EnabledAnacondaModules
	if img.OSCustomizations.FIPS {
		anacondaPipeline.EnabledAnacondaModules = append(
			anacondaPipeline.EnabledAnacondaModules,
			anaconda.ModuleSecurity,
		)
	}
	anacondaPipeline.DisabledAnacondaModules = img.DisabledAnacondaModules
	anacondaPipeline.AdditionalDracutModules = img.AdditionalDracutModules
	anacondaPipeline.AdditionalDrivers = img.AdditionalDrivers
	anacondaPipeline.Locale = img.OSCustomizations.Language

	anacondaPipeline.Checkpoint()

	var rootfsImagePipeline *manifest.ISORootfsImg
	switch img.RootfsType {
	case manifest.SquashfsExt4Rootfs:
		rootfsImagePipeline = manifest.NewISORootfsImg(buildPipeline, anacondaPipeline)
		rootfsImagePipeline.Size = 5 * datasizes.GibiByte
	default:
	}

	bootTreePipeline := manifest.NewEFIBootTree(buildPipeline, img.Product, img.OSVersion)
	bootTreePipeline.Platform = img.Platform
	bootTreePipeline.UEFIVendor = img.Platform.GetUEFIVendor()
	bootTreePipeline.ISOLabel = img.ISOLabel

	kernelOpts := []string{fmt.Sprintf("inst.stage2=hd:LABEL=%s", img.ISOLabel)}
	if img.OSCustomizations.FIPS {
		kernelOpts = append(kernelOpts, "fips=1")
	}
	kernelOpts = append(kernelOpts, img.AdditionalKernelOpts...)
	bootTreePipeline.KernelOpts = kernelOpts

	// Payload type uses an OS Pipeline for the payload
	var osPipeline *manifest.OS
	if anacondaPipeline.Type == manifest.AnacondaInstallerTypePayload {
		osPipeline = manifest.NewOS(buildPipeline, img.Platform, repos)
		osPipeline.OSCustomizations = img.OSCustomizations
		osPipeline.Environment = img.Environment
		osPipeline.Workload = img.Workload
	}

	isoTreePipeline := manifest.NewAnacondaInstallerISOTree(buildPipeline, anacondaPipeline, rootfsImagePipeline, bootTreePipeline)
	// TODO: the partition table is required - make it a ctor arg or set a default one in the pipeline
	isoTreePipeline.PartitionTable = efiBootPartitionTable(rng)
	isoTreePipeline.Release = img.Release

	isoTreePipeline.RootfsCompression = img.RootfsCompression
	isoTreePipeline.RootfsType = img.RootfsType

	isoTreePipeline.OSPipeline = osPipeline
	isoTreePipeline.KernelOpts = img.AdditionalKernelOpts
	if img.OSCustomizations.FIPS {
		isoTreePipeline.KernelOpts = append(isoTreePipeline.KernelOpts, "fips=1")
	}

	isoTreePipeline.ISOBoot = img.ISOBoot

	isoPipeline := manifest.NewISO(buildPipeline, isoTreePipeline, img.ISOLabel)
	isoPipeline.SetFilename(img.Filename)
	isoPipeline.ISOBoot = img.ISOBoot

	artifact := isoPipeline.Export()

	return artifact, nil
}
