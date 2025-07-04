package manifest

import (
	"fmt"

	"github.com/osbuild/images/pkg/artifact"
	"github.com/osbuild/images/pkg/osbuild"
)

type Vagrant struct {
	Base
	filename string
	provider osbuild.VagrantProvider

	imgPipeline FilePipeline
}

func (p Vagrant) Filename() string {
	return p.filename
}

func (p *Vagrant) SetFilename(filename string) {
	p.filename = filename
}

func NewVagrant(buildPipeline Build, imgPipeline FilePipeline, provider osbuild.VagrantProvider) *Vagrant {
	p := &Vagrant{
		Base:        NewBase("vagrant", buildPipeline),
		imgPipeline: imgPipeline,
		filename:    "image.box",
		provider:    provider,
	}

	if buildPipeline != nil {
		buildPipeline.addDependent(p)
	} else {
		imgPipeline.Manifest().addPipeline(p)
	}

	return p
}

func (p *Vagrant) serialize() osbuild.Pipeline {
	pipeline := p.Base.serialize()

	// For the VirtualBox provider we need to inject the ovf stage as well
	if p.provider == osbuild.VagrantProviderVirtualbox {
		// The ovf stage needs the vmdk at root, I really don't quite like this
		// and don't see why it can't be through an input?
		inputName := "vmdk-tree"
		pipeline.AddStage(osbuild.NewCopyStageSimple(
			&osbuild.CopyStageOptions{
				Paths: []osbuild.CopyStagePath{
					{
						From: fmt.Sprintf("input://%s/%s", inputName, p.imgPipeline.Export().Filename()),
						To:   "tree:///",
					},
				},
			},
			osbuild.NewPipelineTreeInputs(inputName, p.imgPipeline.Name()),
		))

		pipeline.AddStage(osbuild.NewOVFStage(&osbuild.OVFStageOptions{
			Vmdk: p.imgPipeline.Filename(),
		}))
	}

	pipeline.AddStage(osbuild.NewVagrantStage(
		osbuild.NewVagrantStageOptions(p.provider),
		osbuild.NewVagrantStagePipelineFilesInputs(p.imgPipeline.Name(), p.imgPipeline.Filename()),
	))

	return pipeline
}

func (p *Vagrant) getBuildPackages(Distro) []string {
	return []string{"qemu-img"}
}

func (p *Vagrant) Export() *artifact.Artifact {
	p.Base.export = true
	mimeType := "application/x-qemu-disk"
	return artifact.New(p.Name(), p.Filename(), &mimeType)
}
