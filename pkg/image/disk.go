package image

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/internal/environment"
	"github.com/osbuild/images/internal/workload"
	"github.com/osbuild/images/pkg/artifact"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/platform"
	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/osbuild/images/pkg/runner"
)

type DiskImage struct {
	Base
	Platform         platform.Platform
	PartitionTable   *disk.PartitionTable
	OSCustomizations manifest.OSCustomizations
	Environment      environment.Environment
	Workload         workload.Workload
	Filename         string
	Compression      string
	ForceSize        *bool
	PartTool         osbuild.PartTool

	NoBLS     bool
	OSProduct string
	OSVersion string
	OSNick    string

	// InstallWeakDeps enables installation of weak dependencies for packages
	// that are statically defined for the payload pipeline of the image.
	InstallWeakDeps *bool
}

type imagePipelineOpts struct {
	QCOW2Compat string
	ForceSize   *bool
	Filename    string
}

func makeImagePipeline(imgType platform.ImageFormat, rawImagePipeline manifest.FilePipeline, buildPipeline manifest.Build, opts *imagePipelineOpts) manifest.FilePipeline {
	if opts == nil {
		opts = &imagePipelineOpts{}
	}

	var imagePipeline manifest.FilePipeline
	switch imgType {
	case platform.FORMAT_UNSET:
		// edge image do not set image types and assume unset is
		// raw
		// TODO: fix edge definitions
		// a logger would be nice here to tell the world about this
		fallthrough
	case platform.FORMAT_RAW:
		imagePipeline = rawImagePipeline
	case platform.FORMAT_QCOW2:
		qcow2Pipeline := manifest.NewQCOW2(buildPipeline, rawImagePipeline)
		qcow2Pipeline.Compat = opts.QCOW2Compat
		imagePipeline = qcow2Pipeline
	case platform.FORMAT_VHD:
		vpcPipeline := manifest.NewVPC(buildPipeline, rawImagePipeline)
		vpcPipeline.ForceSize = opts.ForceSize
		imagePipeline = vpcPipeline
	case platform.FORMAT_VMDK:
		imagePipeline = manifest.NewVMDK(buildPipeline, rawImagePipeline)
	case platform.FORMAT_OVA:
		vmdkPipeline := manifest.NewVMDK(buildPipeline, rawImagePipeline)
		ovfPipeline := manifest.NewOVF(buildPipeline, vmdkPipeline)
		tarPipeline := manifest.NewTar(buildPipeline, ovfPipeline, "archive")
		tarPipeline.Format = osbuild.TarArchiveFormatUstar
		tarPipeline.SetFilename(opts.Filename)
		extLess := strings.TrimSuffix(opts.Filename, filepath.Ext(opts.Filename))
		// The .ovf descriptor needs to be the first file in the archive
		tarPipeline.Paths = []string{
			fmt.Sprintf("%s.ovf", extLess),
			fmt.Sprintf("%s.mf", extLess),
			fmt.Sprintf("%s.vmdk", extLess),
		}
		imagePipeline = tarPipeline
	case platform.FORMAT_GCE:
		// NOTE(akoutsou): temporary workaround; filename required for GCP
		// TODO: define internal raw filename on image type
		rawImagePipeline.SetFilename("disk.raw")
		tarPipeline := manifest.NewTar(buildPipeline, rawImagePipeline, "archive")
		tarPipeline.Format = osbuild.TarArchiveFormatOldgnu
		tarPipeline.RootNode = osbuild.TarRootNodeOmit
		// these are required to successfully import the image to GCP
		tarPipeline.ACLs = common.ToPtr(false)
		tarPipeline.SELinux = common.ToPtr(false)
		tarPipeline.Xattrs = common.ToPtr(false)
		tarPipeline.SetFilename(opts.Filename) // filename extension will determine compression
		imagePipeline = tarPipeline
	default:
		panic(fmt.Sprintf("invalid image format %v for image kind", imgType))
	}

	return imagePipeline
}

func makeCompressionPipeline(compression, filename string, imagePipeline manifest.FilePipeline, buildPipeline manifest.Build) manifest.FilePipeline {
	compressionPipeline := imagePipeline

	switch compression {
	case "xz":
		compressionPipeline = manifest.NewXZ(buildPipeline, imagePipeline)
	case "":
		// nothing to do
	default:
		// panic on unknown strings
		panic(fmt.Sprintf("unsupported compression type %q", compression))
	}
	compressionPipeline.SetFilename(filename)
	return compressionPipeline
}

func NewDiskImage() *DiskImage {
	return &DiskImage{
		Base:     NewBase("disk"),
		PartTool: osbuild.PTSfdisk,
	}
}

func (img *DiskImage) InstantiateManifest(m *manifest.Manifest,
	repos []rpmmd.RepoConfig,
	runner runner.Runner,
	rng *rand.Rand) (*artifact.Artifact, error) {
	buildPipeline := manifest.NewBuild(m, runner, repos, nil)
	buildPipeline.Checkpoint()

	osPipeline := manifest.NewOS(buildPipeline, img.Platform, repos)
	osPipeline.PartitionTable = img.PartitionTable
	osPipeline.OSCustomizations = img.OSCustomizations
	osPipeline.Environment = img.Environment
	osPipeline.Workload = img.Workload
	osPipeline.NoBLS = img.NoBLS
	osPipeline.OSProduct = img.OSProduct
	osPipeline.OSVersion = img.OSVersion
	osPipeline.OSNick = img.OSNick
	if img.InstallWeakDeps != nil {
		osPipeline.InstallWeakDeps = *img.InstallWeakDeps
	}

	rawImagePipeline := manifest.NewRawImage(buildPipeline, osPipeline)
	rawImagePipeline.PartTool = img.PartTool

	opts := &imagePipelineOpts{
		QCOW2Compat: img.Platform.GetQCOW2Compat(),
		ForceSize:   img.ForceSize,
		Filename:    img.Filename,
	}
	imagePipeline := makeImagePipeline(img.Platform.GetImageFormat(), rawImagePipeline, buildPipeline, opts)
	compressionPipeline := makeCompressionPipeline(img.Compression, img.Filename, imagePipeline, buildPipeline)
	return compressionPipeline.Export(), nil
}
