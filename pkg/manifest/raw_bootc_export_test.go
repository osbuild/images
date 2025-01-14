package manifest

import (
	"github.com/osbuild/images/pkg/osbuild"
)

func (br *BuildrootFromContainer) Dependents() []Pipeline {
	return br.dependents
}

func (rbc *RawBootcImage) Serialize() osbuild.Pipeline {
	return rbc.serialize()
}

func (rbc *RawBootcImage) SerializeStart(inputs Inputs) {
	rbc.serializeStart(inputs)
}
