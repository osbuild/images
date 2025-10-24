package manifest

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/customizations/firstboot"
	"github.com/osbuild/images/pkg/customizations/fsnode"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/shutil"
)

var tmplFirstbootAAP = `#!/usr/bin/bash
curl -i --data {{ .HostConfigKey }} {{ .URL }}
`

func renderFirstboot(tmplStr string, data any) (string, error) {
	tmpl, err := template.New("firstboot-unit").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("error parsing firstboot unit template: %w", err)
	}

	var result strings.Builder
	err = tmpl.Execute(&result, data)
	if err != nil {
		return "", fmt.Errorf("error rendering firstboot unit: %w", err)
	}

	return result.String(), nil
}

// parse processes the firstboot options and returns a list of CA certificates to
// include in the image, a list of file nodes to create the firstboot scripts, and
// a systemd unit to run the scripts on first boot.
func parse(fbo *firstboot.FirstbootOptions) ([]string, []*fsnode.File, *osbuild.SystemdUnitCreateStageOptions, error) {
	if fbo == nil {
		return nil, nil, nil, nil
	}

	var certs []string
	var files []*fsnode.File
	var executables []string

	// add the marker file to indicate that firstboot scripts need to be run
	f, err := fsnode.NewFile("/var/local/.osbuild-custom-first-boot", common.ToPtr(fs.FileMode(0770)), "root", "root", []byte{})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error creating firstboot file node: %w", err)
	}
	files = append(files, f)

	appendExec := func(path string, ignoreFailure bool) {
		if ignoreFailure {
			path = "-" + path
		}
		executables = append(executables, path)
	}

	var ci int
	eachCustom := func(opt firstboot.CustomFirstbootOptions) error {
		// keep the naming convention consistent with the existing "osbuild-first-boot"
		name := fmt.Sprintf("osbuild-first-%s", filepath.Base(opt.Name))
		if opt.Name == "" {
			ci++
			name = fmt.Sprintf("osbuild-first-custom-%d", ci)
		}

		// create the executable
		exec := fmt.Sprintf("/usr/local/bin/%s", name)
		f, err := fsnode.NewFile(exec, common.ToPtr(fs.FileMode(0770)), "root", "root", []byte(opt.Contents))
		if err != nil {
			return fmt.Errorf("error creating firstboot file node: %w", err)
		}
		files = append(files, f)

		// prepare data for the systemd unit
		appendExec(exec, opt.IgnoreFailure)

		return nil
	}

	eachSatellite := func(opt firstboot.SatelliteFirstbootOptions) error {
		// add CA certificates to the list
		certs = append(certs, opt.CACerts...)

		// create the executable
		exec := "/usr/local/bin/osbuild-first-satellite"
		f, err := fsnode.NewFile(exec, common.ToPtr(fs.FileMode(0770)), "root", "root", []byte(opt.Command))
		if err != nil {
			return fmt.Errorf("error creating firstboot file node: %w", err)
		}
		files = append(files, f)

		// prepare data for the systemd unit
		appendExec(exec, opt.IgnoreFailure)

		return nil
	}

	eachAAP := func(opt firstboot.AAPFirstbootOptions) error {
		// add CA certificates to the list
		certs = append(certs, opt.CACerts...)

		// render the AAP firstboot script
		data := struct {
			URL           string
			HostConfigKey string
		}{
			URL:           shutil.Quote(opt.JobTemplateURL),
			HostConfigKey: shutil.Quote("host_config_key=" + opt.HostConfigKey),
		}
		aapContent, err := renderFirstboot(tmplFirstbootAAP, data)
		if err != nil {
			return fmt.Errorf("error rendering firstboot aap template: %w", err)
		}

		// create the executable
		exec := "/usr/local/bin/osbuild-first-aap"
		f, err := fsnode.NewFile(exec, common.ToPtr(fs.FileMode(0770)), "root", "root", []byte(aapContent))
		if err != nil {
			return fmt.Errorf("error creating firstboot file node: %w", err)
		}
		files = append(files, f)

		// prepare data for the systemd unit
		appendExec(exec, opt.IgnoreFailure)

		return nil
	}

	// iterate over all firstboot scripts and create the necessary structures
	err = fbo.Each(eachCustom, eachSatellite, eachAAP)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("parsing firstboot options: %w", err)
	}

	// create the main systemd unit:
	unit := osbuild.SystemdUnit{
		Unit: &osbuild.UnitSection{
			ConditionPathExists: []string{"/var/local/.osbuild-custom-first-boot"},
			Wants:               []string{"network-online.target"},
			After:               []string{"network-online.target", "osbuild-first-boot.service"},
		},
		Service: &osbuild.ServiceSection{
			Type:            "oneshot",
			ExecStart:       executables,
			ExecStartPre:    []string{"/usr/bin/rm /var/local/.osbuild-custom-first-boot"},
			RemainAfterExit: true,
		},
		Install: &osbuild.InstallSection{
			WantedBy: []string{"basic.target"},
		},
	}

	unitOptions := &osbuild.SystemdUnitCreateStageOptions{
		Filename: "osbuild-custom-first-boot.service",
		Config:   unit,
		UnitType: osbuild.SystemUnitType,
		UnitPath: osbuild.UsrUnitPath,
	}

	return certs, files, unitOptions, nil
}
