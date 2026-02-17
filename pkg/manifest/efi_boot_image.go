package manifest

import (
	"fmt"

	"github.com/osbuild/images/pkg/artifact"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/osbuild"
)

type EFIBootImage struct {
	Base

	PartitionTable *disk.PartitionTable

	anacondaPipeline    *AnacondaInstaller
	efiBootTreePipeline *EFIBootTree

	filename string
}

func NewEFIBootImage(buildPipeline Build, efiBootTreePipeline *EFIBootTree, anacondaPipeline *AnacondaInstaller) *EFIBootImage {
	p := &EFIBootImage{
		Base:                NewBase("efiboot-image", buildPipeline),
		efiBootTreePipeline: efiBootTreePipeline,
		anacondaPipeline:    anacondaPipeline,
		filename:            "efiboot.img",
	}
	buildPipeline.addDependent(p)
	return p
}

func (p EFIBootImage) Filename() string {
	return p.filename
}

func (p *EFIBootImage) SetFilename(filename string) {
	p.filename = filename
}

func (p *EFIBootImage) Export() *artifact.Artifact {
	p.Base.export = true
	return artifact.New(p.Name(), p.Filename(), nil)
}

func (p *EFIBootImage) serialize() (osbuild.Pipeline, error) {
	pipeline, err := p.Base.serialize()
	if err != nil {
		return osbuild.Pipeline{}, err
	}

	pipeline.AddStage(osbuild.NewTruncateStage(&osbuild.TruncateStageOptions{
		Filename: p.filename,
		Size:     fmt.Sprintf("%d", p.PartitionTable.Size),
	}))

	for _, stage := range osbuild.GenFsStages(p.PartitionTable, p.filename, p.anacondaPipeline.Name()) {
		pipeline.AddStage(stage)
	}

	inputName := "efi-boot-tree"
	copyInputs := osbuild.NewPipelineTreeInputs(inputName, p.efiBootTreePipeline.Name())
	copyOptions, copyDevices, copyMounts := osbuild.GenCopyFSTreeOptions(inputName, p.efiBootTreePipeline.Name(), p.filename, p.PartitionTable)
	pipeline.AddStage(osbuild.NewCopyStage(copyOptions, copyInputs, copyDevices, copyMounts))

	copyInputs = osbuild.NewPipelineTreeInputs(inputName, p.efiBootTreePipeline.Name())
	pipeline.AddStage(osbuild.NewCopyStageSimple(
		&osbuild.CopyStageOptions{
			Paths: []osbuild.CopyStagePath{
				{
					From: fmt.Sprintf("input://%s/EFI", inputName),
					To:   "tree:///",
				},
			},
		},
		copyInputs,
	))

	return pipeline, nil
}
