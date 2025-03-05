package packagesets_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/pkg/distro/packagesets"
	"github.com/osbuild/images/pkg/distro/test_distro"
	"github.com/osbuild/images/pkg/rpmmd"
)

func TestLoadExperimentalYamldirIsHonored(t *testing.T) {
	// XXX: it would be nice if testdistro had a ready-made image-type,
	// i.e. testdistro.TestImageType1
	distro := test_distro.DistroFactory(test_distro.TestDistro1Name)
	arch, err := distro.GetArch(test_distro.TestArchName)
	assert.NoError(t, err)
	it, err := arch.GetImageType(test_distro.TestImageTypeName)
	assert.NoError(t, err)

	tmpdir := t.TempDir()
	t.Setenv("IMAGE_BUILDER_EXPERIMENTAL", fmt.Sprintf("yamldir=%s", tmpdir))

	fakePkgsSetYaml := []byte(`
test_type:
  include:
    - inc1
  exclude:
    - exc1

unrelated:
  include:
    - inc2
  exclude:
    - exc2
`)
	// XXX: we cannot use distro.Name() as it will give us a name+ver
	fakePkgsSetPath := filepath.Join(tmpdir, test_distro.TestDistroNameBase, "package_sets.yaml")
	err = os.MkdirAll(filepath.Dir(fakePkgsSetPath), 0755)
	assert.NoError(t, err)
	err = os.WriteFile(fakePkgsSetPath, fakePkgsSetYaml, 0644)
	assert.NoError(t, err)

	pkgSet := packagesets.Load(it, nil)
	assert.NotNil(t, pkgSet)
	assert.Equal(t, rpmmd.PackageSet{
		Include: []string{"inc1"},
		Exclude: []string{"exc1"},
	}, pkgSet)
}
