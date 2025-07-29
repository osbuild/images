package firstboot

import (
	"errors"
	"net/url"

	"github.com/osbuild/images/pkg/blueprint"
)

type FirstbootOptions struct {
	Custom    []CustomFirstbootOptions
	Satellite *SatelliteFirstbootOptions
	AAP       *AAPFirstbootOptions
}

type CustomFirstbootOptions struct {
	Contents      string
	Name          string
	IgnoreFailure bool
}

type SatelliteFirstbootOptions struct {
	Command string
	CACerts []string
}

type AAPFirstbootOptions struct {
	JobTemplateURL string
	HostConfigKey  string
	CACerts        []string
}

func FirstbootOptionsFromBP(bpFirstboot blueprint.FirstbootCustomization) (*FirstbootOptions, error) {
	var custom []CustomFirstbootOptions
	for _, c := range bpFirstboot.Custom {
		if c.Contents == "" {
			return nil, errors.New("custom firstboot script contents cannot be empty")
		}

		custom = append(custom, CustomFirstbootOptions{
			Contents:      c.Contents,
			Name:          c.Name,
			IgnoreFailure: c.IgnoreFailure,
		})
	}

	var satellite *SatelliteFirstbootOptions
	if bpFirstboot.Satellite != nil {
		if bpFirstboot.Satellite.Command == "" {
			return nil, errors.New("satellite firstboot command cannot be empty")
		}

		var certs []string
		for _, cert := range bpFirstboot.Satellite.CACerts {
			certs = append(certs, cert)
		}

		satellite = &SatelliteFirstbootOptions{
			Command: bpFirstboot.Satellite.Command,
			CACerts: certs,
		}
	}

	var aap *AAPFirstbootOptions
	if bpFirstboot.AAP != nil {
		if bpFirstboot.AAP.JobTemplateURL == "" {
			return nil, errors.New("AAP firstboot job template URL cannot be empty")
		}

		if _, err := url.ParseRequestURI(bpFirstboot.AAP.JobTemplateURL); err != nil {
			return nil, errors.New("AAP firstboot job template URL is not a valid URI")
		}

		var certs []string
		for _, cert := range bpFirstboot.AAP.CACerts {
			certs = append(certs, cert)
		}

		aap = &AAPFirstbootOptions{
			JobTemplateURL: bpFirstboot.AAP.JobTemplateURL,
			HostConfigKey:  bpFirstboot.AAP.HostConfigKey,
			CACerts:        certs,
		}
	}

	return &FirstbootOptions{
		Custom:    custom,
		Satellite: satellite,
		AAP:       aap,
	}, nil
}
