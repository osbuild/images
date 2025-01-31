package imagefilter

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/osbuild/images/pkg/distrosort"
	// we cannot use "maps" yet, as it needs go1.23
	"golang.org/x/exp/maps"
)

// OutputFormat contains the valid output formats for formatting results
type OutputFormat string

const (
	OutputFormatDefault   OutputFormat = ""
	OutputFormatText      OutputFormat = "text"
	OutputFormatJSON      OutputFormat = "json"
	OutputFormatTextShell OutputFormat = "shell"
	OutputFormatTextShort OutputFormat = "short"
)

// ResultFormatter will format the given result list to the given io.Writer
type ResultsFormatter interface {
	Output(io.Writer, []Result) error
}

var supportedFormatters = map[string]ResultsFormatter{
	string(OutputFormatDefault):   &textResultsFormatter{},
	string(OutputFormatText):      &textResultsFormatter{},
	string(OutputFormatJSON):      &jsonResultsFormatter{},
	string(OutputFormatTextShell): &shellResultsFormatter{},
	string(OutputFormatTextShort): &textShortResultsFormatter{},
}

// SupportedOutputFormats returns a list of supported output formats
func SupportedOutputFormats() []string {
	keys := maps.Keys(supportedFormatters)
	sort.Strings(keys)
	return keys
}

// NewResultsFormatter will create a formatter based on the given format.
func NewResultsFormatter(format OutputFormat) (ResultsFormatter, error) {
	rs, ok := supportedFormatters[string(format)]
	if !ok {
		return nil, fmt.Errorf("unsupported formatter %q", format)
	}
	return rs, nil
}

type textResultsFormatter struct{}

func (*textResultsFormatter) Output(w io.Writer, all []Result) error {
	var errs []error

	for _, res := range all {
		// The should be copy/paste friendly, i.e. the "image-builder"
		// cmdline should support:
		//   image-builder manifest centos-9 type:qcow2 arch:s390
		//   image-builder build centos-9 type:qcow2 arch:x86_64
		if _, err := fmt.Fprintf(w, "%s type:%s arch:%s\n", res.Distro.Name(), res.ImgType.Name(), res.Arch.Name()); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

type shellResultsFormatter struct{}

func (*shellResultsFormatter) Output(w io.Writer, all []Result) error {
	var errs []error

	for _, res := range all {
		if _, err := fmt.Fprintf(w, "%s --distro %s --arch %s\n",
			res.ImgType.Name(),
			res.Distro.Name(),
			res.Arch.Name()); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

type textShortResultsFormatter struct{}

func (*textShortResultsFormatter) Output(w io.Writer, all []Result) error {
	var errs []error

	// deliberately break the yaml until the feature is stable, there
	// are open questions, e.g. how this relates to:
	// https://github.com/osbuild/osbuild-composer/pull/4336
	// which adds a similar but slightly different API
	fmt.Fprint(w, "@WARNING - the output format is not stable yet and may change\n")

	outputMap := make(map[string]map[string][]string)
	for _, res := range all {
		if _, ok := outputMap[res.Distro.Name()]; !ok {
			outputMap[res.Distro.Name()] = make(map[string][]string)
		}
		outputMap[res.Distro.Name()][res.ImgType.Name()] = append(outputMap[res.Distro.Name()][res.ImgType.Name()], res.Arch.Name())
	}

	// Sort and prepare output
	var distros []string
	for distro := range outputMap {
		distros = append(distros, distro)
	}
	if err := distrosort.Names(distros); err != nil {
		return fmt.Errorf("cannot sort distro names %q: %w", distros, err)
	}

	for _, distro := range distros {
		var types []string
		for t := range outputMap[distro] {
			types = append(types, t)
		}
		sort.Strings(types)

		var typeArchPairs []string
		for _, t := range types {
			arches := outputMap[distro][t]
			sort.Strings(arches)
			typeArchPairs = append(typeArchPairs, fmt.Sprintf("%s: [ %s ]", t, strings.Join(arches, ", ")))
		}

		if _, err := fmt.Fprintf(w, "%s:\n  %s\n", distro, strings.Join(typeArchPairs, "\n  ")); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

type jsonResultsFormatter struct{}

type distroResultJSON struct {
	Name string `json:"name"`
}

type archResultJSON struct {
	Name string `json:"name"`
}

type imgTypeResultJSON struct {
	Name string `json:"name"`
}

type filteredResultJSON struct {
	Distro  distroResultJSON  `json:"distro"`
	Arch    archResultJSON    `json:"arch"`
	ImgType imgTypeResultJSON `json:"image_type"`
}

func (*jsonResultsFormatter) Output(w io.Writer, all []Result) error {
	var out []filteredResultJSON

	for _, res := range all {
		out = append(out, filteredResultJSON{
			Distro: distroResultJSON{
				Name: res.Distro.Name(),
			},
			Arch: archResultJSON{
				Name: res.Arch.Name(),
			},
			ImgType: imgTypeResultJSON{
				Name: res.ImgType.Name(),
			},
		})
	}

	enc := json.NewEncoder(w)
	return enc.Encode(out)
}
