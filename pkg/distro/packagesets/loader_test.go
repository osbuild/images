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
image_types:
  test_type:
    package_sets:
      - include: [inc1]
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

	pkgSet := packagesets.Load(it, nil)
	assert.NotNil(t, pkgSet)
	assert.Equal(t, rpmmd.PackageSet{
		Include: []string{"from-condition-inc2", "inc1"},
		Exclude: []string{"exc1", "from-condition-exc2"},
	}, pkgSet)
}

func TestLoadYamlMergingWorks(t *testing.T) {
	it := makeTestImageType(t)
	fakePkgsSetYaml := `
# package_sets not associated with an imagetype can be defined here
virtual_package_sets:
  base: &base_pkgset
    include: [from-base-inc]
    exclude: [from-base-exc]
    condition:
      distro_name:
        test-distro:
          include: [from-base-condition-inc]
          exclude: [from-base-condition-exc]

image_types:
  other_type:
    package_sets:
      - &other_type_pkgset
        include: [from-other-type-inc]
        exclude: [from-other-type-exc]
  test_type:
    package_sets:
      - *base_pkgset
      - *other_type_pkgset
      - include: [from-type-inc]
        exclude: [from-type-exc]
        condition:
          distro_name:
            test-distro:
              include: [from-condition-inc]
              exclude: [from-condition-exc]
`
	// XXX: we cannot use distro.Name() as it will give us a name+ver
	baseDir := makeFakePkgsSet(t, test_distro.TestDistroNameBase, fakePkgsSetYaml)
	restore := packagesets.MockDataFS(baseDir)
	defer restore()

	pkgSet := packagesets.Load(it, nil)
	assert.NotNil(t, pkgSet)
	assert.Equal(t, rpmmd.PackageSet{
		Include: []string{"from-base-condition-inc", "from-base-inc", "from-condition-inc", "from-other-type-inc", "from-type-inc"},
		Exclude: []string{"from-base-condition-exc", "from-base-exc", "from-condition-exc", "from-other-type-exc", "from-type-exc"},
	}, pkgSet)
}
