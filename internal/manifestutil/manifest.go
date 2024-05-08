package manifestutil

import (
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/distro"
)

type BuildConfig struct {
	Name      string               `json:"name"`
	Blueprint *blueprint.Blueprint `json:"blueprint,omitempty"`
	Options   distro.ImageOptions  `json:"options"`
	Depends   interface{}          `json:"depends,omitempty"` // ignored
}
