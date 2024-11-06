package main

import (
	"fmt"

	"github.com/osbuild/images/pkg/distrofactory"
	"github.com/osbuild/images/pkg/imagefilter"
)

func newImageFilterDefault() (*imagefilter.ImageFilter, error) {
	fac := distrofactory.NewDefault()
	repos, err := newRepoRegistry()
	if err != nil {
		return nil, err
	}
	return imagefilter.New(fac, repos)
}

func getOneImage(distroName, imgTypeStr, archStr string) (*imagefilter.Result, error) {
	imageFilter, err := newImageFilterDefault()
	if err != nil {
		return nil, err
	}

	filterExprs := []string{
		fmt.Sprintf("name:%s", distroName),
		fmt.Sprintf("arch:%s", archStr),
		fmt.Sprintf("type:%s", imgTypeStr),
	}
	filteredResults, err := imageFilter.Filter(filterExprs...)
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
