package osbuild

import (
	"encoding/json"

	"github.com/osbuild/images/pkg/arch"
)

// treeinfoStageOptions represents the options of the treeinfo stage.
// All paths are relative to .treeinfo path specified via Path field.
// See https://github.com/osbuild/osbuild/blob/v165/stages/org.osbuild.treeinfo.meta.json
type treeinfoStageOptions struct {
	Path     string            `json:"path"` // path to the treeinfo file, must be set to ".treeinfo"
	Treeinfo TreeinfoInputData `json:"treeinfo"`
}

// TreeinfoInputData represents the data stored in the .treeinfo file.
type TreeinfoInputData struct {
	Release   *TreeinfoReleaseStageOptions `json:"release,omitempty"`
	Tree      *TreeinfoTreeStageOptions    `json:"tree,omitempty"`
	Checksums []string                     `json:"checksums,omitempty"` // filenames of files in a tree to calculate shasum for
	Stage2    *TreeinfoStage2StageOptions  `json:"stage2,omitempty"`    // stage2 information (only for Anaconda)

	Images   map[string]map[string]string           `json:"-"` // images compatible with particular $platform - serialized as "images-<arch>"
	Variants map[string]TreeinfoVariantStageOptions `json:"-"` // variants option - serialized as "variant-<uid>"

	General *TreeinfoGeneralStageOptions `json:"general,omitempty"` // generated automatically for backwards compatibility
}

// TreeinfoGeneralStageOptions is a generated section for backwards compatibility.
type TreeinfoGeneralStageOptions struct {
	Arch       string   `json:"arch,omitempty"`
	Family     string   `json:"family,omitempty"`
	Name       string   `json:"name,omitempty"`
	Packagedir string   `json:"packagedir,omitempty"`
	Platforms  []string `json:"platforms,omitempty"`
	Repository string   `json:"repository,omitempty"`
	Variant    string   `json:"variant,omitempty"`
	Version    string   `json:"version,omitempty"`
}

type TreeinfoReleaseStageOptions struct {
	Name    string `json:"name,omitempty"`    // release name, for example: "Fedora", "Red Hat Enterprise Linux", "Spacewalk"
	Short   string `json:"short,omitempty"`   // release short name, for example: "F", "RHEL", "Spacewalk"
	Version string `json:"version,omitempty"` // release version, for example: "21", "7.0", "2.1"
}

type TreeinfoTreeStageOptions struct {
	Arch      string   `json:"arch,omitempty"`      // tree architecture, for example x86_64
	Platforms []string `json:"platforms,omitempty"` // supported platforms; for example x86_64,xen
	Variants  []string `json:"variants,omitempty"`  // UIDs of available variants, for example "Server,Workstation"
}

type TreeinfoStage2StageOptions struct {
	Mainimage string `json:"mainimage,omitempty"` // Anaconda stage2 main image file path
}

type TreeinfoVariantStageOptions struct {
	ID         string `json:"id,omitempty"`
	Name       string `json:"name,omitempty"`
	Packages   string `json:"packages,omitempty"`
	Repository string `json:"repository,omitempty"`
	Type       string `json:"type,omitempty"`
	UID        string `json:"uid,omitempty"`
}

// MarshalJSON populates the `images-*` and `variant-*` fields of the treeinfo structure automatically,
// see https://github.com/osbuild/osbuild/blob/v165/stages/org.osbuild.treeinfo.meta.json#L115
// We do it here because this is hard to represent in go otherwise.
func (t TreeinfoInputData) MarshalJSON() ([]byte, error) {
	type TreeinfoDataAlias TreeinfoInputData
	b, err := json.Marshal(TreeinfoDataAlias(t))
	if err != nil {
		return nil, err
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}

	for k, v := range t.Images {
		m[k] = v
	}
	for k, v := range t.Variants {
		m[k] = v
	}

	return json.Marshal(m)
}

func (treeinfoStageOptions) isStageOptions() {}

func NewTreeinfoStage(name, version string, architecture arch.Arch, variant string, kernel, initrd, stage2 string) *Stage {
	images := make(map[string]map[string]string)
	images["images-"+architecture.String()] = map[string]string{
		"kernel": kernel,
		"initrd": initrd,
	}

	checksums := []string{
		kernel,
		initrd,
		stage2,
	}

	treeinfo := &treeinfoStageOptions{
		Path: ".treeinfo",
		Treeinfo: TreeinfoInputData{
			// XXX: also set ShortName (RHEL/F) but it needs to be added to distrodef first
			Release: &TreeinfoReleaseStageOptions{
				Name:    name,
				Version: version,
			},
			Tree: &TreeinfoTreeStageOptions{
				Arch:      architecture.String(),
				Platforms: []string{architecture.String()},
			},
			Stage2: &TreeinfoStage2StageOptions{
				Mainimage: stage2,
			},
			Checksums: checksums,
			Images:    images,
		},
	}

	treeinfo.Treeinfo.General = &TreeinfoGeneralStageOptions{
		Arch:      treeinfo.Treeinfo.Tree.Arch,
		Family:    treeinfo.Treeinfo.Release.Name,
		Name:      treeinfo.Treeinfo.Release.Name + " " + treeinfo.Treeinfo.Release.Version,
		Platforms: treeinfo.Treeinfo.Tree.Platforms,
		Version:   treeinfo.Treeinfo.Release.Version,
	}

	if variant != "" {
		treeinfo.Treeinfo.Tree.Variants = []string{variant}
		treeinfo.Treeinfo.Variants = map[string]TreeinfoVariantStageOptions{
			"variant-" + variant: {
				Type: "variant",
				ID:   variant,
				UID:  variant,
				Name: variant,
			},
		}
		treeinfo.Treeinfo.General.Variant = variant
	}

	return &Stage{
		Type:    "org.osbuild.treeinfo",
		Options: treeinfo,
	}
}
