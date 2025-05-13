package fedora

import (
	"fmt"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/distro/defs"
)

func getISOLabelFunc(variant string) isoLabelFunc {
	const ISO_LABEL = "%s-%s-%s-%s"

	return func(t *imageType) string {
		return fmt.Sprintf(ISO_LABEL, t.Arch().Distro().Product(), t.Arch().Distro().OsVersion(), variant, t.Arch().Name())
	}

}

func newImageTypeFrom(d *distribution, ar *architecture, imgYAML defs.ImageTypeYAML) imageType {
	typName := imgYAML.Name()
	it := imageType{
		name:                   typName,
		nameAliases:            imgYAML.NameAliases,
		filename:               imgYAML.Filename,
		compression:            imgYAML.Compression,
		mimeType:               imgYAML.MimeType,
		bootable:               imgYAML.Bootable,
		bootISO:                imgYAML.BootISO,
		rpmOstree:              imgYAML.RPMOSTree,
		isoLabel:               getISOLabelFunc(imgYAML.ISOLabel),
		defaultSize:            imgYAML.DefaultSize,
		buildPipelines:         imgYAML.BuildPipelines,
		payloadPipelines:       imgYAML.PayloadPipelines,
		exports:                imgYAML.Exports,
		requiredPartitionSizes: imgYAML.RequiredPartitionSizes,
		environment:            &imgYAML.Environment,
	}
	it.defaultImageConfig = common.Must(defs.ImageConfig(d.name, ar.name, typName, VersionReplacements()))
	it.defaultInstallerConfig = common.Must(defs.InstallerConfig(d.name, ar.name, typName, VersionReplacements()))

	switch imgYAML.Image {
	case "disk":
		it.image = diskImage
	case "container":
		it.image = containerImage
	case "image_installer":
		it.image = imageInstallerImage
	case "live_installer":
		it.image = liveInstallerImage
	case "bootable_container":
		it.image = bootableContainerImage
	case "iot":
		it.image = iotImage
	case "iot_commit":
		it.image = iotCommitImage
	case "iot_container":
		it.image = iotContainerImage
	case "iot_installer":
		it.image = iotInstallerImage
	case "iot_simplified_installer":
		it.image = iotSimplifiedInstallerImage
	case "tar":
		it.image = tarImage
	default:
		err := fmt.Errorf("unknown image func: %v for %v", imgYAML.Image, imgYAML.Name())
		panic(err)
	}

	return it
}
