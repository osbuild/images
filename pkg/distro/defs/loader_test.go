package defs_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/disk"
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

func TestDefsPartitionTable(t *testing.T) {
	it := makeTestImageType(t)
	fakeDistroYaml := `
image_types:
  test_type:
    partition_table:
      test_arch:
        size: 1_000_000_000
        uuid: "D209C89E-EA5E-4FBD-B161-B461CCE297E0"
        type: "gpt"
        partitions:
          - size: 1_048_576
            bootable: true
          - payload_type: filesystem
            size: 209_715_200
            payload:
              type: vfat
              mountpoint: "/boot/efi"
              label: "EFI-SYSTEM"
              fstab_options: "defaults,uid=0,gid=0,umask=077,shortname=winnt"
              fstab_freq: 0
              fstab_passno: 2
          - payload_type: "luks"
            payload:
              label: "crypt_root"
              cipher: "cipher_null"
              passphrase: "osbuild"
              pbkdf:
                iterations: 4
              clevis:
                pin: "null"
                remove_passphrase: true
              payload_type: "lvm"
              payload:
                name: "rootvg"
                description: "bla"
                logical_volumes:
                  - size: 8_589_934_592  # 8 * datasizes.GibiByte,
                    name: rootlv
                    payload_type: "filesystem"
                    payload:
                      type: ext4
                      mountpoint: "/"
`
	// XXX: we cannot use distro.Name() as it will give us a name+ver
	baseDir := makeFakePkgsSet(t, test_distro.TestDistroNameBase, fakeDistroYaml)
	restore := defs.MockDataFS(baseDir)
	defer restore()

	partTable, err := defs.PartitionTable(it, nil)
	require.NoError(t, err)
	assert.Equal(t, &disk.PartitionTable{
		Size: 1_000_000_000,
		UUID: "D209C89E-EA5E-4FBD-B161-B461CCE297E0",
		Type: disk.PT_GPT,
		Partitions: []disk.Partition{
			{
				Size:     1048576,
				Bootable: true,
			},
			{
				Size: 209_715_200,
				Payload: &disk.Filesystem{
					Type:         "vfat",
					Mountpoint:   "/boot/efi",
					Label:        "EFI-SYSTEM",
					FSTabOptions: "defaults,uid=0,gid=0,umask=077,shortname=winnt",
					FSTabFreq:    0,
					FSTabPassNo:  2,
				},
			}, {
				Payload: &disk.LUKSContainer{
					Label:      "crypt_root",
					Cipher:     "cipher_null",
					Passphrase: "osbuild",
					PBKDF: disk.Argon2id{
						Iterations: 4,
					},
					Clevis: &disk.ClevisBind{
						Pin:              "null",
						RemovePassphrase: true,
					},
					Payload: &disk.LVMVolumeGroup{
						Name:        "rootvg",
						Description: "bla",
						LogicalVolumes: []disk.LVMLogicalVolume{
							{
								Name: "rootlv",
								Size: 8_589_934_592,
								Payload: &disk.Filesystem{
									Type:       "ext4",
									Mountpoint: "/",
								},
							},
						},
					},
				},
			},
		},
	}, partTable)
}

func TestDefsPartitionTableOverride(t *testing.T) {
	it := makeTestImageType(t)
	fakeDistroYaml := `
image_types:
  test_type:
    partition_table:
      test_arch: &test_arch_pt
        size: 1_000_000_000
        uuid: "D209C89E-EA5E-4FBD-B161-B461CCE297E0"
        type: "gpt"
        partitions:
          - &default_part_0
            size: 1_048_576
            bootable: true
          - &default_part_1
            size: 2_147_483_648
            payload_type: "filesystem"
            payload: &default_part_1_payload
              type: "ext4"
              label: "root"
              mountpoint: "/"
              fstab_options: "defaults"
    partition_tables_override:
      condition:
        version_greater_or_equal:
          # overrides are applied in order
          "0":
            test_arch:
              <<: *test_arch_pt
              partitions:
                - <<: *default_part_0
                  size: 111_111_111
          "1":
            test_arch:
              <<: *test_arch_pt
              partitions:
                - <<: *default_part_0
                  size: 222_222_222
                - <<: *default_part_1
                  payload:
                    <<: *default_part_1_payload
                    fstab_options: "defaults,ro"
`
	// XXX: we cannot use distro.Name() as it will give us a name+ver
	baseDir := makeFakePkgsSet(t, test_distro.TestDistroNameBase, fakeDistroYaml)
	restore := defs.MockDataFS(baseDir)
	defer restore()

	partTable, err := defs.PartitionTable(it, nil)
	require.NoError(t, err)
	assert.Equal(t, &disk.PartitionTable{
		Size: 1_000_000_000,
		UUID: "D209C89E-EA5E-4FBD-B161-B461CCE297E0",
		Type: disk.PT_GPT,
		Partitions: []disk.Partition{
			{
				Size:     222_222_222,
				Bootable: true,
			},
			{
				Size: 2_147_483_648,
				Payload: &disk.Filesystem{
					Type:         "ext4",
					Label:        "root",
					Mountpoint:   "/",
					FSTabOptions: "defaults,ro",
				},
			},
		},
	}, partTable)
}

func TestDefsImageConfig(t *testing.T) {
	fakeDistroYaml := `
image_config:
  default:
    locale: "C.UTF-8"
    timezone: "DefaultTZ"
  condition:
    distro_name:
      "test-distro":
        timezone: "OverrideTZ"
`
	fakeDistroName := "test-distro"
	fakeDistroVer := "42"
	baseDir := makeFakePkgsSet(t, fakeDistroName, fakeDistroYaml)
	restore := defs.MockDataFS(baseDir)
	defer restore()

	distroNameVer := fakeDistroName + "-" + fakeDistroVer
	imgConfig, err := defs.DistroImageConfig(distroNameVer)
	assert.NoError(t, err)
	assert.Equal(t, &distro.ImageConfig{
		Locale:   common.ToPtr("C.UTF-8"),
		Timezone: common.ToPtr("OverrideTZ"),
	}, imgConfig)
}

func TestDefsPartitionTableErrorsNotForImageType(t *testing.T) {
	it := makeTestImageType(t)

	badDistroYamlUnknownImgType := `
image_types:
  other_image_type:
    partition_table:
      test_arch:
        partitions:
          - size: 1_048_576
`
	badDistroYamlMissingPartitionTable := `
image_types:
  test_type:
`
	badDistroYamlUnknownArch := `
image_types:
  test_type:
    partition_table:
      other_arch:
        partitions:
          - size: 1_048_576
`

	for _, tc := range []struct {
		badYaml     string
		expectedErr error
	}{
		{badDistroYamlUnknownImgType, defs.ErrImageTypeNotFound},
		{badDistroYamlMissingPartitionTable, defs.ErrNoPartitionTableForImgType},
		{badDistroYamlUnknownArch, defs.ErrNoPartitionTableForArch},
	} {
		// XXX: we cannot use distro.Name() as it will give us a name+ver
		baseDir := makeFakePkgsSet(t, test_distro.TestDistroNameBase, tc.badYaml)
		restore := defs.MockDataFS(baseDir)
		defer restore()

		_, err := defs.PartitionTable(it, nil)
		assert.ErrorIs(t, err, tc.expectedErr)
	}
}
