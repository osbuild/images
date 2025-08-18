package manifest_test

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/customizations/kickstart"
	"github.com/osbuild/images/pkg/datasizes"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/ostree"
	"github.com/osbuild/images/pkg/platform"
	"github.com/osbuild/images/pkg/runner"
)

const (
	testKsPath     = "/test.ks"
	testBaseKsPath = "/test-base.ks"
)

// newTestAnacondaISOTree returns a base AnacondaInstallerISOTree pipeline.
func newTestAnacondaISOTree() *manifest.AnacondaInstallerISOTree {
	m := &manifest.Manifest{}
	runner := &runner.Linux{}
	build := manifest.NewBuild(m, runner, nil, nil)

	x86plat := &platform.X86{}

	product := "test-iso"
	osversion := "1"

	preview := false

	anacondaPipeline := manifest.NewAnacondaInstaller(
		manifest.AnacondaInstallerTypePayload,
		build,
		x86plat,
		nil,
		"kernel",
		product,
		osversion,
		preview,
	)
	rootfsImagePipeline := manifest.NewISORootfsImg(build, anacondaPipeline)
	bootTreePipeline := manifest.NewEFIBootTree(build, product, osversion)
	bootTreePipeline.ISOLabel = "test-iso-1"

	pipeline := manifest.NewAnacondaInstallerISOTree(build, anacondaPipeline, rootfsImagePipeline, bootTreePipeline)
	// copy of the default in pkg/image - will be moved to the pipeline
	var efibootImageSize uint64 = 20 * datasizes.MebiByte
	pipeline.PartitionTable = &disk.PartitionTable{
		Size: efibootImageSize,
		Partitions: []disk.Partition{
			{
				Start: 0,
				Size:  efibootImageSize,
				Payload: &disk.Filesystem{
					Type:       "vfat",
					Mountpoint: "/",
					// math/rand is good enough in this case
					/* #nosec G404 */
					UUID: disk.NewVolIDFromRand(rand.New(rand.NewSource(0))),
				},
			},
		},
	}
	return pipeline
}

// Helper to return a comma separated string of the stage names
// used to help debug failures
func dumpStages(stages []*osbuild.Stage) string {
	var stageNames []string
	for _, stage := range stages {
		stageNames = append(stageNames, stage.Type)
	}
	return strings.Join(stageNames, ", ")
}

func checkISOTreeStages(stages []*osbuild.Stage, expected, exclude []string) error {
	commonStages := []string{
		"org.osbuild.mkdir",
		"org.osbuild.copy",
		"org.osbuild.squashfs",
		"org.osbuild.truncate",
		"org.osbuild.mkfs.fat",
		"org.osbuild.copy",
		"org.osbuild.copy",
		"org.osbuild.discinfo",
	}

	// Remove excluded stages from common
	for _, exlStage := range exclude {
		if idx := slices.Index(commonStages, exlStage); idx > -1 {
			commonStages = slices.Delete(commonStages, idx, idx+1)
		}
	}

	for _, expStage := range append(commonStages, expected...) {
		if findStage(expStage, stages) == nil {
			return fmt.Errorf("did not find expected stage: %s", expStage)
		}
	}

	for _, exlStage := range exclude {
		if findStage(exlStage, stages) != nil {
			return fmt.Errorf("stage in pipeline should not have been added: %s", exlStage)
		}
	}
	return nil
}

func getKickstartOptions(stages []*osbuild.Stage) *osbuild.KickstartStageOptions {
	ksStage := findStage("org.osbuild.kickstart", stages)
	options, ok := ksStage.Options.(*osbuild.KickstartStageOptions)
	if !ok {
		panic("kickstart stage options conversion failed")
	}
	return options
}

func findRawKickstartFileStage(stages []*osbuild.Stage) *osbuild.CopyStageOptions {
	// the pipeline can have more than one copy stage - find the one that has
	// the expected destination for the kickstart file
	for _, stage := range stages {
		if stage.Type == "org.osbuild.copy" {
			options, ok := stage.Options.(*osbuild.CopyStageOptions)
			if !ok {
				panic("copy stage options conversion failed")
			}
			if options.Paths[0].To == "tree://"+testKsPath {
				return options
			}
		}
	}
	return nil
}

const (
	ksContainerContent = `reqpart --add-boot

part swap --fstype=swap --size=1024
part / --fstype=ext4 --grow

reboot --eject
%post
bootc switch --mutate-in-place --transport registry local.example.org/registry/org/image
%end
`
)

var (
	ksSudoPost = osbuild.PostOptions{
		Commands: []string{
			`echo -e "%sudo\tALL=(ALL)\tNOPASSWD: ALL" > "/etc/sudoers.d/%sudo"`,
			`chmod 0440 /etc/sudoers.d/%sudo`,
			`echo -e "%wheel\tALL=(ALL)\tNOPASSWD: ALL" > "/etc/sudoers.d/%wheel"`,
			`chmod 0440 /etc/sudoers.d/%wheel`,
			`restorecon -rvF /etc/sudoers.d`,
		},
	}
)

func calculateInlineFileChecksum(parts ...string) string {
	content := "%include /run/install/repo/test-base.ks\n"
	for _, part := range parts {
		content += part
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(content)))
}

func checkKickstartOptions(stages []*osbuild.Stage, unattended, sudopost bool, extra string) error {
	ksParts := make([]string, 0)
	if extra != "" {
		// adding extra bits replaces any other inline kickstart file
		ksParts = []string{extra}
	}

	ksCopyStageOptions := findRawKickstartFileStage(stages)
	expRawFile := len(ksParts) > 0
	if expRawFile && ksCopyStageOptions == nil { // raw kickstart stage (file stage) should exist
		return fmt.Errorf("expected raw kickstart file but not found")
	} else if !expRawFile && ksCopyStageOptions != nil { // no raw kickstart file stage should be found
		return fmt.Errorf("found raw kickstart file but was not expected")
	}

	if ksCopyStageOptions != nil {
		contentHash := calculateInlineFileChecksum(ksParts...)
		expContentID := fmt.Sprintf("input://file-%[1]s/sha256:%[1]s", contentHash)
		// inline file IDs are the hash of their content so this is the hash of the expected content
		if inlineID := ksCopyStageOptions.Paths[0].From; inlineID != expContentID {
			return fmt.Errorf("raw kickstart content mismatch: %s != %s", expContentID, inlineID)
		}
	}

	ksOptions := getKickstartOptions(stages)

	// check the kickstart path depending on whether we have extra raw content included
	if expRawFile && ksOptions.Path != testBaseKsPath {
		return fmt.Errorf("kickstart file path should be %q but is %q", testBaseKsPath, ksOptions.Path)
	} else if !expRawFile && ksOptions.Path != testKsPath {
		return fmt.Errorf("kickstart file path should be %q but is %q", testKsPath, ksOptions.Path)
	}

	if unattended {
		// check that the unattended kickstart options are set
		if ksOptions.DisplayMode != "text" {
			return fmt.Errorf("unexpected kickstart display mode for unattended: %q", ksOptions.DisplayMode)
		}
		if !ksOptions.Reboot.Eject {
			return fmt.Errorf("unattended reboot.eject kickstart option unset")
		}
		if !ksOptions.RootPassword.Lock {
			return fmt.Errorf("unattended rootpassword.lock kickstart option unset")
		}
		if !ksOptions.ZeroMBR {
			return fmt.Errorf("unattended zerombr kickstart option unset")
		}
		if !ksOptions.ClearPart.All {
			return fmt.Errorf("unattended clearpart.all kickstart option unset")
		}
		if !ksOptions.ClearPart.InitLabel {
			return fmt.Errorf("unattended clearpart.initlabel kickstart option unset")
		}

		// just check that some options are set to anything since at this level the
		// values don't matter and can change based on distro defaults
		if ksOptions.Lang == "" {
			return fmt.Errorf("unattended lang kickstart option unset")
		}
		if ksOptions.Timezone == "" {
			return fmt.Errorf("unattended timezone kickstart option unset")
		}
		if ksOptions.Keyboard == "" {
			return fmt.Errorf("unattended keyboard kickstart option unset")
		}
		if ksOptions.AutoPart == nil {
			return fmt.Errorf("unattended autopart kickstart option unset")
		}
		if ksOptions.Network == nil {
			return fmt.Errorf("unattended network kickstart option unset")
		}
	}

	if sudopost {
		foundSudoPost := false
		for _, postOptions := range ksOptions.Post {
			if reflect.DeepEqual(postOptions, ksSudoPost) {
				foundSudoPost = true
			}
		}
		if !foundSudoPost {
			return fmt.Errorf("expected post options for sudoers dropins but was not found")
		}
	}

	return nil
}

func checkRawKickstartForContainer(stages []*osbuild.Stage, extra string) error {
	ksParts := []string{ksContainerContent}
	if extra != "" {
		ksParts = []string{extra}
	}
	ksCopyStageOptions := findRawKickstartFileStage(stages)
	if ksCopyStageOptions == nil { // raw kickstart stage (file stage) should exist
		return fmt.Errorf("expected raw kickstart file but not found")
	}

	if ksCopyStageOptions != nil {
		contentHash := calculateInlineFileChecksum(ksParts...)
		expContentID := fmt.Sprintf("input://file-%[1]s/sha256:%[1]s", contentHash)
		// inline file IDs are the hash of their content so this is the hash of the expected content
		if inlineID := ksCopyStageOptions.Paths[0].From; inlineID != expContentID {
			return fmt.Errorf("raw kickstart content mismatch: %s != %s", expContentID, inlineID)
		}
	}

	ksOptions := getKickstartOptions(stages)

	// check the kickstart path depending on whether we have extra raw content included
	if ksOptions.Path != testBaseKsPath {
		return fmt.Errorf("kickstart file path should be %q but is %q", testBaseKsPath, ksOptions.Path)
	}

	return nil
}

func TestAnacondaISOTreePayloadsBad(t *testing.T) {
	assert := assert.New(t)
	pipeline := newTestAnacondaISOTree()

	assert.PanicsWithValue(
		"pipeline supports at most one ostree commit",
		func() { manifest.SerializeWith(pipeline, manifest.Inputs{Commits: make([]ostree.CommitSpec, 2)}) },
	)
	assert.PanicsWithValue(
		"pipeline supports at most one container",
		func() { manifest.SerializeWith(pipeline, manifest.Inputs{Containers: make([]container.Spec, 2)}) },
	)
}

func TestAnacondaISOTreeSerializeWithOS(t *testing.T) {
	osPayload := manifest.NewTestOS()

	// stages required for the payload type
	payloadStages := []string{"org.osbuild.tar"}

	// stages that should only appear for the other variants of the pipeline
	variantStages := []string{
		"org.osbuild.ostree.init",
		"org.osbuild.ostree.pull",
		"org.osbuild.skopeo",
	}

	t.Run("plain", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.OSPipeline = osPayload
		sp := manifest.SerializeWith(pipeline, manifest.Inputs{})
		assert.NoError(t, checkISOTreeStages(sp.Stages, payloadStages,
			append(variantStages, []string{"org.osbuild.kickstart", "org.osbuild.isolinux"}...)))
	})

	// the os payload variant of the pipeline only adds the kickstart file if
	// KSPath is defined
	t.Run("kspath", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.OSPipeline = osPayload
		pipeline.Kickstart = &kickstart.Options{Path: testKsPath}
		sp := manifest.SerializeWith(pipeline, manifest.Inputs{})
		assert.NoError(t, checkISOTreeStages(sp.Stages, append(payloadStages, "org.osbuild.kickstart"),
			append(variantStages, "org.osbuild.isolinux")))
	})

	// enable syslinux iso and check for stage
	t.Run("kspath+syslinux", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.OSPipeline = osPayload
		pipeline.Kickstart = &kickstart.Options{Path: testKsPath}
		pipeline.ISOBoot = manifest.SyslinuxISOBoot
		sp := manifest.SerializeWith(pipeline, manifest.Inputs{})
		assert.NoError(t, checkISOTreeStages(sp.Stages, append(payloadStages, "org.osbuild.isolinux", "org.osbuild.kickstart"),
			variantStages))
	})

	// enable grub2 iso and check for stage
	t.Run("kspath+grub2iso", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.OSPipeline = osPayload
		pipeline.Kickstart = &kickstart.Options{Path: testKsPath}
		pipeline.ISOBoot = manifest.Grub2ISOBoot
		sp := manifest.SerializeWith(pipeline, manifest.Inputs{})

		// No isolinux stage
		assert.Error(t, checkISOTreeStages(sp.Stages, append(payloadStages, "org.osbuild.isolinux",
			"org.osbuild.kickstart"), variantStages))

		// Grub2 BIOS iso uses org.osbuild.grub2.iso.legacy
		assert.NoError(t, checkISOTreeStages(sp.Stages, append(payloadStages, "org.osbuild.grub2.iso.legacy",
			"org.osbuild.kickstart"), variantStages))
	})

	t.Run("unattended", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.OSPipeline = osPayload
		pipeline.Kickstart = &kickstart.Options{Path: testKsPath, Unattended: true}
		pipeline.ISOBoot = manifest.SyslinuxISOBoot
		sp := manifest.SerializeWith(pipeline, manifest.Inputs{})
		assert.NoError(t, checkISOTreeStages(sp.Stages, append(payloadStages, "org.osbuild.isolinux", "org.osbuild.kickstart"), variantStages))
		assert.NoError(t, checkKickstartOptions(sp.Stages, pipeline.Kickstart.Unattended, len(pipeline.Kickstart.SudoNopasswd) > 0, ""))
	})

	t.Run("unattended+sudo", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.OSPipeline = osPayload
		pipeline.Kickstart = &kickstart.Options{
			Path:         testKsPath,
			Unattended:   true,
			SudoNopasswd: []string{`%wheel`, `%sudo`},
		}
		pipeline.ISOBoot = manifest.SyslinuxISOBoot
		sp := manifest.SerializeWith(pipeline, manifest.Inputs{})
		assert.NoError(t, checkISOTreeStages(sp.Stages, append(payloadStages, "org.osbuild.isolinux", "org.osbuild.kickstart"), variantStages))
		assert.NoError(t, checkKickstartOptions(sp.Stages, pipeline.Kickstart.Unattended, len(pipeline.Kickstart.SudoNopasswd) > 0, ""))
	})

	t.Run("user-kickstart-without-sudo-bits", func(t *testing.T) {
		userks := "%post\necho 'Some kind of text in a file sent by post'\n%end"
		pipeline := newTestAnacondaISOTree()
		pipeline.OSPipeline = osPayload
		pipeline.Kickstart = &kickstart.Options{
			Path:       testKsPath,
			Unattended: false,
			UserFile: &kickstart.File{
				Contents: userks,
			},
		}
		pipeline.ISOBoot = manifest.SyslinuxISOBoot
		sp := manifest.SerializeWith(pipeline, manifest.Inputs{})
		assert.NoError(t, checkISOTreeStages(sp.Stages, append(payloadStages, "org.osbuild.isolinux", "org.osbuild.kickstart"), variantStages))
		assert.NoError(t, checkKickstartOptions(sp.Stages, pipeline.Kickstart.Unattended, len(pipeline.Kickstart.SudoNopasswd) > 0, userks))
	})

	t.Run("unhappy/user-kickstart-with-unattended", func(t *testing.T) {
		userks := "%post\necho 'Some kind of text in a file sent by post'\n%end"
		pipeline := newTestAnacondaISOTree()
		pipeline.OSPipeline = osPayload
		pipeline.Kickstart = &kickstart.Options{
			Path:       testKsPath,
			Unattended: true,
			UserFile: &kickstart.File{
				Contents: userks,
			},
		}
		pipeline.ISOBoot = manifest.SyslinuxISOBoot
		assert.Panics(t, func() { manifest.SerializeWith(pipeline, manifest.Inputs{}) })
	})

	t.Run("unhappy/user-kickstart-with-sudo-bits", func(t *testing.T) {
		userks := "%post\necho 'Some kind of text in a file sent by post'\n%end"
		pipeline := newTestAnacondaISOTree()
		pipeline.OSPipeline = osPayload
		pipeline.Kickstart = &kickstart.Options{
			Path:         testKsPath,
			SudoNopasswd: []string{`%wheel`, `%sudo`},
			UserFile: &kickstart.File{
				Contents: userks,
			},
		}
		pipeline.ISOBoot = manifest.SyslinuxISOBoot
		assert.Panics(t, func() { manifest.SerializeWith(pipeline, manifest.Inputs{}) })
	})

	t.Run("plain+squashfs-rootfs", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.OSPipeline = osPayload
		pipeline.RootfsType = manifest.SquashfsRootfs
		sp := manifest.SerializeWith(pipeline, manifest.Inputs{})
		assert.NoError(t, checkISOTreeStages(sp.Stages, payloadStages,
			append(variantStages, []string{"org.osbuild.kickstart", "org.osbuild.isolinux"}...)),
			dumpStages(sp.Stages))
	})

	t.Run("plain+erofs-rootfs", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.OSPipeline = osPayload
		pipeline.RootfsType = manifest.ErofsRootfs
		sp := manifest.SerializeWith(pipeline, manifest.Inputs{})
		assert.NoError(t, checkISOTreeStages(sp.Stages,
			append(payloadStages, "org.osbuild.erofs"),
			append(variantStages, []string{"org.osbuild.kickstart", "org.osbuild.isolinux", "org.osbuild.squashfs"}...)),
			dumpStages(sp.Stages))
	})
}

func TestAnacondaISOTreeSerializeWithOSTree(t *testing.T) {
	ostreeCommit := ostree.CommitSpec{
		Ref:      "test/99/ostree",
		URL:      "http://example.com/ostree/repo",
		Checksum: "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	}
	// stages required for the payload type
	payloadStages := []string{
		"org.osbuild.ostree.init",
		"org.osbuild.ostree.pull",
		"org.osbuild.kickstart",
	}

	// stages that should only appear for the other variants of the pipeline
	variantStages := []string{
		"org.osbuild.tar",
		"org.osbuild.skopeo",
	}

	t.Run("plain", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.Kickstart = &kickstart.Options{Path: testKsPath, OSTree: &kickstart.OSTree{}}
		sp := manifest.SerializeWith(pipeline, manifest.Inputs{Commits: []ostree.CommitSpec{ostreeCommit}})
		assert.NoError(t, checkISOTreeStages(sp.Stages, payloadStages,
			append(variantStages, "org.osbuild.isolinux")))
	})

	// enable syslinux iso and check for stage
	t.Run("isolinux", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.Kickstart = &kickstart.Options{Path: testKsPath, OSTree: &kickstart.OSTree{}}
		pipeline.ISOBoot = manifest.SyslinuxISOBoot
		sp := manifest.SerializeWith(pipeline, manifest.Inputs{Commits: []ostree.CommitSpec{ostreeCommit}})
		assert.NoError(t, checkISOTreeStages(sp.Stages, append(payloadStages, "org.osbuild.isolinux"), variantStages))
	})

	t.Run("unattended", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.Kickstart = &kickstart.Options{Path: testKsPath, Unattended: true, OSTree: &kickstart.OSTree{}}
		pipeline.ISOBoot = manifest.SyslinuxISOBoot
		sp := manifest.SerializeWith(pipeline, manifest.Inputs{Commits: []ostree.CommitSpec{ostreeCommit}})
		assert.NoError(t, checkISOTreeStages(sp.Stages, append(payloadStages, "org.osbuild.isolinux"), variantStages))
		assert.NoError(t, checkKickstartOptions(sp.Stages, pipeline.Kickstart.Unattended, len(pipeline.Kickstart.SudoNopasswd) > 0, ""))
	})

	t.Run("unattended+sudo", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.Kickstart = &kickstart.Options{
			Path:         testKsPath,
			Unattended:   true,
			SudoNopasswd: []string{`%wheel`, `%sudo`},
			OSTree:       &kickstart.OSTree{},
		}
		pipeline.ISOBoot = manifest.SyslinuxISOBoot
		sp := manifest.SerializeWith(pipeline, manifest.Inputs{Commits: []ostree.CommitSpec{ostreeCommit}})
		assert.NoError(t, checkISOTreeStages(sp.Stages, append(payloadStages, "org.osbuild.isolinux"), variantStages))
		assert.NoError(t, checkKickstartOptions(sp.Stages, pipeline.Kickstart.Unattended, len(pipeline.Kickstart.SudoNopasswd) > 0, ""))
	})

	t.Run("user-kickstart-without-sudo-bits", func(t *testing.T) {
		userks := "%post\necho 'Some kind of text in a file sent by post'\n%end"
		pipeline := newTestAnacondaISOTree()
		pipeline.Kickstart = &kickstart.Options{
			Path:       testKsPath,
			Unattended: false,
			UserFile: &kickstart.File{
				Contents: userks,
			},
			OSTree: &kickstart.OSTree{},
		}
		pipeline.ISOBoot = manifest.SyslinuxISOBoot
		sp := manifest.SerializeWith(pipeline, manifest.Inputs{Commits: []ostree.CommitSpec{ostreeCommit}})
		assert.NoError(t, checkISOTreeStages(sp.Stages, append(payloadStages, "org.osbuild.isolinux"), variantStages))
		assert.NoError(t, checkKickstartOptions(sp.Stages, pipeline.Kickstart.Unattended, len(pipeline.Kickstart.SudoNopasswd) > 0, userks))
	})

	t.Run("unhappy/user-kickstart-with-unattended", func(t *testing.T) {
		userks := "%post\necho 'Some kind of text in a file sent by post'\n%end"
		pipeline := newTestAnacondaISOTree()
		pipeline.Kickstart = &kickstart.Options{
			Path:       testKsPath,
			Unattended: true,
			UserFile: &kickstart.File{
				Contents: userks,
			},
			OSTree: &kickstart.OSTree{},
		}
		pipeline.ISOBoot = manifest.SyslinuxISOBoot
		assert.Panics(t, func() { manifest.SerializeWith(pipeline, manifest.Inputs{Commits: []ostree.CommitSpec{ostreeCommit}}) })
	})

	t.Run("unhappy/user-kickstart-with-sudo-bits", func(t *testing.T) {
		userks := "%post\necho 'Some kind of text in a file sent by post'\n%end"
		pipeline := newTestAnacondaISOTree()
		pipeline.Kickstart = &kickstart.Options{
			Path:       testKsPath,
			Unattended: false,
			UserFile: &kickstart.File{
				Contents: userks,
			},
			SudoNopasswd: []string{`%wheel`, `%sudo`},
			OSTree:       &kickstart.OSTree{},
		}
		pipeline.ISOBoot = manifest.SyslinuxISOBoot
		assert.Panics(t, func() { manifest.SerializeWith(pipeline, manifest.Inputs{Commits: []ostree.CommitSpec{ostreeCommit}}) })
	})

	t.Run("plain+squashfs-rootfs", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.RootfsType = manifest.SquashfsRootfs
		pipeline.Kickstart = &kickstart.Options{Path: testKsPath, OSTree: &kickstart.OSTree{}}
		sp := manifest.SerializeWith(pipeline, manifest.Inputs{Commits: []ostree.CommitSpec{ostreeCommit}})
		assert.NoError(t, checkISOTreeStages(sp.Stages, payloadStages, append(variantStages, "org.osbuild.isolinux")), dumpStages(sp.Stages))
	})

	t.Run("plain+erofs-erofs", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.RootfsType = manifest.ErofsRootfs
		pipeline.Kickstart = &kickstart.Options{Path: testKsPath, OSTree: &kickstart.OSTree{}}
		sp := manifest.SerializeWith(pipeline, manifest.Inputs{Commits: []ostree.CommitSpec{ostreeCommit}})
		assert.NoError(t, checkISOTreeStages(sp.Stages,
			append(payloadStages, "org.osbuild.erofs"),
			append(variantStages, []string{"org.osbuild.isolinux", "org.osbuild.squashfs"}...)),
			dumpStages(sp.Stages))
	})
}

func makeFakeContainerPayload() container.Spec {
	return container.Spec{
		Source:    "example.org/registry/org/image",
		Digest:    "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		ImageID:   "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		LocalName: "local.example.org/registry/org/image",
	}
}

func TestAnacondaISOTreeSerializeWithContainer(t *testing.T) {
	containerPayload := makeFakeContainerPayload()
	payloadStages := []string{
		"org.osbuild.skopeo",
		"org.osbuild.kickstart",
	}

	// stages that should only appear for the other variants of the pipeline
	variantStages := []string{
		"org.osbuild.tar",
		"org.osbuild.ostree.init",
		"org.osbuild.ostree.pull",
	}

	t.Run("kspath", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.Kickstart = &kickstart.Options{Path: testKsPath}
		sp := manifest.SerializeWith(pipeline, manifest.Inputs{Containers: []container.Spec{containerPayload}})
		assert.NoError(t, checkISOTreeStages(sp.Stages, payloadStages, append(variantStages, "org.osbuild.isolinux")))
	})

	// enable syslinux iso and check again
	t.Run("isolinux", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.Kickstart = &kickstart.Options{Path: testKsPath}
		pipeline.ISOBoot = manifest.SyslinuxISOBoot
		sp := manifest.SerializeWith(pipeline, manifest.Inputs{Containers: []container.Spec{containerPayload}})
		assert.NoError(t, checkISOTreeStages(sp.Stages, append(payloadStages, "org.osbuild.isolinux"), variantStages))
	})

	t.Run("kernel-options", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.Kickstart = &kickstart.Options{
			Path:                testKsPath,
			Unattended:          true,
			KernelOptionsAppend: []string{"kernel.opt=1", "debug"},
		}
		sp := manifest.SerializeWith(pipeline, manifest.Inputs{Containers: []container.Spec{containerPayload}})
		kickstartSt := findStage("org.osbuild.kickstart", sp.Stages)
		assert.NotNil(t, kickstartSt)
		opts := kickstartSt.Options.(*osbuild.KickstartStageOptions)
		assert.Equal(t, "kernel.opt=1 debug", opts.Bootloader.Append)
	})

	t.Run("network-on-boot", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.Kickstart = &kickstart.Options{Path: testKsPath, NetworkOnBoot: true}
		sp := manifest.SerializeWith(pipeline, manifest.Inputs{Containers: []container.Spec{containerPayload}})
		kickstartSt := findStage("org.osbuild.kickstart", sp.Stages)
		assert.NotNil(t, kickstartSt)
		opts := kickstartSt.Options.(*osbuild.KickstartStageOptions)
		assert.Equal(t, 1, len(opts.Network))
		assert.Equal(t, "on", opts.Network[0].OnBoot)
	})

	t.Run("user-kickstart", func(t *testing.T) {
		userks := "%post\necho 'Some kind of text in a file sent by post'\n%end"
		pipeline := newTestAnacondaISOTree()
		pipeline.Kickstart = &kickstart.Options{
			Path: testKsPath,
			UserFile: &kickstart.File{
				Contents: userks,
			},
		}
		pipeline.ISOBoot = manifest.SyslinuxISOBoot
		sp := manifest.SerializeWith(pipeline, manifest.Inputs{Containers: []container.Spec{containerPayload}})
		assert.NoError(t, checkISOTreeStages(sp.Stages, append(payloadStages, "org.osbuild.isolinux"), variantStages))
		assert.NoError(t, checkRawKickstartForContainer(sp.Stages, userks))
	})

	t.Run("remove-payload-signtures", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.Kickstart = &kickstart.Options{Path: testKsPath}
		pipeline.PayloadRemoveSignatures = true
		sp := manifest.SerializeWith(pipeline, manifest.Inputs{Containers: []container.Spec{containerPayload}})
		skopeoStage := findStage("org.osbuild.skopeo", sp.Stages)
		assert.NotNil(t, skopeoStage)
		assert.Equal(t, skopeoStage.Options.(*osbuild.SkopeoStageOptions).RemoveSignatures, common.ToPtr(true))
	})

	t.Run("plain+squashfs-rootfs", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.RootfsType = manifest.SquashfsRootfs
		pipeline.Kickstart = &kickstart.Options{Path: testKsPath}
		sp := manifest.SerializeWith(pipeline, manifest.Inputs{Containers: []container.Spec{containerPayload}})
		assert.NoError(t, checkISOTreeStages(sp.Stages, payloadStages,
			append(variantStages, "org.osbuild.isolinux")),
			dumpStages(sp.Stages))
	})

	t.Run("plain+erofs-rootfs", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.RootfsType = manifest.ErofsRootfs
		pipeline.Kickstart = &kickstart.Options{Path: testKsPath}
		sp := manifest.SerializeWith(pipeline, manifest.Inputs{Containers: []container.Spec{containerPayload}})
		assert.NoError(t, checkISOTreeStages(sp.Stages,
			append(payloadStages, "org.osbuild.erofs"),
			append(variantStages, []string{"org.osbuild.isolinux", "org.osbuild.squashfs"}...)),
			dumpStages(sp.Stages))
	})
}

func TestMakeKickstartSudoersPost(t *testing.T) {
	exp := &osbuild.PostOptions{
		Commands: []string{
			`echo -e "%group31\tALL=(ALL)\tNOPASSWD: ALL" > "/etc/sudoers.d/%group31"`,
			`chmod 0440 /etc/sudoers.d/%group31`,
			`echo -e "user42\tALL=(ALL)\tNOPASSWD: ALL" > "/etc/sudoers.d/user42"`,
			`chmod 0440 /etc/sudoers.d/user42`,
			`restorecon -rvF /etc/sudoers.d`,
		},
	}
	assert.Equal(t, exp, manifest.MakeKickstartSudoersPost([]string{"user42", "%group31"}))
	assert.Equal(t, exp, manifest.MakeKickstartSudoersPost([]string{"%group31", "user42"}))
	assert.Equal(t, exp, manifest.MakeKickstartSudoersPost([]string{"%group31", "user42", "%group31"}))
	assert.Equal(t, exp, manifest.MakeKickstartSudoersPost([]string{"%group31", "user42", "%group31", "%group31", "user42", "%group31", "%group31", "user42", "%group31"}))
}

func stagesFrom(pipeline manifest.Pipeline) []*osbuild.Stage {
	containerPayload := makeFakeContainerPayload()
	sp := manifest.SerializeWith(pipeline, manifest.Inputs{Containers: []container.Spec{containerPayload}})
	return sp.Stages
}

func TestPayloadRemoveSignatures(t *testing.T) {
	for _, tc := range []struct {
		removeSig bool
		expected  *bool
	}{
		{true, common.ToPtr(true)},
		{false, nil},
	} {
		pipeline := newTestAnacondaISOTree()
		pipeline.Kickstart = &kickstart.Options{Path: testKsPath}
		pipeline.PayloadRemoveSignatures = tc.removeSig

		skopeoStage := findStage("org.osbuild.skopeo", stagesFrom(pipeline))
		assert.NotNil(t, skopeoStage)
		assert.Equal(t, tc.expected, skopeoStage.Options.(*osbuild.SkopeoStageOptions).RemoveSignatures)
	}
}

func TestISORootfsType(t *testing.T) {
	var rootFS struct {
		ISORootfsType manifest.ISORootfsType `yaml:"iso_rootfs_type"`
	}

	for _, tc := range []struct {
		fstype   string
		expected manifest.ISORootfsType
	}{
		{"squashfs-ext4", manifest.SquashfsExt4Rootfs},
		{"squashfs", manifest.SquashfsRootfs},
		{"erofs", manifest.ErofsRootfs},
	} {
		err := yaml.Unmarshal([]byte(fmt.Sprintf("iso_rootfs_type: %s", tc.fstype)), &rootFS)
		require.NoError(t, err)
		assert.Equal(t, tc.expected, rootFS.ISORootfsType)
	}
}

func TestISORootfsTypeError(t *testing.T) {
	var rootFS struct {
		ISORootfsType manifest.ISORootfsType `yaml:"iso_rootfs_type"`
	}

	err := yaml.Unmarshal([]byte("iso_rootfs_type: non-exiting"), &rootFS)
	assert.EqualError(t, err, `unmarshal yaml via json for "non-exiting" failed: unknown ISORootfsType: "non-exiting"`)
}

func TestISOBootType(t *testing.T) {
	var isoBoot struct {
		ISOBootType manifest.ISOBootType `yaml:"iso_boot_type"`
	}

	for _, tc := range []struct {
		boottype string
		expected manifest.ISOBootType
	}{
		{"", manifest.Grub2UEFIOnlyISOBoot},
		{"grub2-uefi", manifest.Grub2UEFIOnlyISOBoot},
		{"syslinux", manifest.SyslinuxISOBoot},
		{"grub2", manifest.Grub2ISOBoot},
	} {
		err := yaml.Unmarshal([]byte(fmt.Sprintf("iso_boot_type: %s", tc.boottype)), &isoBoot)
		require.NoError(t, err)
		assert.Equal(t, tc.expected, isoBoot.ISOBootType)
	}
}

func TestISOBootTypeError(t *testing.T) {
	var isoBoot struct {
		ISOBootType manifest.ISOBootType `yaml:"iso_boot_type"`
	}

	err := yaml.Unmarshal([]byte("iso_boot_type: lilo"), &isoBoot)
	assert.EqualError(t, err, `unmarshal yaml via json for "lilo" failed: unknown ISOBootType: "lilo"`)
}

func TestAnacondaISOTreeSerializeInstallRootfsType(t *testing.T) {
	for _, tc := range []struct {
		fs       disk.FSType
		expected string
	}{
		{disk.FS_BTRFS, "autopart --nohome --type=btrfs\n"},
		{disk.FS_XFS, "autopart --nohome --type=plain --fstype=xfs\n"},
		{disk.FS_VFAT, "autopart --nohome --type=plain --fstype=vfat\n"},
		{disk.FS_EXT4, "autopart --nohome --type=plain --fstype=ext4\n"},
		{disk.FS_NONE, "autopart --nohome --type=plain --fstype=ext4\n"},
	} {
		pipeline := newTestAnacondaISOTree()
		pipeline.Kickstart = &kickstart.Options{Path: testKsPath}
		pipeline.InstallRootfsType = tc.fs

		_ = manifest.SerializeWith(pipeline, manifest.Inputs{Containers: []container.Spec{makeFakeContainerPayload()}})

		inlineData := manifest.GetInline(pipeline)
		assert.Len(t, inlineData, 1)

		assert.Contains(t, inlineData[0], tc.expected)
	}
}
