package defs_test

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/customizations/oscap"
	"github.com/osbuild/images/pkg/customizations/users"
	"github.com/osbuild/images/pkg/datasizes"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/defs"
	"github.com/osbuild/images/pkg/distro/generic"
	"github.com/osbuild/images/pkg/distro/test_distro"
	"github.com/osbuild/images/pkg/platform"
	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/osbuild/images/pkg/runner"
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

func TestYamlLintClean(t *testing.T) {
	_, err := exec.LookPath("yamllint")
	if errors.Is(err, exec.ErrNotFound) {
		t.Skip("this test needs yamllint")
	}
	require.NoError(t, err)

	pl, err := filepath.Glob("*/*.yaml")
	require.NoError(t, err)
	for _, p := range pl {
		cmd := exec.Command("yamllint", "--format=parsable", p)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		assert.NoError(t, err)
	}
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
          conditions:
            "some-description-1":
              when:
                distro_name: "test-distro"
              append:
                include: [from-condition-inc2]
                exclude: [from-condition-exc2]
            "some-description-2":
              when:
                distro_name: "other-distro"
              append:
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

	pkgSet, err := defs.PackageSets(it)
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

	pkgSet, err := defs.PackageSets(it)
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
    conditions:
      "some description 1":
        when:
          distro_name: "test-distro"
        append:
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
        conditions:
          "some description 2":
            when:
              distro_name: "test-distro"
            append:
              include: [from-condition-inc]
              exclude: [from-condition-exc]
`
	// XXX: we cannot use distro.Name() as it will give us a name+ver
	baseDir := makeFakeDefs(t, test_distro.TestDistroNameBase, fakePkgsSetYaml)
	restore := defs.MockDataFS(baseDir)
	defer restore()

	pkgSet, err := defs.PackageSets(it)
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
              label: "ESP"
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

	partTable, err := defs.PartitionTable(it)
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
					Label:        "ESP",
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

var fakeImageTypesYaml = `
image_types:
  test_type:
    filename: "disk.img"
    image_func: "disk"
    platforms:
      - arch: x86_64
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
      conditions:
        "some description-0":
          when:
            version_equal: "0"
          override:
            test_arch:
              <<: *test_arch_pt
              partitions:
                - <<: *default_part_0
                  size: 111_111_111
        "some description-1":
          when:
            version_greater_or_equal: "1"
            version_less_than: "2"
          override:
            test_arch:
              <<: *test_arch_pt
              partitions:
                - <<: *default_part_0
                  size: 222_222_222
                - <<: *default_part_1
                  payload:
                    <<: *default_part_1_payload
                    fstab_options: "defaults,ro"
        "some description-2":
          when:
            version_greater_or_equal: "2"
          override:
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
	baseDir := makeFakeDefs(t, test_distro.TestDistroNameBase, fakeImageTypesYaml)
	restore := defs.MockDataFS(baseDir)
	defer restore()

	partTable, err := defs.PartitionTable(it)
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
      conditions:
       "some description-2":
          when:
            version_less_than: "2"
          override:
            test_arch:
              <<: *test_arch_pt
              partitions:
                - <<: *default_part_0
                  size: 333_333_333
                - *default_part_1
`
	// XXX: we cannot use distro.Name() as it will give us a name+ver
	baseDir := makeFakeDefs(t, test_distro.TestDistroNameBase, fakeDistroYaml)
	restore := defs.MockDataFS(baseDir)
	defer restore()

	partTable, err := defs.PartitionTable(it)
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
      conditions:
        "some description":
          when:
            distro_name: "test-distro"
          override:
            test_arch:
              partitions:
                - <<: *default_part_0
                  size: 111_111_111
`
	// XXX: we cannot use distro.Name() as it will give us a name+ver
	baseDir := makeFakeDefs(t, test_distro.TestDistroNameBase, fakeDistroYaml)
	restore := defs.MockDataFS(baseDir)
	defer restore()

	partTable, err := defs.PartitionTable(it)
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
    users:
      - name: testuser
  conditions:
    "some description":
      when:
        distro_name: "test-distro"
      shallow_merge:
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
		Users:    []users.User{users.User{Name: "testuser"}},
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

		_, err := defs.PartitionTable(it)
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
      default_kernel: "kernel"
      conditions:
        "some description for version lt":
          when:
            version_less_than: "2"
          shallow_merge:
            timezone: "OverrideTZ"
        "test-distro is version '1' (no minor) so considered '1' is > '1.4'":
          when:
            version_less_than: "1.4"
          shallow_merge:
            default_kernel: kernel-lt-14
        "some description for distro_name":
          when:
            distro_name: "test-distro"
          shallow_merge:
            locale: "en_US.UTF-8"
        "some description for architecture":
          when:
            arch: "test_arch"
          shallow_merge:
            hostname: "test-arch-hn"
        "some description for version":
          when:
            version_less_than: "2"
          shallow_merge:
            default_kernel: "kernel-lt-2"
`
	fakeDistroName := "test-distro"
	baseDir := makeFakeDefs(t, fakeDistroName, fakeDistroYaml)
	restore := defs.MockDataFS(baseDir)
	defer restore()

	imgConfig, err := defs.ImageConfig("test-distro-1", "test_arch", "test_type")
	require.NoError(t, err)
	assert.Equal(t, &distro.ImageConfig{
		Hostname:      common.ToPtr("test-arch-hn"),
		Locale:        common.ToPtr("en_US.UTF-8"),
		Timezone:      common.ToPtr("OverrideTZ"),
		DefaultKernel: common.ToPtr("kernel-lt-2"),
	}, imgConfig)
}

func TestImageTypes(t *testing.T) {
	fakeDistroYaml := `
image_types:
  server-qcow2:
    name_aliases: ["qcow2"]
    filename: "disk.qcow2"
    compression: xz
    mime_type: "application/x-qemu-disk"
    environment:
      packages: ["cloud-init"]
      services: ["cloud-init.service"]
    bootable: true
    boot_iso: true
    rpm_ostree: false
    iso_label: "Workstation"
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
        uefi_vendor: "{{.DistroVendor}}"
`
	fakeDistroName := "test-distro"
	baseDir := makeFakeDefs(t, fakeDistroName, fakeDistroYaml)
	restore := defs.MockDataFS(baseDir)
	defer restore()

	testDistroYAML := `
distros:
 - name: test-distro-1
   vendor: test-vendor
   defs_path: test-distro/
`
	err := os.WriteFile(filepath.Join(baseDir, "distros.yaml"), []byte(testDistroYAML), 0644)
	require.NoError(t, err)

	distro, err := defs.NewDistroYAML("test-distro-1")
	require.NoError(t, err)

	imgTypes := distro.ImageTypes()
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
	assert.Equal(t, true, imgType.BootISO)
	assert.Equal(t, false, imgType.RPMOSTree)
	assert.Equal(t, "Workstation", imgType.ISOLabel)
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
			UEFIVendor:   "test-vendor",
		},
	}, imgType.InternalPlatforms)
}

func TestImageTypesUEFIVendorErrorWhenEmpty(t *testing.T) {
	fakeDistroYaml := `
image_types:
  server-qcow2:
    platforms:
      - arch: x86_64
        uefi_vendor: "{{.DistroVendor}}"
`
	fakeDistroName := "test-distro"
	baseDir := makeFakeDefs(t, fakeDistroName, fakeDistroYaml)
	restore := defs.MockDataFS(baseDir)
	defer restore()

	testDistroYAML := `
distros:
 - name: test-distro-1
   defs_path: test-distro/
`
	err := os.WriteFile(filepath.Join(baseDir, "distros.yaml"), []byte(testDistroYAML), 0644)
	require.NoError(t, err)

	_, err = defs.NewDistroYAML("test-distro-1")
	require.ErrorContains(t, err, `cannot execute template for "vendor" field (is it set?)`)
}

var fakeDistroYamlInstallerConf = `
image_types:
  test_type:
    installer_config:
      additional_dracut_modules:
        - base-dracut-mod1
      additional_drivers:
        - base-drv1
      squashfs_rootfs: true
`

func TestImageTypeInstallerConfig(t *testing.T) {
	fakeDistroYaml := fakeDistroYamlInstallerConf

	fakeDistroName := "test-distro"
	baseDir := makeFakeDefs(t, fakeDistroName, fakeDistroYaml)
	restore := defs.MockDataFS(baseDir)
	defer restore()

	installerConfig, err := defs.InstallerConfig("test-distro-1", "test_arch", "test_type")
	require.NoError(t, err)
	assert.Equal(t, &distro.InstallerConfig{
		AdditionalDracutModules: []string{"base-dracut-mod1"},
		AdditionalDrivers:       []string{"base-drv1"},
		SquashfsRootfs:          common.ToPtr(true),
	}, installerConfig)
}

func TestImageTypeInstallerConfigMergeVerLT(t *testing.T) {
	fakeDistroYaml := fakeDistroYamlInstallerConf + `
      conditions:
        "some description":
          when:
            version_less_than: "2"
          shallow_merge:
            additional_dracut_modules:
              - override-dracut-mod1
`
	fakeDistroName := "test-distro"
	baseDir := makeFakeDefs(t, fakeDistroName, fakeDistroYaml)
	restore := defs.MockDataFS(baseDir)
	defer restore()

	installerConfig, err := defs.InstallerConfig("test-distro-1", "test_arch", "test_type")
	require.NoError(t, err)
	assert.Equal(t, &distro.InstallerConfig{
		// AdditionalDrivers,SquashfsRootfs merged from parent
		AdditionalDrivers:       []string{"base-drv1"},
		SquashfsRootfs:          common.ToPtr(true),
		AdditionalDracutModules: []string{"override-dracut-mod1"},
	}, installerConfig)
}

func TestImageTypeInstallerConfigMergeDistroName(t *testing.T) {
	fakeDistroYaml := fakeDistroYamlInstallerConf + `
      conditions:
        "some description":
          when:
            distro_name: "test-distro"
          shallow_merge:
            additional_dracut_modules:
              - override-dracut-mod1
            additional_drivers:
             - override-drv1
`

	fakeDistroName := "test-distro"
	baseDir := makeFakeDefs(t, fakeDistroName, fakeDistroYaml)
	restore := defs.MockDataFS(baseDir)
	defer restore()

	installerConfig, err := defs.InstallerConfig("test-distro-1", "test_arch", "test_type")
	require.NoError(t, err)
	assert.Equal(t, &distro.InstallerConfig{
		AdditionalDracutModules: []string{"override-dracut-mod1"},
		AdditionalDrivers:       []string{"override-drv1"},
		// SquashfsRootfs merged from parent
		SquashfsRootfs: common.ToPtr(true),
	}, installerConfig)
}

func TestImageTypeInstallerConfigMergeArch(t *testing.T) {
	fakeDistroYaml := fakeDistroYamlInstallerConf + `
      conditions:
        "some description":
          when:
            arch: "test_arch"
          shallow_merge:
            additional_drivers:
             - override-drv1
`

	fakeDistroName := "test-distro"
	baseDir := makeFakeDefs(t, fakeDistroName, fakeDistroYaml)
	restore := defs.MockDataFS(baseDir)
	defer restore()

	installerConfig, err := defs.InstallerConfig("test-distro-1", "test_arch", "test_type")
	require.NoError(t, err)
	assert.Equal(t, &distro.InstallerConfig{
		AdditionalDrivers: []string{"override-drv1"},
		// AdditionalDracutModules,SquashfsRootfs merged from parent
		AdditionalDracutModules: []string{"base-dracut-mod1"},
		SquashfsRootfs:          common.ToPtr(true),
	}, installerConfig)
}

func makeFakeDistrosYAML(t *testing.T, content, imgTypes string) string {
	t.Helper()

	tmpdir := t.TempDir()
	distrosPath := filepath.Join(tmpdir, "distros.yaml")
	err := os.WriteFile(distrosPath, []byte(content), 0644)
	assert.NoError(t, err)

	var di struct {
		Distros []defs.DistroYAML `yaml:"distros"`
	}
	err = yaml.Unmarshal([]byte(content), &di)
	assert.NoError(t, err)
	for _, d := range di.Distros {
		p := filepath.Join(tmpdir, d.DefsPath, "distro.yaml")
		err = os.MkdirAll(filepath.Dir(p), 0755)
		assert.NoError(t, err)
		err = os.WriteFile(p, []byte(`---`+"\n"+imgTypes), 0644)
		assert.NoError(t, err)
	}

	return tmpdir
}

var fakeDistrosYAML = `
distros:
  - &fedora_rawhide
    name: fedora-43
    preview: true
    os_version: 43
    release_version: 43
    module_platform_id: platform:f43
    product: "Fedora"
    ostree_ref_tmpl: "fedora/43/%s/iot"
    defs_path: fedora
    iso_label_tmpl: "{{.Product}}-ISO"
    runner: &fedora_runner
      name: org.osbuild.fedora43
      build_packages: ["glibc"]
    bootstrap_containers:
      x86_64: "registry.fedoraproject.org/fedora-toolbox:43"
    oscap_profiles_allowlist:
      - "xccdf_org.ssgproject.content_profile_ospp"

  - &fedora_stable
    <<: *fedora_rawhide
    name: "fedora-{{.MajorVersion}}"
    match: "fedora-[0-9]*"
    preview: false
    os_version: "{{.MajorVersion}}"
    release_version: "{{.MajorVersion}}"
    module_platform_id: "platform:f{{.MajorVersion}}"
    ostree_ref_tmpl: "fedora/{{.MajorVersion}}/%s/iot"
    runner:
      <<: *fedora_runner
      name: org.osbuild.fedora{{.MajorVersion}}
    bootstrap_containers:
      x86_64: "registry.fedoraproject.org/fedora-toolbox:{{.MajorVersion}}"

  - name: centos-10
    product: "CentOS Stream"
    os_version: "10-stream"
    release_version: 10
    module_platform_id: "platform:el10"
    vendor: "centos"
    ostree_ref_tmpl: "centos/10/%s/edge"
    default_fs_type: "xfs"
    defs_path: rhel-10

  - name: "rhel-{{.MajorVersion}}.{{.MinorVersion}}"
    match: "rhel-10.*"
    product: "Red Hat Enterprise Linux"
    os_version: "{{.MajorVersion}}.{{.MinorVersion}}"
    release_version: "{{.MajorVersion}}"
    module_platform_id: "platform:el{{.MajorVersion}}"
    vendor: "redhat"
    ostree_ref_tmpl: "rhel/{{.MajorVersion}}/%s/edge"
    default_fs_type: "xfs"
    defs_path: rhel-10
`

func TestDistrosLoadingExact(t *testing.T) {
	baseDir := makeFakeDistrosYAML(t, fakeDistrosYAML, "")
	restore := defs.MockDataFS(baseDir)
	defer restore()

	distro, err := defs.NewDistroYAML("fedora-43")
	require.NoError(t, err)
	assert.Equal(t, &defs.DistroYAML{
		Name:             "fedora-43",
		Preview:          true,
		OsVersion:        "43",
		ReleaseVersion:   "43",
		ModulePlatformID: "platform:f43",
		Product:          "Fedora",
		OSTreeRefTmpl:    "fedora/43/%s/iot",
		DefsPath:         "fedora",
		ISOLabelTmpl:     "{{.Product}}-ISO",
		Runner: runner.RunnerConf{
			Name:          "org.osbuild.fedora43",
			BuildPackages: []string{"glibc"},
		},
		BootstrapContainers: map[arch.Arch]string{
			arch.ARCH_X86_64: "registry.fedoraproject.org/fedora-toolbox:43",
		},
		OscapProfilesAllowList: []oscap.Profile{
			oscap.Ospp,
		},
	}, distro)

	distro, err = defs.NewDistroYAML("centos-10")
	require.NoError(t, err)
	assert.Equal(t, &defs.DistroYAML{
		Name:             "centos-10",
		Vendor:           "centos",
		OsVersion:        "10-stream",
		ReleaseVersion:   "10",
		ModulePlatformID: "platform:el10",
		Product:          "CentOS Stream",
		OSTreeRefTmpl:    "centos/10/%s/edge",
		DefsPath:         "rhel-10",
		DefaultFSType:    disk.FS_XFS,
	}, distro)
}

func TestDistrosLoadingFactoryCompat(t *testing.T) {
	baseDir := makeFakeDistrosYAML(t, fakeDistrosYAML, "")
	restore := defs.MockDataFS(baseDir)
	defer restore()

	distro, err := defs.NewDistroYAML("rhel-10.1")
	require.NoError(t, err)
	assert.Equal(t, &defs.DistroYAML{
		Name:             "rhel-10.1",
		Match:            "rhel-10.*",
		Vendor:           "redhat",
		OsVersion:        "10.1",
		ReleaseVersion:   "10",
		ModulePlatformID: "platform:el10",
		Product:          "Red Hat Enterprise Linux",
		OSTreeRefTmpl:    "rhel/10/%s/edge",
		DefsPath:         "rhel-10",
		DefaultFSType:    disk.FS_XFS,
	}, distro)

	distro, err = defs.NewDistroYAML("fedora-40")
	require.NoError(t, err)
	assert.Equal(t, &defs.DistroYAML{
		Name:             "fedora-40",
		Match:            "fedora-[0-9]*",
		OsVersion:        "40",
		ReleaseVersion:   "40",
		ModulePlatformID: "platform:f40",
		Product:          "Fedora",
		OSTreeRefTmpl:    "fedora/40/%s/iot",
		DefsPath:         "fedora",
		ISOLabelTmpl:     "{{.Product}}-ISO",
		Runner: runner.RunnerConf{
			Name:          "org.osbuild.fedora40",
			BuildPackages: []string{"glibc"},
		},
		BootstrapContainers: map[arch.Arch]string{
			arch.ARCH_X86_64: "registry.fedoraproject.org/fedora-toolbox:40",
		},
		OscapProfilesAllowList: []oscap.Profile{
			oscap.Ospp,
		},
	}, distro)
}

func TestDistroYAMLCondition(t *testing.T) {
	fakeImageTypesYaml := `
image_types:
  ec2:
    filename: "disk.raw"
    image_func: "disk"
    exports: ["image"]
    platforms:
      - arch: x86_64
        uefi_vendor: "some-uefi-vendor"
  container:
    filename: "container.tar.gz"
    image_func: "container"
    exports: ["archive"]
    platforms:
      - arch: x86_64
`

	fakeDistrosYAML := `
distros:
 - &rhel8
   name: rhel-8
   conditions:
     "some image types are rhel-only":
       when:
         not_distro_name: "rhel"
       ignore_image_types:
         - ec2
   defs_path: test-distro/
 - <<: *rhel8
   name: centos-8
   defs_path: test-distro/
`
	baseDir := makeFakeDistrosYAML(t, fakeDistrosYAML, fakeImageTypesYaml)
	restore := defs.MockDataFS(baseDir)
	defer restore()

	for _, tc := range []struct {
		distroNameVer    string
		expectedImgTypes []string
	}{
		{"rhel-8", []string{"container", "ec2"}},
		{"centos-8", []string{"container"}},
	} {
		t.Run(tc.distroNameVer, func(t *testing.T) {
			// Note that we load from the "generic" distro here as
			// the resolving of available image types happens on
			// this layer. XXX: consolidate it to the YAML level
			// already?

			distro := generic.DistroFactory(tc.distroNameVer)
			require.NotNil(t, distro)
			assert.Equal(t, tc.distroNameVer, distro.Name())
			a, err := distro.GetArch("x86_64")
			require.NoError(t, err)

			assert.Equal(t, tc.expectedImgTypes, a.ListImageTypes())
		})
	}
}

func TestDistrosLoadingNotFound(t *testing.T) {
	baseDir := makeFakeDistrosYAML(t, fakeDistrosYAML, "")
	restore := defs.MockDataFS(baseDir)
	defer restore()

	distro, err := defs.NewDistroYAML("non-exiting")
	assert.Nil(t, err)
	assert.Nil(t, distro)
}

func TestWhenConditionEvalEmpty(t *testing.T) {
	wc := &defs.WhenCondition{}
	assert.Equal(t, wc.Eval(&distro.ID{Name: "foo"}, "arch"), true)
}

func TestWhenConditionEvalSimple(t *testing.T) {
	wc := &defs.WhenCondition{DistroName: "distro"}
	assert.Equal(t, wc.Eval(&distro.ID{Name: "distro"}, "other-arch"), true)
}

func TestWhenConditionEvalAnd(t *testing.T) {
	wc := &defs.WhenCondition{DistroName: "distro", Architecture: "arch"}
	assert.Equal(t, wc.Eval(&distro.ID{Name: "distro"}, "other-arch"), false)
	assert.Equal(t, wc.Eval(&distro.ID{Name: "distro"}, "arch"), true)
}

func TestImageTypesPlatformOverrides(t *testing.T) {
	fakeImageTypesYaml := `
image_types:
  server-qcow2:
    filename: "disk.qcow2"
    exports: ["qcow2"]
    platforms_override:
      conditions:
        "test platform override, simulate old distro is bios only":
          when:
            version_less_than: "2"
          override:
            - arch: x86_64
              # note no uefi_vendor here
    platforms:
      - arch: x86_64
        uefi_vendor: "some-uefi-vendor"
`

	fakeDistrosYAML := `
distros:
 - name: test-distro-1
   vendor: test-vendor
   defs_path: test-distro/
 - name: test-distro-2
   vendor: test-vendor
   defs_path: test-distro/
`
	baseDir := makeFakeDistrosYAML(t, fakeDistrosYAML, fakeImageTypesYaml)
	restore := defs.MockDataFS(baseDir)
	defer restore()

	for _, tc := range []struct {
		distroNameVer      string
		expectedUEFIVendor string
	}{
		{"test-distro-1", ""},
		{"test-distro-2", "some-uefi-vendor"},
	} {

		distro, err := defs.NewDistroYAML(tc.distroNameVer)
		require.NoError(t, err)

		imgTypes := distro.ImageTypes()
		assert.Len(t, imgTypes, 1)
		imgType := imgTypes["server-qcow2"]
		platforms, err := imgType.PlatformsFor(tc.distroNameVer)
		assert.NoError(t, err)
		assert.Equal(t, []platform.PlatformConf{
			{
				Arch:       arch.ARCH_X86_64,
				UEFIVendor: tc.expectedUEFIVendor,
			},
		}, platforms)
	}
}

func TestImageTypesPlatformOverridesMultiMarchError(t *testing.T) {
	fakeImageTypesYaml := `
image_types:
  server-qcow2:
    filename: "disk.qcow2"
    exports: ["qcow2"]
    platforms:
      - arch: x86_64
    platforms_override:
      conditions:
        "this is true":
          when:
            version_less_than: "2"
          override:
            - arch: x86_64
              uefi_vendor: "uefi-for-ver-2"
        "this is also true":
          when:
            version_less_than: "3"
          override:
            - arch: x86_64
              uefi_vendor: "uefi-for-ver-3"
`

	fakeDistrosYAML := `
distros:
 - name: test-distro-1
   vendor: test-vendor
   defs_path: test-distro/
`
	baseDir := makeFakeDistrosYAML(t, fakeDistrosYAML, fakeImageTypesYaml)
	restore := defs.MockDataFS(baseDir)
	defer restore()

	distro, err := defs.NewDistroYAML("test-distro-1")
	assert.NoError(t, err)
	imgTypes := distro.ImageTypes()
	assert.Len(t, imgTypes, 1)
	imgType := imgTypes["server-qcow2"]
	_, err = imgType.PlatformsFor("test-distro-1")
	assert.EqualError(t, err, `platform conditionals for image type "server-qcow2" should match only once but matched 2 times`)
}
