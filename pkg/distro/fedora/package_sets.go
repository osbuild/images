package fedora

import (
	"fmt"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/internal/environment"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/defs"
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
	default:
		err := fmt.Errorf("unknown image func: %v", imgYAML.Image)
		panic(err)
	}
	switch imgYAML.Environment {
	case "kvm":
		it.environment = &environment.KVM{}
	}
	// XXX: fix for multiple loaders like the installers
	it.packageSets = map[string]packageSetFunc{
		osPkgsKey: packageSetLoader,
	}

	return it
}
