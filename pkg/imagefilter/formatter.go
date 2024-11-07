package imagefilter

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// OutputFormat contains the valid output formats for formatting results
type OutputFormat string

const (
	OutputFormatDefault OutputFormat = ""
	OutputFormatText    OutputFormat = "text"
	OutputFormatJSON    OutputFormat = "json"
)

// ResultFormatter will format the given result list to the given io.Writer
type ResultsFormatter interface {
	Output(io.Writer, []Result) error
}

// NewResultFormatter will create a formatter based on the given format.
func NewResultsFormatter(format OutputFormat) (ResultsFormatter, error) {
	switch format {
	case OutputFormatDefault, OutputFormatText:
		return &textResultsFormatter{}, nil
	case OutputFormatJSON:
		return &jsonResultsFormatter{}, nil
	default:
		return nil, fmt.Errorf("unsupported formatter %q", format)
	}
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
