package image

import (
	"fmt"
	"math/rand"

	"github.com/osbuild/osbuild-composer/internal/artifact"
	"github.com/osbuild/osbuild-composer/internal/common"
	"github.com/osbuild/osbuild-composer/internal/disk"
	"github.com/osbuild/osbuild-composer/internal/environment"
	"github.com/osbuild/osbuild-composer/internal/manifest"
	"github.com/osbuild/osbuild-composer/internal/platform"
	"github.com/osbuild/osbuild-composer/internal/rpmmd"
	"github.com/osbuild/osbuild-composer/internal/runner"
	"github.com/osbuild/osbuild-composer/internal/workload"
)

type AnacondaLiveInstaller struct {
	Base
	Platform    platform.Platform
	Environment environment.Environment
	Workload    workload.Workload

	ExtraBasePackages rpmmd.PackageSet

	ISOLabelTempl string
	Product       string
	Variant       string
	OSName        string
	OSVersion     string
	Release       string

	Filename string

	AdditionalKernelOpts []string
}

func NewAnacondaLiveInstaller() *AnacondaLiveInstaller {
	return &AnacondaLiveInstaller{
		Base: NewBase("live-installer"),
	}
}

func (img *AnacondaLiveInstaller) InstantiateManifest(m *manifest.Manifest,
	repos []rpmmd.RepoConfig,
	runner runner.Runner,
	rng *rand.Rand) (*artifact.Artifact, error) {
	buildPipeline := manifest.NewBuild(m, runner, repos)
	buildPipeline.Checkpoint()

	livePipeline := manifest.NewAnacondaInstaller(m,
		manifest.AnacondaInstallerTypeLive,
		buildPipeline,
		img.Platform,
		repos,
		"kernel",
		img.Product,
		img.OSVersion)

	livePipeline.ExtraPackages = img.ExtraBasePackages.Include

	livePipeline.Variant = img.Variant
	livePipeline.Biosdevname = (img.Platform.GetArch() == platform.ARCH_X86_64)

	livePipeline.Checkpoint()

	rootfsPartitionTable := &disk.PartitionTable{
		Size: 20 * common.MebiByte,
		Partitions: []disk.Partition{
			{
				Start: 0,
				Size:  20 * common.MebiByte,
				Payload: &disk.Filesystem{
					Type:       "vfat",
					Mountpoint: "/",
					UUID:       disk.NewVolIDFromRand(rng),
				},
			},
		},
	}

	// TODO: replace isoLabelTmpl with more high-level properties
	isoLabel := fmt.Sprintf(img.ISOLabelTempl, img.Platform.GetArch())

	rootfsImagePipeline := manifest.NewISORootfsImg(m, buildPipeline, livePipeline)
	rootfsImagePipeline.Size = 8 * common.GibiByte

	bootTreePipeline := manifest.NewEFIBootTree(m, buildPipeline, img.Product, img.OSVersion)
	bootTreePipeline.Platform = img.Platform
	bootTreePipeline.UEFIVendor = img.Platform.GetUEFIVendor()
	bootTreePipeline.ISOLabel = isoLabel

	kernelOpts := []string{
		fmt.Sprintf("root=live:CDLABEL=%s", isoLabel),
		"rd.live.image",
		"quiet",
		"rhgb",
	}

	kernelOpts = append(kernelOpts, img.AdditionalKernelOpts...)

	bootTreePipeline.KernelOpts = kernelOpts

	// enable ISOLinux on x86_64 only
	isoLinuxEnabled := img.Platform.GetArch() == platform.ARCH_X86_64

	isoTreePipeline := manifest.NewAnacondaInstallerISOTree(m,
		buildPipeline,
		livePipeline,
		rootfsImagePipeline,
		bootTreePipeline,
		isoLabel)
	isoTreePipeline.PartitionTable = rootfsPartitionTable
	isoTreePipeline.Release = img.Release
	isoTreePipeline.OSName = img.OSName

	isoTreePipeline.KernelOpts = kernelOpts
	isoTreePipeline.ISOLinux = isoLinuxEnabled

	isoPipeline := manifest.NewISO(m, buildPipeline, isoTreePipeline, isoLabel)
	isoPipeline.Filename = img.Filename
	isoPipeline.ISOLinux = isoLinuxEnabled

	artifact := isoPipeline.Export()

	return artifact, nil
}
