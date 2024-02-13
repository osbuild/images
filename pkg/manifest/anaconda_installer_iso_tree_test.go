package manifest

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/ostree"
	"github.com/osbuild/images/pkg/platform"
	"github.com/osbuild/images/pkg/runner"
	"github.com/stretchr/testify/assert"
)

// newTestAnacondaISOTree returns a base AnacondaInstallerISOTree pipeline.
func newTestAnacondaISOTree() *AnacondaInstallerISOTree {
	m := &Manifest{}
	runner := &runner.Linux{}
	build := NewBuild(m, runner, nil, nil)

	x86plat := &platform.X86{}

	product := ""
	osversion := ""

	anacondaPipeline := NewAnacondaInstaller(
		AnacondaInstallerTypePayload,
		build,
		x86plat,
		nil,
		"kernel",
		product,
		osversion,
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

func TestAnacondaISOTreePayloadsBad(t *testing.T) {
	assert := assert.New(t)
	pipeline := newTestAnacondaISOTree()

	assert.PanicsWithValue(
		"pipeline supports at most one ostree commit",
		func() { pipeline.serializeStart(nil, nil, make([]ostree.CommitSpec, 2)) },
	)
	assert.PanicsWithValue(
		"pipeline supports at most one container",
		func() { pipeline.serializeStart(nil, make([]container.Spec, 2), nil) },
	)
}

func TestAnacondaISOTreeSerializeWithOS(t *testing.T) {
	osPayload := NewTestOS()

	payloadStages := []string{"org.osbuild.tar"}

	t.Run("plain", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.OSPipeline = osPayload
		pipeline.serializeStart(nil, nil, nil)
		sp := pipeline.serialize()
		pipeline.serializeEnd()
		assert.NoError(t, checkISOTreeStages(sp.Stages, payloadStages, []string{"org.osbuild.kickstart", "org.osbuild.isolinux"}))
	})

	// the os payload variant of the pipeline only adds the kickstart file if
	// KSPath is defined
	t.Run("kspath", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.OSPipeline = osPayload
		pipeline.KSPath = "/test.ks"
		pipeline.serializeStart(nil, nil, nil)
		sp := pipeline.serialize()
		pipeline.serializeEnd()
		assert.NoError(t, checkISOTreeStages(sp.Stages, append(payloadStages, "org.osbuild.kickstart"), []string{"org.osbuild.isolinux"}))
	})

	// enable ISOLinux and check for stage
	t.Run("kspath+isolinux", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.OSPipeline = osPayload
		pipeline.KSPath = "/test.ks"
		pipeline.ISOLinux = true
		pipeline.serializeStart(nil, nil, nil)
		sp := pipeline.serialize()
		pipeline.serializeEnd()
		assert.NoError(t, checkISOTreeStages(sp.Stages, append(payloadStages, "org.osbuild.isolinux", "org.osbuild.kickstart"), nil))
	})
}

func TestAnacondaISOTreeSerializeWithOSTree(t *testing.T) {
	ostreeCommit := ostree.CommitSpec{
		Ref:      "test/99/ostree",
		URL:      "http://example.com/ostree/repo",
		Checksum: "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	}
	payloadStages := []string{
		"org.osbuild.ostree.init",
		"org.osbuild.ostree.pull",
		"org.osbuild.kickstart",
	}

	t.Run("plain", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.serializeStart(nil, nil, []ostree.CommitSpec{ostreeCommit})
		sp := pipeline.serialize()
		pipeline.serializeEnd()
		assert.NoError(t, checkISOTreeStages(sp.Stages, payloadStages, []string{"org.osbuild.isolinux"}))
	})

	// enable ISOLinux and check for stage
	t.Run("isolinux", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.ISOLinux = true
		pipeline.serializeStart(nil, nil, []ostree.CommitSpec{ostreeCommit})
		sp := pipeline.serialize()
		pipeline.serializeEnd()
		assert.NoError(t, checkISOTreeStages(sp.Stages, append(payloadStages, "org.osbuild.isolinux"), nil))
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

	t.Run("kspath", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.KSPath = "/test.ks"
		pipeline.serializeStart(nil, []container.Spec{containerPayload}, nil)
		sp := pipeline.serialize()
		pipeline.serializeEnd()
		assert.NoError(t, checkISOTreeStages(sp.Stages, payloadStages, []string{"org.osbuild.isolinux"}))
	})

	// enable ISOLinux and check again
	t.Run("isolinux", func(t *testing.T) {
		pipeline := newTestAnacondaISOTree()
		pipeline.KSPath = "/test.ks"
		pipeline.ISOLinux = true
		pipeline.serializeStart(nil, []container.Spec{containerPayload}, nil)
		sp := pipeline.serialize()
		pipeline.serializeEnd()
		assert.NoError(t, checkISOTreeStages(sp.Stages, append(payloadStages, "org.osbuild.isolinux"), nil))
	})
}
