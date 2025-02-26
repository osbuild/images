package manifest

import (
	"github.com/osbuild/images/pkg/dnfjson"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/ostree"
	"github.com/osbuild/images/pkg/platform"
	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/osbuild/images/pkg/runner"
)

// findStage is used in both internal and external package tests.
// TODO: make all tests external and define this only in the manifest_test
// package.
func findStage(name string, stages []*osbuild.Stage) *osbuild.Stage {
	for _, s := range stages {
		if s.Type == name {
			return s
		}
	}
	return nil
}

var FindStage = findStage

func (p *Tar) Serialize() osbuild.Pipeline {
	return p.serialize()
}

func (p *Zstd) Serialize() osbuild.Pipeline {
	return p.serialize()
}

func (p *OS) Serialize() osbuild.Pipeline {
	repos := []rpmmd.RepoConfig{}
	packages := []rpmmd.PackageSpec{
		{Name: "pkg1", Checksum: "sha1:c02524e2bd19490f2a7167958f792262754c5f46"},
	}
	p.serializeStart(Inputs{
		Depsolved: dnfjson.DepsolveResult{
			Packages: packages,
			Repos:    repos,
		},
	})
	return p.serialize()
}

func (p *OS) SerializeWith(inputs Inputs) osbuild.Pipeline {
	p.serializeStart(inputs)
	return p.serialize()
}

func (p *OS) GetBuildPackages(d Distro) []string {
	return p.getBuildPackages(d)
}

func (p *OS) GetPackageSetChain(d Distro) []rpmmd.PackageSet {
	return p.getPackageSetChain(d)
}

// NewTestOS is used in both internal and external package tests.
// TODO: make all tests external and define this only in the manifest_test
// package.
func NewTestOS() *OS {
	repos := []rpmmd.RepoConfig{}
	m := New()
	runner := &runner.Fedora{Version: 38}
	build := NewBuild(&m, runner, repos, nil)
	build.Checkpoint()

	// create an x86_64 platform with bios boot
	platform := &platform.X86{
		BIOS: true,
	}

	os := NewOS(build, platform, repos)

	return os
}

func (p *OSTreeDeployment) Serialize() osbuild.Pipeline {
	p.serializeStart(Inputs{
		Commits: []ostree.CommitSpec{{}},
	})
	return p.serialize()
}
