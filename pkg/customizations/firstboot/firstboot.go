package firstboot

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"

	"github.com/osbuild/blueprint/pkg/blueprint"
	"github.com/osbuild/images/pkg/shutil"
)

type FirstbootOptions struct {
	Scripts []Script
}

type Script struct {
	Filename      string
	Contents      string
	IgnoreFailure bool
	Certs         []string
	After         []string
	Before        []string
}

var ErrFirstbootAlreadySet = errors.New("firstboot customization already set")

var reservedRegexp = regexp.MustCompile(`^(custom|satellite|aap)-\d+$`)

// scriptNameGenerator assigns stable executable and unit names for firstboot scripts.
type scriptNameGenerator struct {
	counter     int
	alreadyUsed []string
}

func (g *scriptNameGenerator) generate(inputName, prefix string) string {
	outputName := ""
	if inputName != "" {
		outputName = fmt.Sprintf("osbuild-first-%s", inputName)
	}
	useNumberBased := inputName == "" || !filepath.IsLocal(inputName) ||
		reservedRegexp.MatchString(inputName) ||
		(outputName != "" && slices.Contains(g.alreadyUsed, outputName))

	if useNumberBased {
		g.counter++
		for slices.Contains(g.alreadyUsed, fmt.Sprintf("osbuild-first-%s-%d", prefix, g.counter)) {
			g.counter++
		}
		name := fmt.Sprintf("osbuild-first-%s-%d", prefix, g.counter)
		g.alreadyUsed = append(g.alreadyUsed, name)
		return name
	}

	g.alreadyUsed = append(g.alreadyUsed, outputName)
	return outputName
}

func scriptFromCommon(common blueprint.FirstbootCommonCustomization, filename, contents string, certs []string) Script {
	return Script{
		Filename:      filename,
		Contents:      contents,
		IgnoreFailure: common.IgnoreFailure,
		Certs:         certs,
		After:         slices.Clone(common.After),
		Before:        slices.Clone(common.Before),
	}
}

// FirstbootOptionsFromBP converts a blueprint FirstbootCustomization to
// FirstbootOptions. Validation is done in the blueprint package, so this function
// assumes the input is valid, however, JSON unmarshalling errors are possible.
func FirstbootOptionsFromBP(bpFirstboot blueprint.FirstbootCustomization) (*FirstbootOptions, error) {
	fo := &FirstbootOptions{}
	var satDone, aapDone bool
	var ng scriptNameGenerator

	for _, fbsc := range bpFirstboot.Scripts {
		cust, sat, aap, err := fbsc.SelectUnion()
		if err != nil {
			return nil, err
		}

		if cust != nil {
			fo.Scripts = append(fo.Scripts, scriptFromCommon(
				cust.FirstbootCommonCustomization,
				ng.generate(cust.Name, "custom"),
				cust.Contents,
				nil,
			))
		}

		if sat != nil {
			if satDone {
				return nil, fmt.Errorf("%w: satellite", ErrFirstbootAlreadySet)
			}
			satDone = true

			fo.Scripts = append(fo.Scripts, scriptFromCommon(
				sat.FirstbootCommonCustomization,
				ng.generate(sat.Name, "satellite"),
				sat.Command,
				sat.CACerts,
			))
		}

		if aap != nil {
			if aapDone {
				return nil, fmt.Errorf("%w: aap", ErrFirstbootAlreadySet)
			}
			aapDone = true

			contents := fmt.Sprintf("#!/usr/bin/bash\ncurl -i --data %s %s\n",
				shutil.Quote("host_config_key="+aap.HostConfigKey),
				shutil.Quote(aap.JobTemplateURL),
			)

			fo.Scripts = append(fo.Scripts, scriptFromCommon(
				aap.FirstbootCommonCustomization,
				ng.generate(aap.Name, "aap"),
				contents,
				aap.CACerts,
			))
		}
	}

	return fo, nil
}
