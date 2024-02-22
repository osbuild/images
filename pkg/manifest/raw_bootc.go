package manifest

import (
	"fmt"

	"github.com/osbuild/images/pkg/artifact"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/ostree"
	"github.com/osbuild/images/pkg/platform"
	"github.com/osbuild/images/pkg/rpmmd"
)

// A RawBootcImage represents a raw bootc image file which can be booted in a
// hypervisor.
type RawBootcImage struct {
	Base

	filename string
	platform platform.Platform

	containers     []container.SourceSpec
	containerSpecs []container.Spec

	// customizations go here because there is no intermediate
	// tree, with `bootc install to-filesystem` we can only work
	// with the image itself
	PartitionTable *disk.PartitionTable
}

func (p RawBootcImage) Filename() string {
	return p.filename
}

func (p *RawBootcImage) SetFilename(filename string) {
	p.filename = filename
}

func NewRawBootcImage(buildPipeline Build, containers []container.SourceSpec, platform platform.Platform) *RawBootcImage {
	p := &RawBootcImage{
		Base:     NewBase("image", buildPipeline),
		filename: "disk.img",
		platform: platform,

		containers: containers,
	}
	buildPipeline.addDependent(p)
	return p
}

func (p *RawBootcImage) getContainerSources() []container.SourceSpec {
	return p.containers
}

func (p *RawBootcImage) getContainerSpecs() []container.Spec {
	return p.containerSpecs
}

func (p *RawBootcImage) serializeStart(_ []rpmmd.PackageSpec, containerSpecs []container.Spec, _ []ostree.CommitSpec) {
	if len(p.containerSpecs) > 0 {
		panic("double call to serializeStart()")
	}
	p.containerSpecs = containerSpecs
}

func (p *RawBootcImage) serializeEnd() {
	if len(p.containerSpecs) == 0 {
		panic("serializeEnd() call when serialization not in progress")
	}
	p.containerSpecs = nil
}

func (p *RawBootcImage) serialize() osbuild.Pipeline {
	pipeline := p.Base.serialize()

	pt := p.PartitionTable
	if pt == nil {
		panic("no partition table in live image")
	}

	for _, stage := range osbuild.GenImagePrepareStages(pt, p.filename, osbuild.PTSfdisk) {
		pipeline.AddStage(stage)
	}

	if len(p.containerSpecs) != 1 {
		panic(fmt.Sprintf("expected a single container input got %v", p.containerSpecs))
	}
	inputs := osbuild.ContainerDeployInputs{
		Images: osbuild.NewContainersInputForSingleSource(p.containerSpecs[0]),
	}
	devices, mounts, err := osbuild.GenBootupdDevicesMounts(p.filename, p.PartitionTable)
	if err != nil {
		panic(err)
	}
	st, err := osbuild.NewBootcInstallToFilesystemStage(inputs, devices, mounts)
	if err != nil {
		panic(err)
	}
	pipeline.AddStage(st)

	// XXX: there is no way right now to support any customizations,
	// we cannot touch the filesystem after bootc installed it or
	// we risk messing with it's selinux labels or future fsverity
	// magic.  Once we have a mechanism like --copy-etc from
	// https://github.com/containers/bootc/pull/267 things should
	// be a bit better

	for _, stage := range osbuild.GenImageFinishStages(pt, p.filename) {
		pipeline.AddStage(stage)
	}

	return pipeline
}

// XXX: copied from raw.go
func (p *RawBootcImage) Export() *artifact.Artifact {
	p.Base.export = true
	return artifact.New(p.Name(), p.Filename(), nil)
}
