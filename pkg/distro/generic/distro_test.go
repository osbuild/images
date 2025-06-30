package generic

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/pkg/distro/defs"
)

func TestISOLabel(t *testing.T) {
	imgType := &imageType{
		arch: &architecture{
			name: "some-arch",
		},
	}
	d := &distribution{
		DistroYAML: defs.DistroYAML{
			Name:         "rhel-9.1",
			Product:      "some-product",
			ISOLabelTmpl: "name:{{.Distro.Name}},major:{{.Distro.MajorVersion}},minor:{{.Distro.MinorVersion}},product:{{.Product}},arch:{{.Arch}},iso-label:{{.ISOLabel}}",
		},
	}

	isoLabelFunc := d.getISOLabelFunc("iso-label")
	assert.Equal(t, "name:rhel,major:9,minor:1,product:some-product,arch:some-arch,iso-label:iso-label", isoLabelFunc(imgType))
}
