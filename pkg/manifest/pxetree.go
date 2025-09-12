package manifest

import (
	"fmt"
	"strings"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/customizations/fsnode"
	"github.com/osbuild/images/pkg/osbuild"
)

type PXETree struct {
	Base
	RootfsCompression string
	RootfsType        ISORootfsType

	osPipeline *OS
	files      []*fsnode.File // grub template and README files
}

// NewPXETree creates a pipeline with a kernel, initrd, and compressed root filesystem
// suitable for use with PXE booting a system.
// Defaults to using xz compressed squashfs rootfs
func NewPXETree(buildPipeline Build, osPipeline *OS) *PXETree {
	p := &PXETree{
		Base:              NewBase("pxe-tree", buildPipeline),
		osPipeline:        osPipeline,
		RootfsCompression: "xz",
		RootfsType:        SquashfsRootfs,
	}
	buildPipeline.addDependent(p)
	return p
}

func (p *PXETree) getBuildPackages(Distro) []string {
	switch p.RootfsType {
	case ErofsRootfs:
		return []string{"erofs-utils"}
	default:
		return []string{"squashfs-tools"}
	}
}

// Create a directory tree containing the kernel, initrd, and compressed rootfs
func (p *PXETree) serialize() (osbuild.Pipeline, error) {
	pipeline, err := p.Base.serialize()
	if err != nil {
		return pipeline, err
	}

	inputName := "tree"
	copyStageOptions := &osbuild.CopyStageOptions{
		Paths: []osbuild.CopyStagePath{
			{
				From: fmt.Sprintf("input://%s/boot/vmlinuz-%s", inputName, p.osPipeline.kernelVer),
				To:   "tree:///vmlinuz",
			},
			{
				From: fmt.Sprintf("input://%s/boot/initramfs-%s.img", inputName, p.osPipeline.kernelVer),
				To:   "tree:///initrd.img",
			},
			{
				From: fmt.Sprintf("input://%s/boot/efi/EFI", inputName),
				To:   "tree:///EFI",
			},
		},
	}
	copyStageInputs := osbuild.NewPipelineTreeInputs(inputName, p.osPipeline.Name())
	copyStage := osbuild.NewCopyStageSimple(copyStageOptions, copyStageInputs)
	pipeline.AddStage(copyStage)

	// Compress the os tree
	if p.RootfsType == ErofsRootfs {
		erofsOptions := osbuild.ErofsStageOptions{
			Filename: "rootfs.img",
		}

		var compression osbuild.ErofsCompression
		if p.RootfsCompression != "" {
			compression.Method = p.RootfsCompression
		} else {
			// default to zstd if not specified
			compression.Method = "zstd"
		}
		compression.Level = common.ToPtr(8)
		erofsOptions.Compression = &compression
		erofsOptions.ExtendedOptions = []string{"all-fragments", "dedupe"}
		erofsOptions.ClusterSize = common.ToPtr(131072)

		// TODO this is shared with the ISO, should it be?
		// Clean up the root filesystem's /boot to save space
		erofsOptions.ExcludePaths = installerBootExcludePaths
		pipeline.AddStage(osbuild.NewErofsStage(&erofsOptions, p.osPipeline.Name()))
	} else {
		var squashfsOptions osbuild.SquashfsStageOptions

		squashfsOptions.Filename = "rootfs.img"
		squashfsOptions.Compression.Method = "xz"

		if squashfsOptions.Compression.Method == "xz" {
			squashfsOptions.Compression.Options = &osbuild.FSCompressionOptions{
				BCJ: osbuild.BCJOption(p.osPipeline.platform.GetArch().String()),
			}
		}

		// TODO this is shared with the ISO, should it be?
		// Clean up the root filesystem's /boot to save space
		squashfsOptions.ExcludePaths = installerBootExcludePaths
		pipeline.AddStage(osbuild.NewSquashfsStage(&squashfsOptions, p.osPipeline.Name()))
	}

	// Make an example grub.cfg
	pipeline.AddStages(p.makeGrubConfig()...)

	// Make a README file
	pipeline.AddStages(p.makeREADME()...)

	// Make sure all the files are readable
	options := osbuild.ChmodStageOptions{
		Items: map[string]osbuild.ChmodStagePathOptions{
			"/EFI": {
				Mode:      "ugo+Xr",
				Recursive: true,
			},
			"/vmlinuz": {
				Mode: "0755",
			},
			"/initrd.img": {
				Mode: "0644",
			},
			"/rootfs.img": {
				Mode: "0644",
			},
			"/grub.cfg": {
				Mode: "0644",
			},
			"/README": {
				Mode: "0644",
			},
		},
	}
	pipeline.AddStage(osbuild.NewChmodStage(&options))
	return pipeline, nil
}

// dracutStageOptions returns the basic dracut setup for booting from a compressed
// root filesystem using root=live:... on the kernel cmdline.
func (p *PXETree) DracutConfStageOptions() *osbuild.DracutConfStageOptions {
	return &osbuild.DracutConfStageOptions{
		Filename: "40-pxe.conf",
		Config: osbuild.DracutConfigFile{
			EarlyMicrocode: common.ToPtr(false),
			AddModules:     []string{"qemu", "qemu-net", "livenet", "dmsquash-live"},
			Compress:       "xz",
		},
	}
}

// TODO - find a better way to specify the template
var grubTemplate = `set timeout=60
menuentry 'http-rootfs' {
    linux /vmlinuz root=live:http://HTTP-SERVER/rootfs.img rd.live.image @CMDLINE@
    initrd /initrd.img
}
menuentry 'combined-rootfs' {
    linux /vmlinuz root=live:/rootfs.img rd.live.image @CMDLINE@
    initrd /combined.img
}
`

// makeGrubConfig returns stages that creates an example grub config file
// It adds any kernel arguments from the blueprint to the cmdline in the template
func (p *PXETree) makeGrubConfig() []*osbuild.Stage {
	template := strings.ReplaceAll(grubTemplate, "@CMDLINE@", strings.Join(p.osPipeline.OSCustomizations.KernelOptionsAppend, " "))
	f, err := fsnode.NewFile("/grub.cfg", nil, nil, nil, []byte(template))
	if err != nil {
		panic(err)
	}
	p.files = append(p.files, f)
	return osbuild.GenFileNodesStages([]*fsnode.File{f})
}

// TODO - find a better way to specify the README
var readme = `
# About this archive

This archive contains files suitable for use with PXE booting or UEFI HTTP booting.
It includes this following:

* EFI/ directory tree of shim and Grub2 bootloader files
* vmlinuz - kernel
* inird.img - initial ramdisk
* rootfs.img - compressed root filesystem
* grub.cfg - a grub2 template

Make sure that the system has enough RAM to hold the kernel, initrd, and roofs
in memory. This size will depend on how large your rootfs.img is and what kinds
of workloads you are running. 2GiB is usually enough for a small image. If there isn't
enough RAM to boot you will likely see a kernel panic.

# PXE booting with a HTTP server

The grub.cfg file is a template that will need to be modified for your speific situation.
It expects the kernel, initrd, and rootfs to be placed at the / of the tftp server's
directory tree.

The first entry uses http to serve the rootfs.img you will need to replace
'HTTP-SERVER' with the url of a http server. For experimentation you can launch
a simple server using python like this:

    python3 -m http.server 8000

Run that from the directory with the rootfs.img file.

# PXE booting with combined image

The second entry is for use with a combined initrd and rootfs image. This saves you the
step of setting up a HTTP server but requires that you assemble the combined image from
the initrd and the root filesystem.

## Create a combined image

    echo rootfs.img | cpio -c --quiet -L -o > rootfs.cpio
    cat initrd.img rootfs.cpio > combined.img

`

// makeREADME returns a stage that creates a README file
func (p *PXETree) makeREADME() []*osbuild.Stage {
	f, err := fsnode.NewFile("/README", nil, nil, nil, []byte(readme))
	if err != nil {
		panic(err)
	}
	p.files = append(p.files, f)
	return osbuild.GenFileNodesStages([]*fsnode.File{f})
}

func (p *PXETree) getInline() []string {
	inlineData := []string{}

	// inline data for custom files
	for _, file := range p.files {
		inlineData = append(inlineData, string(file.Data()))
	}

	return inlineData
}
