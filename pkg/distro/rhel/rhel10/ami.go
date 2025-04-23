package rhel10

import (
	"github.com/osbuild/images/pkg/datasizes"
	"github.com/osbuild/images/pkg/distro/rhel"
)

// TODO: move these to the EC2 environment

func amiKernelOptions() []string {
	return []string{"console=tty0", "console=ttyS0,115200n8", "nvme_core.io_timeout=4294967295"}
}

func amiAarch64KernelOptions() []string {
	return append(amiKernelOptions(), "iommu.strict=0")
}

func amiSapKernelOptions() []string {
	return append(amiKernelOptions(), []string{"processor.max_cstate=1", "intel_idle.max_cstate=1"}...)
}

func mkAMIImgTypeX86_64(d *rhel.Distribution) *rhel.ImageType {
	it := rhel.NewImageType(
		"ami",
		"image.raw",
		"application/octet-stream",
		map[string]rhel.PackageSetFunc{
			rhel.OSPkgsKey: packageSetLoader,
		},
		rhel.DiskImage,
		[]string{"build"},
		[]string{"os", "image"},
		[]string{"image"},
	)

	it.KernelOptions = amiKernelOptions()
	it.Bootable = true
	it.DefaultSize = 10 * datasizes.GibiByte
	it.DefaultImageConfig = imageConfig(d, "x86_64", "ami")
	it.BasePartitionTables = defaultBasePartitionTables

	return it
}

func mkAMIImgTypeAarch64(rd *rhel.Distribution) *rhel.ImageType {
	it := rhel.NewImageType(
		"ami",
		"image.raw",
		"application/octet-stream",
		map[string]rhel.PackageSetFunc{
			rhel.OSPkgsKey: packageSetLoader,
		},
		rhel.DiskImage,
		[]string{"build"},
		[]string{"os", "image"},
		[]string{"image"},
	)

	it.KernelOptions = amiAarch64KernelOptions()
	it.Bootable = true
	it.DefaultSize = 10 * datasizes.GibiByte
	it.DefaultImageConfig = imageConfig(rd, "aarch64", "ami")
	it.BasePartitionTables = defaultBasePartitionTables

	return it
}

// RHEL internal-only x86_64 EC2 image type
func mkEc2ImgTypeX86_64(rd *rhel.Distribution) *rhel.ImageType {
	it := rhel.NewImageType(
		"ec2",
		"image.raw.xz",
		"application/xz",
		map[string]rhel.PackageSetFunc{
			rhel.OSPkgsKey: packageSetLoader,
		},
		rhel.DiskImage,
		[]string{"build"},
		[]string{"os", "image", "xz"},
		[]string{"xz"},
	)

	it.Compression = "xz"
	it.KernelOptions = amiKernelOptions()
	it.Bootable = true
	it.DefaultSize = 10 * datasizes.GibiByte
	it.DefaultImageConfig = imageConfig(rd, "x86_64", "ec2")
	it.BasePartitionTables = defaultBasePartitionTables

	return it
}

// RHEL internal-only aarch64 EC2 image type
func mkEC2ImgTypeAarch64(rd *rhel.Distribution) *rhel.ImageType {
	it := rhel.NewImageType(
		"ec2",
		"image.raw.xz",
		"application/xz",
		map[string]rhel.PackageSetFunc{
			rhel.OSPkgsKey: packageSetLoader,
		},
		rhel.DiskImage,
		[]string{"build"},
		[]string{"os", "image", "xz"},
		[]string{"xz"},
	)

	it.Compression = "xz"
	it.KernelOptions = amiAarch64KernelOptions()
	it.Bootable = true
	it.DefaultSize = 10 * datasizes.GibiByte
	it.DefaultImageConfig = imageConfig(rd, "aarch64", "ec2")
	it.BasePartitionTables = defaultBasePartitionTables

	return it
}

// RHEL internal-only x86_64 EC2 HA image type
func mkEc2HaImgTypeX86_64(rd *rhel.Distribution) *rhel.ImageType {
	it := rhel.NewImageType(
		"ec2-ha",
		"image.raw.xz",
		"application/xz",
		map[string]rhel.PackageSetFunc{
			rhel.OSPkgsKey: packageSetLoader,
		},
		rhel.DiskImage,
		[]string{"build"},
		[]string{"os", "image", "xz"},
		[]string{"xz"},
	)

	it.Compression = "xz"
	it.KernelOptions = amiKernelOptions()
	it.Bootable = true
	it.DefaultSize = 10 * datasizes.GibiByte
	it.DefaultImageConfig = imageConfig(rd, "x86_64", "ec2-ha")
	it.BasePartitionTables = defaultBasePartitionTables

	return it
}

func mkEC2SapImgTypeX86_64(rd *rhel.Distribution) *rhel.ImageType {
	it := rhel.NewImageType(
		"ec2-sap",
		"image.raw.xz",
		"application/xz",
		map[string]rhel.PackageSetFunc{
			rhel.OSPkgsKey: packageSetLoader,
		},
		rhel.DiskImage,
		[]string{"build"},
		[]string{"os", "image", "xz"},
		[]string{"xz"},
	)

	it.Compression = "xz"
	it.KernelOptions = amiSapKernelOptions()
	it.Bootable = true
	it.DefaultSize = 10 * datasizes.GibiByte
	// XXX: inherit in YAML
	it.DefaultImageConfig = sapImageConfig(rd.OsVersion()).InheritFrom(imageConfig(rd, "x86_64", "ec2-ha"))
	it.BasePartitionTables = defaultBasePartitionTables

	return it
}
