package rhel8

import (
	"github.com/osbuild/images/pkg/datasizes"
	"github.com/osbuild/images/pkg/distro/rhel"
)

func mkAmiImgTypeX86_64(d *rhel.Distribution) *rhel.ImageType {
	it := rhel.NewImageType(
		"ami",
		"image.raw",
		"application/octet-stream",
		packageSetLoader,
		rhel.DiskImage,
		[]string{"build"},
		[]string{"os", "image"},
		[]string{"image"},
	)

	it.DefaultImageConfig = imageConfig(d, "x86_64", "ami")
	it.Bootable = true
	it.DefaultSize = 10 * datasizes.GibiByte
	it.BasePartitionTables = partitionTables

	return it
}

func mkEc2ImgTypeX86_64(rd *rhel.Distribution) *rhel.ImageType {
	it := rhel.NewImageType(
		"ec2",
		"image.raw.xz",
		"application/xz",
		packageSetLoader,
		rhel.DiskImage,
		[]string{"build"},
		[]string{"os", "image", "xz"},
		[]string{"xz"},
	)

	it.Compression = "xz"
	it.DefaultImageConfig = imageConfig(rd, "x86_64", "ec2")
	it.Bootable = true
	it.DefaultSize = 10 * datasizes.GibiByte
	it.BasePartitionTables = partitionTables

	return it
}

func mkEc2HaImgTypeX86_64(rd *rhel.Distribution) *rhel.ImageType {
	it := rhel.NewImageType(
		"ec2-ha",
		"image.raw.xz",
		"application/xz",
		packageSetLoader,
		rhel.DiskImage,
		[]string{"build"},
		[]string{"os", "image", "xz"},
		[]string{"xz"},
	)

	it.Compression = "xz"
	it.DefaultImageConfig = imageConfig(rd, "x86_64", "ec2-ha")
	it.Bootable = true
	it.DefaultSize = 10 * datasizes.GibiByte
	it.BasePartitionTables = partitionTables

	return it
}

func mkAmiImgTypeAarch64(rd *rhel.Distribution) *rhel.ImageType {
	it := rhel.NewImageType(
		"ami",
		"image.raw",
		"application/octet-stream",
		packageSetLoader,
		rhel.DiskImage,
		[]string{"build"},
		[]string{"os", "image"},
		[]string{"image"},
	)

	it.DefaultImageConfig = imageConfig(rd, "aarch64", "ami")
	it.Bootable = true
	it.DefaultSize = 10 * datasizes.GibiByte
	it.BasePartitionTables = partitionTables

	return it
}

func mkEc2ImgTypeAarch64(rd *rhel.Distribution) *rhel.ImageType {
	it := rhel.NewImageType(
		"ec2",
		"image.raw.xz",
		"application/xz",
		packageSetLoader,
		rhel.DiskImage,
		[]string{"build"},
		[]string{"os", "image", "xz"},
		[]string{"xz"},
	)

	it.Compression = "xz"
	it.DefaultImageConfig = imageConfig(rd, "aarch64", "ec2")
	it.Bootable = true
	it.DefaultSize = 10 * datasizes.GibiByte
	it.BasePartitionTables = partitionTables

	return it
}

func mkEc2SapImgTypeX86_64(rd *rhel.Distribution) *rhel.ImageType {
	it := rhel.NewImageType(
		"ec2-sap",
		"image.raw.xz",
		"application/xz",
		packageSetLoader,
		rhel.DiskImage,
		[]string{"build"},
		[]string{"os", "image", "xz"},
		[]string{"xz"},
	)

	it.Compression = "xz"
	it.DefaultImageConfig = imageConfig(rd, "x86_64", "ec2-sap")
	it.Bootable = true
	it.DefaultSize = 10 * datasizes.GibiByte
	it.BasePartitionTables = partitionTables

	return it
}
