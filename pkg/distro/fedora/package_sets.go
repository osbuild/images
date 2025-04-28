package fedora

import (
	"fmt"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/internal/environment"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/defs"
	"github.com/osbuild/images/pkg/platform"
	"github.com/osbuild/images/pkg/rpmmd"
)

func packageSetLoader(t *imageType) (rpmmd.PackageSet, error) {
	return defs.PackageSet(t, "", VersionReplacements())
}

func imageConfig(d distribution, imageType string) *distro.ImageConfig {
	// arch is currently not used in fedora
	arch := ""
	return common.Must(defs.ImageConfig(d.name, arch, imageType, VersionReplacements()))
}

// XXX: move to generic code, this is not fedora specific
func newPlatformFromYaml(arch string, d defs.ImagePlatform) (res platform.Platform) {
	switch arch {
	case "x86_64":
		res = &platform.X86{
			BIOS:         d.BIOS,
			UEFIVendor:   d.UEFIVendor,
			BasePlatform: d.BasePlatform,
		}
	case "aarch64":
		res = &platform.Aarch64{
			UEFIVendor:   d.UEFIVendor,
			BasePlatform: d.BasePlatform,
		}
	case "ppc64le":
		res = &platform.PPC64LE{
			BIOS:         d.BIOS,
			BasePlatform: d.BasePlatform,
		}
	case "s390x":
		res = &platform.S390X{
			Zipl:         d.Zipl,
			BasePlatform: d.BasePlatform,
		}
	default:
		err := fmt.Errorf("unsupported platform %v", arch)
		panic(err)
	}

	return res
}

func newImageTypeFromYaml(d distribution, typeName string) imageType {
	imgYAML, err := defs.ImageType(d.name, typeName)
	if err != nil {
		panic(err)
	}
	it := imageType{
		name:                   imgYAML.Name,
		filename:               imgYAML.Filename,
		mimeType:               imgYAML.MimeType,
		kernelOptions:          imgYAML.KernelOptions,
		bootable:               imgYAML.Bootable,
		defaultSize:            imgYAML.DefaultSize,
		buildPipelines:         imgYAML.BuildPipelines,
		payloadPipelines:       imgYAML.PayloadPipelines,
		exports:                imgYAML.Exports,
		requiredPartitionSizes: imgYAML.RequiredPartitionSizes,
	}
	it.defaultImageConfig = imageConfig(d, typeName)
	switch imgYAML.Image {
	case "disk":
		it.image = diskImage
	case "container":
		it.image = containerImage
	default:
		err := fmt.Errorf("unknown image func: %v for %v", imgYAML.Image, imgYAML.Name)
		panic(err)
	}
	switch imgYAML.Environment {
	case "":
		// nothing
	case "azure":
		it.environment = &environment.Azure{}
	case "ec2":
		it.environment = &environment.EC2{}
	case "kvm":
		it.environment = &environment.KVM{}
	default:
		err := fmt.Errorf("unknown env %q", imgYAML.Environment)
		panic(err)
	}
	// XXX: fix for multiple loaders like the installers
	it.packageSets = map[string]packageSetFunc{
		osPkgsKey: packageSetLoader,
	}

	return it
}
