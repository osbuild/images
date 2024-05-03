package manifest

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"testing"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/customizations/kickstart"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/ostree"
	"github.com/osbuild/images/pkg/platform"
	"github.com/osbuild/images/pkg/runner"
	"github.com/stretchr/testify/assert"
)

const (
	testKsPath     = "/test.ks"
	testBaseKsPath = "/test-base.ks"
)

// newTestAnacondaISOTree returns a base AnacondaInstallerISOTree pipeline.
func newTestAnacondaISOTree() *AnacondaInstallerISOTree {
	m := &Manifest{}
	runner := &runner.Linux{}
	build := NewBuild(m, runner, nil, nil)

	x86plat := &platform.X86{}

	product := ""
	osversion := ""

	preview := false

	anacondaPipeline := NewAnacondaInstaller(
		AnacondaInstallerTypePayload,
		build,
		x86plat,
		nil,
		"kernel",
		product,
		osversion,
		preview,
	)
	rootfsImagePipeline := NewISORootfsImg(build, anacondaPipeline)
	bootTreePipeline := NewEFIBootTree(build, product, osversion)

	pipeline := NewAnacondaInstallerISOTree(build, anacondaPipeline, rootfsImagePipeline, bootTreePipeline)
	// copy of the default in pkg/image - will be moved to the pipeline
	var efibootImageSize uint64 = 20 * common.MebiByte
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
	ksSudoContent = `%post
echo -e "%sudo\tALL=(ALL)\tNOPASSWD: ALL" > "/etc/sudoers.d/%sudo"
chmod 0440 /etc/sudoers.d/%sudo
echo -e "%wheel\tALL=(ALL)\tNOPASSWD: ALL" > "/etc/sudoers.d/%wheel"
chmod 0440 /etc/sudoers.d/%wheel
restorecon -rvF /etc/sudoers.d
%end
`
	ksContainerContent = `reqpart --add-boot

part swap --fstype=swap --size=1024
part / --fstype=ext4 --grow

reboot --eject
%post
bootc switch --mutate-in-place --transport registry local.example.org/registry/org/image
%end
`
)

func calculateInlineFileChecksum(parts ...string) string {
	content := "%include /run/install/repo/test-base.ks\n"
	for _, part := range parts {
		content += part
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(content)))
}

func checkKickstartOptions(stages []*osbuild.Stage, unattended, sudobits bool, extra string) error {
	ksParts := make([]string, 0)
	if sudobits {
		ksParts = append(ksParts, "\n"+ksSudoContent)
	}
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
		func() { pipeline.serializeStart(nil, nil, make([]ostree.CommitSpec, 2), nil) },
	)
	assert.PanicsWithValue(
		"pipeline supports at most one container",
		func() { pipeline.serializeStart(nil, make([]container.Spec, 2), nil, nil) },
	)
}

func TestAnacondaISOTreeSerializeWithOS(t *testing.T) {
	osPayload := NewTestOS()

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
		pipeline.serializeStart(nil, nil, nil, nil)
		sp := pipeline.serialize()
		pipeline.serializeEnd()
		assert.NoError(t, checkISOTreeStages(sp.Stages, payloadStages,
			append(variantStages, []string{"org.osbuild.kickstart", "org.osbuild.isolinux"}...)))
	})

	// the os payload variant of the pipeline only adds the kickstart file if
	// KSPath is defined
	t.Run("kspath", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.OSPipeline = osPayload
		pipeline.Kickstart = &kickstart.Options{Path: testKsPath}
		pipeline.serializeStart(nil, nil, nil, nil)
		sp := pipeline.serialize()
		pipeline.serializeEnd()
		assert.NoError(t, checkISOTreeStages(sp.Stages, append(payloadStages, "org.osbuild.kickstart"),
			append(variantStages, "org.osbuild.isolinux")))
	})

	// enable ISOLinux and check for stage
	t.Run("kspath+isolinux", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.OSPipeline = osPayload
		pipeline.Kickstart = &kickstart.Options{Path: testKsPath}
		pipeline.ISOLinux = true
		pipeline.serializeStart(nil, nil, nil, nil)
		sp := pipeline.serialize()
		pipeline.serializeEnd()
		assert.NoError(t, checkISOTreeStages(sp.Stages, append(payloadStages, "org.osbuild.isolinux", "org.osbuild.kickstart"),
			variantStages))
	})

	t.Run("unattended", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.OSPipeline = osPayload
		pipeline.Kickstart = &kickstart.Options{Path: testKsPath, Unattended: true}
		pipeline.ISOLinux = true
		pipeline.serializeStart(nil, nil, nil, nil)
		sp := pipeline.serialize()
		pipeline.serializeEnd()
		assert.NoError(t, checkISOTreeStages(sp.Stages, append(payloadStages, "org.osbuild.isolinux", "org.osbuild.kickstart"),
			variantStages))
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
		pipeline.ISOLinux = true
		pipeline.serializeStart(nil, nil, nil, nil)
		sp := pipeline.serialize()
		pipeline.serializeEnd()
		assert.NoError(t, checkISOTreeStages(sp.Stages, append(payloadStages, "org.osbuild.isolinux", "org.osbuild.kickstart"),
			variantStages))
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
		pipeline.ISOLinux = true
		pipeline.serializeStart(nil, nil, nil, nil)
		sp := pipeline.serialize()
		pipeline.serializeEnd()
		assert.NoError(t, checkISOTreeStages(sp.Stages, append(payloadStages, "org.osbuild.isolinux", "org.osbuild.kickstart"),
			variantStages))
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
		pipeline.ISOLinux = true
		pipeline.serializeStart(nil, nil, nil, nil)
		assert.Panics(t, func() { pipeline.serialize() })
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
		pipeline.ISOLinux = true
		pipeline.serializeStart(nil, nil, nil, nil)
		assert.Panics(t, func() { pipeline.serialize() })
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
		pipeline.serializeStart(nil, nil, []ostree.CommitSpec{ostreeCommit}, nil)
		sp := pipeline.serialize()
		pipeline.serializeEnd()
		assert.NoError(t, checkISOTreeStages(sp.Stages, payloadStages,
			append(variantStages, "org.osbuild.isolinux")))
	})

	// enable ISOLinux and check for stage
	t.Run("isolinux", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.Kickstart = &kickstart.Options{Path: testKsPath, OSTree: &kickstart.OSTree{}}
		pipeline.ISOLinux = true
		pipeline.serializeStart(nil, nil, []ostree.CommitSpec{ostreeCommit}, nil)
		sp := pipeline.serialize()
		pipeline.serializeEnd()
		assert.NoError(t, checkISOTreeStages(sp.Stages, append(payloadStages, "org.osbuild.isolinux"), variantStages))
	})

	t.Run("unattended", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.Kickstart = &kickstart.Options{Path: testKsPath, Unattended: true, OSTree: &kickstart.OSTree{}}
		pipeline.ISOLinux = true
		pipeline.serializeStart(nil, nil, []ostree.CommitSpec{ostreeCommit}, nil)
		sp := pipeline.serialize()
		pipeline.serializeEnd()
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
		pipeline.ISOLinux = true
		pipeline.serializeStart(nil, nil, []ostree.CommitSpec{ostreeCommit}, nil)
		sp := pipeline.serialize()
		pipeline.serializeEnd()
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
		pipeline.ISOLinux = true
		pipeline.serializeStart(nil, nil, []ostree.CommitSpec{ostreeCommit}, nil)
		sp := pipeline.serialize()
		pipeline.serializeEnd()
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
		pipeline.ISOLinux = true
		pipeline.serializeStart(nil, nil, []ostree.CommitSpec{ostreeCommit}, nil)
		assert.Panics(t, func() { pipeline.serialize() })
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
		pipeline.ISOLinux = true
		pipeline.serializeStart(nil, nil, []ostree.CommitSpec{ostreeCommit}, nil)
		assert.Panics(t, func() { pipeline.serialize() })
	})
}

func TestAnacondaISOTreeSerializeWithContainer(t *testing.T) {

	containerPayload := container.Spec{
		Source:    "example.org/registry/org/image",
		Digest:    "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		ImageID:   "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		LocalName: "local.example.org/registry/org/image",
	}
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
		pipeline.serializeStart(nil, []container.Spec{containerPayload}, nil, nil)
		sp := pipeline.serialize()
		pipeline.serializeEnd()
		assert.NoError(t, checkISOTreeStages(sp.Stages, payloadStages,
			append(variantStages, "org.osbuild.isolinux")))
	})

	// enable ISOLinux and check again
	t.Run("isolinux", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.Kickstart = &kickstart.Options{Path: testKsPath}
		pipeline.ISOLinux = true
		pipeline.serializeStart(nil, []container.Spec{containerPayload}, nil, nil)
		sp := pipeline.serialize()
		pipeline.serializeEnd()
		assert.NoError(t, checkISOTreeStages(sp.Stages, append(payloadStages, "org.osbuild.isolinux"), variantStages))
	})

	t.Run("kernel-options", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.Kickstart = &kickstart.Options{
			Path:                testKsPath,
			Unattended:          true,
			KernelOptionsAppend: []string{"kernel.opt=1", "debug"},
		}
		pipeline.serializeStart(nil, []container.Spec{containerPayload}, nil, nil)
		sp := pipeline.serialize()
		pipeline.serializeEnd()
		kickstartSt := findStage("org.osbuild.kickstart", sp.Stages)
		assert.NotNil(t, kickstartSt)
		opts := kickstartSt.Options.(*osbuild.KickstartStageOptions)
		assert.Equal(t, "kernel.opt=1 debug", opts.Bootloader.Append)
	})

	t.Run("network-on-boot", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.Kickstart = &kickstart.Options{Path: testKsPath, NetworkOnBoot: true}
		pipeline.serializeStart(nil, []container.Spec{containerPayload}, nil, nil)
		sp := pipeline.serialize()
		pipeline.serializeEnd()
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
		pipeline.ISOLinux = true
		pipeline.serializeStart(nil, []container.Spec{containerPayload}, nil, nil)
		sp := pipeline.serialize()
		pipeline.serializeEnd()
		assert.NoError(t, checkISOTreeStages(sp.Stages, append(payloadStages, "org.osbuild.isolinux"), variantStages))
		assert.NoError(t, checkRawKickstartForContainer(sp.Stages, userks))
	})
	t.Run("remove-payload-signtures", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.Kickstart = &kickstart.Options{Path: testKsPath}
		pipeline.PayloadRemoveSignatures = true
		pipeline.serializeStart(nil, []container.Spec{containerPayload}, nil, nil)
		sp := pipeline.serialize()
		pipeline.serializeEnd()
		skopeoStage := findStage("org.osbuild.skopeo", sp.Stages)
		assert.NotNil(t, skopeoStage)
		assert.Equal(t, skopeoStage.Options.(*osbuild.SkopeoStageOptions).RemoveSignatures, common.ToPtr(true))
	})
}

func TestMakeKickstartSudoersPostEmpty(t *testing.T) {
	assert.Equal(t, "", makeKickstartSudoersPost(nil))
}

func TestMakeKickstartSudoersPost(t *testing.T) {
	exp := `
%post
echo -e "%group31\tALL=(ALL)\tNOPASSWD: ALL" > "/etc/sudoers.d/%group31"
chmod 0440 /etc/sudoers.d/%group31
echo -e "user42\tALL=(ALL)\tNOPASSWD: ALL" > "/etc/sudoers.d/user42"
chmod 0440 /etc/sudoers.d/user42
restorecon -rvF /etc/sudoers.d
%end
`
	assert.Equal(t, exp, makeKickstartSudoersPost([]string{"user42", "%group31"}))
	assert.Equal(t, exp, makeKickstartSudoersPost([]string{"%group31", "user42"}))
	assert.Equal(t, exp, makeKickstartSudoersPost([]string{"%group31", "user42", "%group31"}))
	assert.Equal(t, exp, makeKickstartSudoersPost([]string{"%group31", "user42", "%group31", "%group31", "user42", "%group31", "%group31", "user42", "%group31"}))
}
