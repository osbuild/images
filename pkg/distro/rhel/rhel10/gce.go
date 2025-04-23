package rhel10

import (
	"github.com/osbuild/images/pkg/datasizes"
	"github.com/osbuild/images/pkg/distro/rhel"
)

func gceKernelOptions() []string {
	return []string{"biosdevname=0", "scsi_mod.use_blk_mq=Y", "console=ttyS0,38400n8d"}
}

func mkGCEImageType(rd *rhel.Distribution) *rhel.ImageType {
	it := rhel.NewImageType(
		"gce",
		"image.tar.gz",
		"application/gzip",
		map[string]rhel.PackageSetFunc{
			rhel.OSPkgsKey: packageSetLoader,
		},
		rhel.DiskImage,
		[]string{"build"},
		[]string{"os", "image", "archive"},
		[]string{"archive"},
	)

	it.DefaultImageConfig = imageConfig(rd, "", "gce")
	it.KernelOptions = gceKernelOptions()
	it.DefaultSize = 20 * datasizes.GibiByte
	it.Bootable = true
	// TODO: the base partition table still contains the BIOS boot partition, but the image is UEFI-only
	it.BasePartitionTables = defaultBasePartitionTables

	return it
}
