package defs_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/datasizes"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/defs"
	"github.com/osbuild/images/pkg/distro/test_distro"
	"github.com/osbuild/images/pkg/platform"
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

func makeFakeDefs(t *testing.T, distroName, content string) string {
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
      os:
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
      container:
        - include: [inc-cnt1]
          exclude: [exc-cnt1]
`
	// XXX: we cannot use distro.Name() as it will give us a name+ver
	baseDir := makeFakeDefs(t, test_distro.TestDistroNameBase, fakePkgsSetYaml)
	restore := defs.MockDataFS(baseDir)
	defer restore()

	pkgSet, err := defs.PackageSets(it, nil)
	assert.NoError(t, err)
	assert.Equal(t, map[string]rpmmd.PackageSet{
		"os": {
			Include: []string{"from-condition-inc2", "inc1"},
			Exclude: []string{"exc1", "from-condition-exc2"},
		},
		"container": {
			Include: []string{"inc-cnt1"},
			Exclude: []string{"exc-cnt1"},
		},
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
     os:
      - include:
          - inc1
        exclude:
          - exc1

  unrelated:
    package_sets:
     os:
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

	pkgSet, err := defs.PackageSets(it, nil)
	assert.NoError(t, err)
	assert.Equal(t, map[string]rpmmd.PackageSet{
		"os": {
			Include: []string{"inc1"},
			Exclude: []string{"exc1"},
		},
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
     os:
      - &other_type_pkgset
        include: [from-other-type-inc]
        exclude: [from-other-type-exc]
  test_type:
    package_sets:
     os:
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
	baseDir := makeFakeDefs(t, test_distro.TestDistroNameBase, fakePkgsSetYaml)
	restore := defs.MockDataFS(baseDir)
	defer restore()

	pkgSet, err := defs.PackageSets(it, nil)
	assert.NoError(t, err)
	assert.Equal(t, map[string]rpmmd.PackageSet{
		"os": {
			Include: []string{"from-base-condition-inc", "from-base-inc", "from-condition-inc", "from-other-type-inc", "from-type-inc"},
			Exclude: []string{"from-base-condition-exc", "from-base-exc", "from-condition-exc", "from-other-type-exc", "from-type-exc"},
		},
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
	baseDir := makeFakeDefs(t, test_distro.TestDistroNameBase, fakeDistroYaml)
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

var fakeDistroYaml = `
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
          "2":
            test_arch:
              <<: *test_arch_pt
              partitions:
                - <<: *default_part_0
                  size: 333_333_333
                - *default_part_1
`

func TestDefsPartitionTableOverrideGreatEqual(t *testing.T) {
	it := makeTestImageType(t)

	// XXX: we cannot use distro.Name() as it will give us a name+ver
	baseDir := makeFakeDefs(t, test_distro.TestDistroNameBase, fakeDistroYaml)
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

func TestDefsPartitionTableOverridelessThan(t *testing.T) {
	it := makeTestImageType(t)

	patched := strings.Replace(fakeDistroYaml, "version_greater_or_equal:", "version_less_than:", -1)

	// XXX: we cannot use distro.Name() as it will give us a name+ver
	baseDir := makeFakeDefs(t, test_distro.TestDistroNameBase, patched)
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
				Size:     333_333_333,
				Bootable: true,
			},
			{
				Size: 2_147_483_648,
				Payload: &disk.Filesystem{
					Type:         "ext4",
					Label:        "root",
					Mountpoint:   "/",
					FSTabOptions: "defaults",
				},
			},
		},
	}, partTable)
}

func TestDefsPartitionTableOverrideDistoName(t *testing.T) {
	it := makeTestImageType(t)

	fakeDistroYaml := `
image_types:
  test_type:
    partition_table:
      test_arch: &test_arch_pt
        partitions:
          - &default_part_0
            size: 1_048_576
            bootable: true
    partition_tables_override:
      condition:
        distro_name:
          "test-distro":
              test_arch:
                partitions:
                  - <<: *default_part_0
                    size: 111_111_111
`
	// XXX: we cannot use distro.Name() as it will give us a name+ver
	baseDir := makeFakeDefs(t, test_distro.TestDistroNameBase, fakeDistroYaml)
	restore := defs.MockDataFS(baseDir)
	defer restore()

	partTable, err := defs.PartitionTable(it, nil)
	require.NoError(t, err)
	assert.Equal(t, &disk.PartitionTable{
		Partitions: []disk.Partition{
			{
				Size:     111_111_111,
				Bootable: true,
			},
		},
	}, partTable)
}

func TestDefsDistroImageConfig(t *testing.T) {
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
	baseDir := makeFakeDefs(t, fakeDistroName, fakeDistroYaml)
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
		baseDir := makeFakeDefs(t, test_distro.TestDistroNameBase, tc.badYaml)
		restore := defs.MockDataFS(baseDir)
		defer restore()

		_, err := defs.PartitionTable(it, nil)
		assert.ErrorIs(t, err, tc.expectedErr)
	}
}

func TestImageTypeImageConfig(t *testing.T) {
	fakeDistroYaml := `
image_types:
  test_type:
    image_config:
      hostname: "foo"
      locale: "C.UTF-8"
      timezone: "DefaultTZ"
      condition:
        version_less_than:
          "2":
            timezone: "OverrideTZ"
        distro_name:
          "test-distro":
            locale: "en_US.UTF-8"
        architecture:
          "test_arch":
            hostname: "test-arch-hn"
`
	fakeDistroName := "test-distro"
	baseDir := makeFakeDefs(t, fakeDistroName, fakeDistroYaml)
	restore := defs.MockDataFS(baseDir)
	defer restore()

	imgConfig, err := defs.ImageConfig("test-distro-1", "test_arch", "test_type", nil)
	require.NoError(t, err)
	assert.Equal(t, &distro.ImageConfig{
		Hostname: common.ToPtr("test-arch-hn"),
		Locale:   common.ToPtr("en_US.UTF-8"),
		Timezone: common.ToPtr("OverrideTZ"),
	}, imgConfig)

}

func TestImageTypes(t *testing.T) {
	fakeDistroYaml := `
image_types:
  server_qcow2:
    name_aliases: ["qcow2"]
    filename: "disk.qcow2"
    compression: xz
    mime_type: "application/x-qemu-disk"
    environment:
      packages: ["cloud-init"]
      services: ["cloud-init.service"]
    bootable: true
    default_size: 5_368_709_120  # 5 * datasizes.GibiByte
    image_func: "disk"
    build_pipelines: ["build"]
    payload_pipelines: ["os", "image", "qcow2"]
    exports: ["qcow2"]
    required_partition_sizes:
      "/": 1_073_741_824  # 1 * datasizes.GiB
    platforms:
      - arch: ppc64le
        bios_platform: "powerpc-ieee1275"
        image_format: "qcow2"
        qcow2_compat: "1.1"
`
	fakeDistroName := "test-distro"
	baseDir := makeFakeDefs(t, fakeDistroName, fakeDistroYaml)
	restore := defs.MockDataFS(baseDir)
	defer restore()

	imgTypes, err := defs.ImageTypes("test-distro-1")
	require.NoError(t, err)
	assert.Len(t, imgTypes, 1)
	imgType := imgTypes["server-qcow2"]
	assert.Equal(t, "server-qcow2", imgType.Name())
	assert.Equal(t, []string{"qcow2"}, imgType.NameAliases)
	assert.Equal(t, "disk.qcow2", imgType.Filename)
	assert.Equal(t, "xz", imgType.Compression)
	assert.Equal(t, "application/x-qemu-disk", imgType.MimeType)
	assert.Equal(t, []string{"cloud-init"}, imgType.Environment.GetPackages())
	assert.Len(t, imgType.Environment.GetRepos(), 0)
	assert.Equal(t, []string{"cloud-init.service"}, imgType.Environment.GetServices())
	assert.Equal(t, true, imgType.Bootable)
	assert.Equal(t, uint64(5*datasizes.GibiByte), imgType.DefaultSize)
	assert.Equal(t, "disk", imgType.Image)
	assert.Equal(t, []string{"build"}, imgType.BuildPipelines)
	assert.Equal(t, []string{"os", "image", "qcow2"}, imgType.PayloadPipelines)
	assert.Equal(t, []string{"qcow2"}, imgType.Exports)
	assert.Equal(t, map[string]uint64{"/": 1_073_741_824}, imgType.RequiredPartitionSizes)
	assert.Equal(t, []platform.PlatformConf{
		{
			Arch:         arch.ARCH_PPC64LE,
			BIOSPlatform: "powerpc-ieee1275",
			ImageFormat:  platform.FORMAT_QCOW2,
			QCOW2Compat:  "1.1",
		},
	}, imgType.Platforms)
}

var fakeDistroYamlInstallerConf = `
image_types:
  test_type:
    installer_config:
      additional_dracut_modules:
        - base-dracut-mod1
      additional_drivers:
        - base-drv1
`

func TestImageTypeInstallerConfig(t *testing.T) {
	fakeDistroYaml := fakeDistroYamlInstallerConf

	fakeDistroName := "test-distro"
	baseDir := makeFakeDefs(t, fakeDistroName, fakeDistroYaml)
	restore := defs.MockDataFS(baseDir)
	defer restore()

	installerConfig, err := defs.InstallerConfig("test-distro-1", "test_arch", "test_type", nil)
	require.NoError(t, err)
	assert.Equal(t, &distro.InstallerConfig{
		AdditionalDracutModules: []string{"base-dracut-mod1"},
		AdditionalDrivers:       []string{"base-drv1"},
	}, installerConfig)
}

func TestImageTypeInstallerConfigOverrideVerLT(t *testing.T) {
	fakeDistroYaml := fakeDistroYamlInstallerConf + `
      condition:
        version_less_than:
          "2":
            # Note that this fully override the installer config
            additional_dracut_modules:
              - override-dracut-mod1
`
	fakeDistroName := "test-distro"
	baseDir := makeFakeDefs(t, fakeDistroName, fakeDistroYaml)
	restore := defs.MockDataFS(baseDir)
	defer restore()

	installerConfig, err := defs.InstallerConfig("test-distro-1", "test_arch", "test_type", nil)
	require.NoError(t, err)
	assert.Equal(t, &distro.InstallerConfig{
		AdditionalDracutModules: []string{"override-dracut-mod1"},
		// Note that there is no "AdditionalDrivers" here as
		// the InstallerConfig is fully replaced, do any
		// merging in YAML
	}, installerConfig)
}

func TestImageTypeInstallerConfigOverrideDistroName(t *testing.T) {
	fakeDistroYaml := fakeDistroYamlInstallerConf + `
      condition:
        distro_name:
          "test-distro":
            additional_dracut_modules:
              - override-dracut-mod1
            additional_drivers:
             - override-drv1
`

	fakeDistroName := "test-distro"
	baseDir := makeFakeDefs(t, fakeDistroName, fakeDistroYaml)
	restore := defs.MockDataFS(baseDir)
	defer restore()

	installerConfig, err := defs.InstallerConfig("test-distro-1", "test_arch", "test_type", nil)
	require.NoError(t, err)
	assert.Equal(t, &distro.InstallerConfig{
		AdditionalDracutModules: []string{"override-dracut-mod1"},
		AdditionalDrivers:       []string{"override-drv1"},
	}, installerConfig)
}

func TestImageTypeInstallerConfigOverrideArch(t *testing.T) {
	fakeDistroYaml := fakeDistroYamlInstallerConf + `
      condition:
        architecture:
          "test_arch":
            additional_drivers:
             - override-drv1
`

	fakeDistroName := "test-distro"
	baseDir := makeFakeDefs(t, fakeDistroName, fakeDistroYaml)
	restore := defs.MockDataFS(baseDir)
	defer restore()

	installerConfig, err := defs.InstallerConfig("test-distro-1", "test_arch", "test_type", nil)
	require.NoError(t, err)
	assert.Equal(t, &distro.InstallerConfig{
		AdditionalDrivers: []string{"override-drv1"},
	}, installerConfig)
}
