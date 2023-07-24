package osbuild

import (
	"github.com/osbuild/images/pkg/rpmmd"
)

type DNF4MarkStagePackageOptions struct {
	Name string `json:"name"`
	Mark string `json:"mark"`
}

type DNF4MarkStageOptions struct {
	Packages []DNF4MarkStagePackageOptions `json:"packages"`
}

func (o DNF4MarkStageOptions) isStageOptions() {}

func (o DNF4MarkStageOptions) validate() error {
	return nil
}

func NewDNF4MarkStageOptions(packages []DNF4MarkStagePackageOptions) *DNF4MarkStageOptions {
	return &DNF4MarkStageOptions{
		Packages: packages,
	}
}

func NewDNF4MarkStage(options *DNF4MarkStageOptions) *Stage {
	if err := options.validate(); err != nil {
		panic(err)
	}

	return &Stage{
		Type:    "org.osbuild.dnf4.mark",
		Options: options,
	}
}

func NewDNF4MarkStageFromPackageSpecs(packageSpecs []rpmmd.PackageSpec) *Stage {
	var packages []DNF4MarkStagePackageOptions

	for _, ps := range packageSpecs {
		var reason string

		// For dnf4 the CLI interface which is used by the stage only accepts
		// `install`, `group`, and `remove`. Filter out and translate.
		if ps.Reason == "user" {
			reason = "install"
		} else if ps.Reason == "group" {
			reason = "group"
		} else {
			continue
		}

		packages = append(packages, DNF4MarkStagePackageOptions{
			Name: ps.Name,
			Mark: reason,
		})
	}

	options := NewDNF4MarkStageOptions(packages)

	return NewDNF4MarkStage(options)
}
