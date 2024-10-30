package main

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distrofactory"
	"github.com/osbuild/images/pkg/distrosort"
)

// XXX: move this file into distro factory?

type FilterResult struct {
	Distro  distro.Distro
	Arch    distro.Arch
	ImgType distro.ImageType
}

func getOneImage(distroName, imgTypeStr, archStr string) (*FilterResult, error) {
	filterExprs := []string{
		fmt.Sprintf("name:%s", distroName),
		fmt.Sprintf("arch:%s", archStr),
		fmt.Sprintf("type:%s", imgTypeStr),
	}
	filteredResults, err := getFilteredImages(filterExprs)
	if err != nil {
		return nil, err
	}
	switch len(filteredResults) {
	case 0:
		return nil, fmt.Errorf("cannot find image for %s %s %s", distroName, imgTypeStr, archStr)
	case 1:
		return &filteredResults[0], nil
	default:
		return nil, fmt.Errorf("internal error: found %v results for %s %s %s", len(filteredResults), distroName, imgTypeStr, archStr)
	}
}

// XXX: rename FilterDistros to FilterImages(?)
func FilterDistros(fac *distrofactory.Factory, distroNames []string, filters Filters) ([]FilterResult, error) {
	var res []FilterResult

	if err := distrosort.Names(distroNames); err != nil {
		return nil, err
	}
	for _, distroName := range distroNames {
		distro := fac.GetDistro(distroName)
		if distro == nil {
			logrus.Debugf("skipping %v: has repositories but unsupported", distroName)
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
				if filters.Matches(distro, a, imgType) {
					res = append(res, FilterResult{distro, a, imgType})
				}
			}
		}
	}

	return res, nil
}

// Filters is a way to filter a list of distros
type Filters []filter

// NewFilters creates a filtering for a list of distros
func NewFilters(sl []string) (Filters, error) {
	var filters []filter
	for _, s := range sl {
		l := strings.SplitN(s, ":", 2)
		switch l[0] {
		case s:
			filters = append(filters, &distroNameFilter{
				filter: l[0],
				exact:  false,
			})
		case "name":
			filters = append(filters, &distroNameFilter{l[1], true})
		case "arch":
			filters = append(filters, &archFilter{l[1]})
		case "type":
			filters = append(filters, &imgTypeFilter{l[1]})
			// mostly here to show how powerful this is
		case "bootmode":
			filters = append(filters, &bootmodeFilter{l[1]})
		default:
			return nil, fmt.Errorf("unsupported filter prefix: %q", l[0])
		}
	}
	return filters, nil
}

// Matches returns true if the given (distro,arch,imgType) tuple matches
// the filter expressions
func (fl Filters) Matches(distro distro.Distro, arch distro.Arch, imgType distro.ImageType) bool {
	matches := true
	for _, f := range fl {
		matches = matches && f.Matches(distro, arch, imgType)
	}
	return matches
}

type filter interface {
	Matches(distro distro.Distro, arch distro.Arch, imgType distro.ImageType) bool
}

type distroNameFilter struct {
	filter string
	exact  bool
}

func (d *distroNameFilter) Matches(distro distro.Distro, arch distro.Arch, imgType distro.ImageType) bool {
	if d.exact {
		return distro.Name() == d.filter
	}
	return strings.Contains(distro.Name(), d.filter)
}

type archFilter struct {
	filter string
}

func (d *archFilter) Matches(distro distro.Distro, arch distro.Arch, imgType distro.ImageType) bool {
	return strings.Contains(arch.Name(), d.filter)
}

type imgTypeFilter struct {
	filter string
}

func (d *imgTypeFilter) Matches(distro distro.Distro, arch distro.Arch, imgType distro.ImageType) bool {
	return strings.Contains(imgType.Name(), d.filter)
}

type bootmodeFilter struct {
	filter string
}

func (d *bootmodeFilter) Matches(distro distro.Distro, arch distro.Arch, imgType distro.ImageType) bool {
	return imgType.BootMode().String() == d.filter
}
