package osbuild

import (
	"github.com/osbuild/images/pkg/rpmmd"
)

type DNFMarkStagePackageOptions struct {
	Name  string `json:"name"`
	Mark  string `json:"mark"`
	Group string `json:"group,omitempty"`
}

type DNFMarkStageOptions struct {
	Packages []DNFMarkStagePackageOptions `json:"packages"`
}

func (o DNFMarkStageOptions) isStageOptions() {}

func (o DNFMarkStageOptions) validate() error {
	return nil
}

func NewDNFMarkStageOptions(packages []DNFMarkStagePackageOptions) *DNFMarkStageOptions {
	return &DNFMarkStageOptions{
		Packages: packages,
	}
}

type DNFMarkStage struct {
}

func NewDNFMarkStage(options *DNFMarkStageOptions) *Stage {
	if err := options.validate(); err != nil {
		panic(err)
	}

	return &Stage{
		Type:    "org.osbuild.dnf.mark",
		Options: options,
	}
}

func NewDNFMarkStageFromPackageSpecs(packageSpecs []rpmmd.PackageSpec) *Stage {
	var packages []DNFMarkStagePackageOptions

	for _, ps := range packageSpecs {
		packages = append(packages, DNFMarkStagePackageOptions{
			Name: ps.Name,
			Mark: ps.Reason,
		})
	}

	options := NewDNFMarkStageOptions(packages)

	return NewDNFMarkStage(options)
}
