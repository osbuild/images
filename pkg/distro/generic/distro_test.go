package generic_test

import (
	"testing"

	"github.com/osbuild/images/pkg/distro/generic"
	testrepos "github.com/osbuild/images/test/data/repositories"
	"github.com/stretchr/testify/assert"
)

func TestBootstrapContainers(t *testing.T) {
	repos, err := testrepos.New()
	assert.NoError(t, err)

	for _, distroName := range repos.ListDistros() {
		t.Run(distroName, func(t *testing.T) {
			d := generic.DistroFactory(distroName)
			// TODO: remove once everthing is a generic distro
			if d == nil {
				t.Skipf("%s not a generic distro yet", distroName)
			}
			assert.NotNil(t, d)
			assert.NotEmpty(t, d.(*generic.Distribution).DistroYAML.BootstrapContainers)
		})
	}
}
