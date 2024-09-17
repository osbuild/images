package manifest

import (
	"github.com/osbuild/images/pkg/osbuild"
)

var FindStage = findStage

func (p *Tar) Serialize() osbuild.Pipeline {
	return p.serialize()
}
