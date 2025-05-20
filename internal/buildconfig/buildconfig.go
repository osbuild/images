package buildconfig

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/osbuild/blueprint/pkg/blueprint"
	"github.com/osbuild/images/pkg/distro"
)

type BuildConfig struct {
	Name      string               `json:"name"`
	Blueprint *blueprint.Blueprint `json:"blueprint,omitempty"`
	Options   distro.ImageOptions  `json:"options"`
	Depends   interface{}          `json:"depends,omitempty"` // ignored
}

func New(path string) (*BuildConfig, error) {
	fp, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fp.Close()

	dec := json.NewDecoder(fp)
	dec.DisallowUnknownFields()
	var conf BuildConfig

	if err := dec.Decode(&conf); err != nil {
		return nil, err
	}
	if dec.More() {
		return nil, fmt.Errorf("multiple configuration objects or extra data found in %q", path)
	}
	return &conf, nil
}
