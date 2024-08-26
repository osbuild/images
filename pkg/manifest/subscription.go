package manifest

import (
	"github.com/osbuild/images/pkg/customizations/fsnode"
	"github.com/osbuild/images/pkg/customizations/subscription"
	"github.com/osbuild/images/pkg/osbuild"
)

type Subscription struct {
	Base

	Subscription *subscription.ImageOptions

	// Custom directories and files to create in the pipeline
	Directories []*fsnode.Directory
	Files       []*fsnode.File
}

// NewSubscription creates a new subscription pipeline for creating files
// required to register a system on first boot.
// The pipeline is intended to be used to create the files necessary for
// registering a system, but outside the OS tree, so they can be copied to
// other locations in the tree after they're created (for example, to an ISO).
func NewSubscription(buildPipeline Build, subOptions *subscription.ImageOptions) *Subscription {
	name := "subscription"
	p := &Subscription{
		Base:         NewBase(name, buildPipeline),
		Subscription: subOptions,
	}
	buildPipeline.addDependent(p)
	return p
}

func (p *Subscription) serialize() osbuild.Pipeline {
	pipeline := p.Base.serialize()
	if p.Subscription != nil {
		serviceDir, err := fsnode.NewDirectory("/etc/systemd/system", nil, nil, nil, true)
		if err != nil {
			panic(err)
		}
		p.Directories = append(p.Directories, serviceDir)

		subStage, subDirs, subFiles, _, err := subscriptionService(*p.Subscription, &subscriptionServiceOptions{InsightsOnBoot: true, UnitPath: osbuild.EtcUnitPath})
		if err != nil {
			panic(err)
		}
		p.Directories = append(p.Directories, subDirs...)
		p.Files = append(p.Files, subFiles...)

		pipeline.AddStages(osbuild.GenDirectoryNodesStages(p.Directories)...)
		pipeline.AddStages(osbuild.GenFileNodesStages(p.Files)...)
		pipeline.AddStage(subStage)
	}
	return pipeline
}

func (p *Subscription) getInline() []string {
	inlineData := []string{}

	// inline data for custom files
	for _, file := range p.Files {
		inlineData = append(inlineData, string(file.Data()))
	}

	return inlineData
}
