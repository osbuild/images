package manifest

import (
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/ostree"
	"github.com/osbuild/images/pkg/rpmmd"
)

func (br *BuildrootFromContainer) Dependents() []Pipeline {
	return br.dependents
}

func (rbc *RawBootcImage) Serialize() osbuild.Pipeline {
	return rbc.serialize()
}

func (rbc *RawBootcImage) SerializeStart(a []rpmmd.PackageSpec, b []container.Spec, c []ostree.CommitSpec) {
	rbc.serializeStart(a, b, c)
}
