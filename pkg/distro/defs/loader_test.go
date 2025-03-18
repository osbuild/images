package defs_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/defs"
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
	fakePkgsSetPath := filepath.Join(tmpdir, distroName, "distro.yaml")
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
	restore := defs.MockDataFS(baseDir)
	defer restore()

	pkgSet, err := defs.PackageSet(it, "", nil)
	assert.NoError(t, err)
	assert.Equal(t, rpmmd.PackageSet{
		Include: []string{"from-condition-inc2", "inc1"},
		Exclude: []string{"exc1", "from-condition-exc2"},
	}, pkgSet)
}

func TestLoadOverrideTypeName(t *testing.T) {
	it := makeTestImageType(t)
	fakePkgsSetYaml := `
image_types:
  test_type:
    package_sets:
      - include: [default-inc2]
        exclude: [default-exc2]
  override_name:
    package_sets:
      - include: [from-override-inc1]
        exclude: [from-override-exc1]

`
	// XXX: we cannot use distro.Name() as it will give us a name+ver
	baseDir := makeFakePkgsSet(t, test_distro.TestDistroNameBase, fakePkgsSetYaml)
	restore := defs.MockDataFS(baseDir)
	defer restore()

	pkgSet, err := defs.PackageSet(it, "override-name", nil)
	assert.NoError(t, err)
	assert.Equal(t, rpmmd.PackageSet{
		Include: []string{"from-override-inc1"},
		Exclude: []string{"from-override-exc1"},
	}, pkgSet)
}

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
image_types:
  test_type:
    package_sets:
      - include:
          - inc1
        exclude:
          - exc1

  unrelated:
    package_sets:
      - include:
          - inc2
        exclude:
          - exc2
`)
	// XXX: we cannot use distro.Name() as it will give us a name+ver
	fakePkgsSetPath := filepath.Join(tmpdir, test_distro.TestDistroNameBase, "distro.yaml")
	err = os.MkdirAll(filepath.Dir(fakePkgsSetPath), 0755)
	assert.NoError(t, err)
	err = os.WriteFile(fakePkgsSetPath, fakePkgsSetYaml, 0644)
	assert.NoError(t, err)

	pkgSet, err := defs.PackageSet(it, "", nil)
	assert.NoError(t, err)
	assert.Equal(t, rpmmd.PackageSet{
		Include: []string{"inc1"},
		Exclude: []string{"exc1"},
	}, pkgSet)
}

func TestLoadYamlMergingWorks(t *testing.T) {
	it := makeTestImageType(t)
	fakePkgsSetYaml := `
.common:
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
	restore := defs.MockDataFS(baseDir)
	defer restore()

	pkgSet, err := defs.PackageSet(it, "", nil)
	assert.NoError(t, err)
	assert.Equal(t, rpmmd.PackageSet{
		Include: []string{"from-base-condition-inc", "from-base-inc", "from-condition-inc", "from-other-type-inc", "from-type-inc"},
		Exclude: []string{"from-base-condition-exc", "from-base-exc", "from-condition-exc", "from-other-type-exc", "from-type-exc"},
	}, pkgSet)
}
