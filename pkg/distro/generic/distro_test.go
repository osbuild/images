package generic

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/pkg/distro/defs"
	testrepos "github.com/osbuild/images/test/data/repositories"
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

func TestBootstrapContainers(t *testing.T) {
	repos, err := testrepos.New()
	assert.NoError(t, err)

	for _, distroName := range repos.ListDistros() {
		t.Run(distroName, func(t *testing.T) {
			d := DistroFactory(distroName)
			// TODO: remove once everthing is a generic distro
			if d == nil {
				t.Skipf("%s not a generic distro yet", distroName)
			}
			assert.NotNil(t, d)
			assert.NotEmpty(t, d.(*distribution).DistroYAML.BootstrapContainers)
		})
	}
}
