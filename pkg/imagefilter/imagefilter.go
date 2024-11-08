package imagefilter

import (
	"fmt"

	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distrofactory"
	"github.com/osbuild/images/pkg/distrosort"
)

type DistroLister interface {
	ListDistros() []string
}

// Result contains a result from a imagefilter.Filter run
type Result struct {
	Distro  distro.Distro
	Arch    distro.Arch
	ImgType distro.ImageType
}

// ImageFilter is an a flexible way to filter the available images.
type ImageFilter struct {
	fac   *distrofactory.Factory
	repos DistroLister
}

// New creates a new ImageFilter that can be used to filter the list
// of available images
func New(fac *distrofactory.Factory, repos DistroLister) (*ImageFilter, error) {
	if fac == nil {
		return nil, fmt.Errorf("cannot create ImageFilter without a valid distrofactory")
	}
	if repos == nil {
		return nil, fmt.Errorf("cannot create ImageFilter without a valid reporegistry")
	}

	return &ImageFilter{fac: fac, repos: repos}, nil
}

// Filter filters the available images for the given
// distrofactory/reporegistry based on the given filter terms. Glob
// like patterns (?, *) are supported, see fnmatch(3).
//
// Without a prefix in the filter term a simple name filtering is performed.
// With a prefix the specified property is filtered, e.g. "arch:i386". Adding
// filtering will narrow down the filtering (terms are combined via AND).
//
// The following prefixes are supported:
// "distro:" - the distro name, e.g. rhel-9, or fedora*
// "arch:" - the architecture, e.g. x86_64
// "type": - the image type, e.g. ami, or qcow?
// "bootmode": - the bootmode, e.g. "legacy", "uefi", "hybrid"
func (i *ImageFilter) Filter(searchTerms ...string) ([]Result, error) {
	var res []Result

	distroNames := i.repos.ListDistros()
	filter, err := newFilter(searchTerms...)
	if err != nil {
		return nil, err
	}

	if err := distrosort.Names(distroNames); err != nil {
		return nil, err
	}
	for _, distroName := range distroNames {
		distro := i.fac.GetDistro(distroName)
		if distro == nil {
			// XXX: log here?
			continue
		}
		for _, archName := range distro.ListArches() {
			a, err := distro.GetArch(archName)
			if err != nil {
				return nil, err
			}
			for _, imgTypeName := range a.ListImageTypes() {
				imgType, err := a.GetImageType(imgTypeName)
				if err != nil {
					return nil, err
				}
				if filter.Matches(distro, a, imgType) {
					res = append(res, Result{distro, a, imgType})
				}
			}
		}
	}

	return res, nil
}
