package packagesets_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/packagesets"
	"github.com/osbuild/images/pkg/distro/test_distro"
	"github.com/osbuild/images/pkg/rpmmd"
)

func makeTestImageType(t *testing.T) distro.ImageType {
	// XXX: it would be nice if testdistro had a ready-made image-type,
	// i.e. testdistro.TestImageType1
	distro := test_distro.DistroFactory(test_distro.TestDistro1Name)
	arch, err := distro.GetArch(test_distro.TestArchName)
	assert.NoError(t, err)
	it, err := arch.GetImageType(test_distro.TestImageTypeName)
	assert.NoError(t, err)
	return it
}

func makeFakePkgsSet(t *testing.T, distroName, content string) string {
	tmpdir := t.TempDir()
	fakePkgsSetPath := filepath.Join(tmpdir, distroName, "package_sets.yaml")
	err := os.MkdirAll(filepath.Dir(fakePkgsSetPath), 0755)
	assert.NoError(t, err)
	err = os.WriteFile(fakePkgsSetPath, []byte(content), 0644)
	assert.NoError(t, err)
	return tmpdir
}

func TestLoadConditionDistro(t *testing.T) {
	it := makeTestImageType(t)
	fakePkgsSetYaml := `
test_type:
  include: [inc1]
  exclude: [exc1]
  condition:
    distro_name:
      test-distro:
        include: [from-condition-inc2]
        exclude: [from-condition-exc2]
      other-distro:
        include: [inc3]
        exclude: [exc3]
`
	// XXX: we cannot use distro.Name() as it will give us a name+ver
	baseDir := makeFakePkgsSet(t, test_distro.TestDistroNameBase, fakePkgsSetYaml)
	restore := packagesets.MockDataFS(baseDir)
	defer restore()

	pkgSet, err := packagesets.Load(it, "", nil)
	assert.NoError(t, err)
	assert.Equal(t, rpmmd.PackageSet{
		Include: []string{"inc1", "from-condition-inc2"},
		Exclude: []string{"exc1", "from-condition-exc2"},
	}, pkgSet)
}

func TestLoadOverrideTypeName(t *testing.T) {
	it := makeTestImageType(t)
	fakePkgsSetYaml := `
override_name:
  include: [from-override-inc1]
  exclude: [from-override-exc1]
test_type:
  include: [default-inc2]
  exclude: [default-exc2]
`
	// XXX: we cannot use distro.Name() as it will give us a name+ver
	baseDir := makeFakePkgsSet(t, test_distro.TestDistroNameBase, fakePkgsSetYaml)
	restore := packagesets.MockDataFS(baseDir)
	defer restore()

	pkgSet, err := packagesets.Load(it, "override-name", nil)
	assert.NoError(t, err)
	assert.Equal(t, rpmmd.PackageSet{
		Include: []string{"from-override-inc1"},
		Exclude: []string{"from-override-exc1"},
	}, pkgSet)
}
