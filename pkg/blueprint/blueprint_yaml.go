package blueprint

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type blueprintOnDisk struct {
	Name           string          `json:"name" toml:"name"`
	Description    string          `json:"description" toml:"description"`
	Version        string          `json:"version,omitempty" toml:"version,omitempty"`
	Packages       []Package       `json:"packages" toml:"packages"`
	Modules        []Package       `json:"modules" toml:"modules"`
	Groups         []Group         `json:"groups" toml:"groups"`
	Containers     []Container     `json:"containers,omitempty" toml:"containers,omitempty"`
	Customizations *Customizations `json:"customizations,omitempty" toml:"customizations"`
	Distro         string          `json:"distro" toml:"distro"`

	// EXPERIMENTAL
	Minimal bool `json:"minimal" toml:"minimal"`
}

func Parse(path string) (*Blueprint, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	switch ext := filepath.Ext(path); ext {
	case ".json":
		return parseJSONFromReader(f, path)
	default:
		return nil, fmt.Errorf("unsupported file format %q", ext)
	}
}

func parseJSONFromReader(r io.Reader, what string) (*Blueprint, error) {
	var bpod blueprintOnDisk

	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&bpod); err != nil {
		return nil, err
	}
	if dec.More() {
		return nil, fmt.Errorf("cannot support multiple blueprints from %q", what)
	}

	return bpFromBpod(&bpod)
}

func bpFromBpod(bpod *blueprintOnDisk) (*Blueprint, error) {
	var bp Blueprint

	bp.Name = bpod.Name
	bp.Description = bpod.Description
	bp.Version = bpod.Version
	bp.Packages = bpod.Packages
	bp.Modules = bpod.Modules
	bp.Groups = bpod.Groups
	bp.Containers = bpod.Containers
	bp.Customizations = bpod.Customizations
	bp.Distro = bpod.Distro
	bp.Minimal = bpod.Minimal

	return &bp, nil
}
