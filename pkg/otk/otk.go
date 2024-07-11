package otk

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/osbuild/images/pkg/blueprint"
)

type otkCustomizationsFile struct {
	file *os.File
}

type defines struct {
	Hostname     string   `yaml:"hostname"`
	KernelAppend string   `yaml:"kernel_append"`
	Languages    []string `yaml:"languages"`
	Keyboard     string   `yaml:"keyboard"`
}

func bpToDefines(bp blueprint.Blueprint) defines {
	d := defines{}

	customizations := bp.Customizations
	if customizations == nil {
		return d
	}

	if customizations.Hostname != nil {
		d.Hostname = *customizations.Hostname
	}

	if customizations.Kernel != nil {
		d.KernelAppend = customizations.Kernel.Append
	}

	if customizations.Locale != nil {
		d.Languages = customizations.Locale.Languages
		if customizations.Locale.Keyboard != nil {
			d.Keyboard = *customizations.Locale.Keyboard
		}
	}

	return d

}

func NewCustomizationsFile(bp blueprint.Blueprint, entrypoint string) (*otkCustomizationsFile, error) {
	tmpfile, err := os.CreateTemp("", "otk-customizations")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary customizations file: %w", err)
	}
	defer tmpfile.Close()

	type cust struct {
		Defines defines `yaml:"otk.define.customizations"`
		Include string  `yaml:"otk.include"`
	}
	absPath, err := filepath.Abs(entrypoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for %q: %w", entrypoint, err)
	}

	c := cust{
		Defines: bpToDefines(bp),
		Include: absPath,
	}

	if err := yaml.NewEncoder(tmpfile).Encode(c); err != nil {
		return nil, fmt.Errorf("failed to write temporary customizations file: %w", err)
	}

	return &otkCustomizationsFile{
		file: tmpfile,
	}, nil
}

func (f *otkCustomizationsFile) Path() string {
	return f.file.Name()
}

func (f *otkCustomizationsFile) Cleanup() error {
	if err := os.Remove(f.file.Name()); err != nil {
		return fmt.Errorf("failed to remove temporary customizations file: %w", err)
	}
	return nil
}
