package manifest

import "github.com/osbuild/images/pkg/osbuild"

func (it *ISOTree) Serialize() (osbuild.Pipeline, error) {
	return it.serialize()
}
