package manifest

import (
	"fmt"
	"html/template"
	"io/fs"
	"strings"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/customizations/firstboot"
	"github.com/osbuild/images/pkg/customizations/fsnode"
)

// checkName prevents path traversal
func checkName(str string) error {
	if str == "" {
		return fmt.Errorf("name cannot be empty")
	}

	for _, r := range str {
		if !(('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z') || ('0' <= r && r <= '9') || r == '-' || r == '_') {
			return fmt.Errorf("name can only contain alphanumeric characters, dashes, and underscores")
		}
	}

	return nil
}

var tmplFirstbootUnit = `[Unit]
ConditionPathExists=!/var/local/.osbuild-custom-first-boot-done
Wants=network-online.target
After=network-online.target
After=osbuild-first-boot.service

[Service]
Type=oneshot
{{ range .Executables }}
ExecStart={{ . -}}
{{ end }}
ExecStartPost=/usr/bin/touch /var/local/.osbuild-custom-first-boot-done
RemainAfterExit=yes

[Install]
WantedBy=basic.target
`

var tmplFirstbootAAP = `#!/usr/bin/bash
curl -s -i --data "host_config_key={{ .HostConfigKey }}" {{ .URL -}}
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

func firstbootFileNodes(fbo *firstboot.FirstbootOptions) ([]string, []*fsnode.File, error) {
	if fbo == nil {
		return nil, nil, nil
	}

	var certs []string
	var files []*fsnode.File
	var executables []string

	if fbo.Satellite != nil {
		// add CA certificates to the list
		for _, cert := range fbo.Satellite.CACerts {
			certs = append(certs, cert)
		}

		// create the Satellite firstboot script
		f, err := fsnode.NewFile("/usr/local/bin/osbuild-first-satellite", common.ToPtr(fs.FileMode(0770)), "root", "root", []byte(fbo.Satellite.Command))
		if err != nil {
			return nil, nil, fmt.Errorf("error creating firstboot file node: %w", err)
		}

		files = append(files, f)
		executables = append(executables, "-"+f.Path())
	}

	if fbo.AAP != nil {
		// add CA certificates to the list
		for _, cert := range fbo.AAP.CACerts {
			certs = append(certs, cert)
		}

		// create the AAP firstboot script
		data := struct {
			URL           string
			HostConfigKey string
		}{
			URL:           fbo.AAP.JobTemplateURL,
			HostConfigKey: fbo.AAP.HostConfigKey,
		}
		aapContent, err := renderFirstboot(tmplFirstbootAAP, data)
		if err != nil {
			return nil, nil, fmt.Errorf("error rendering firstboot aap template: %w", err)
		}

		f, err := fsnode.NewFile("/usr/local/bin/osbuild-first-aap", common.ToPtr(fs.FileMode(0770)), "root", "root", []byte(aapContent))
		if err != nil {
			return nil, nil, fmt.Errorf("error creating firstboot file node: %w", err)
		}

		files = append(files, f)
		executables = append(executables, "-"+f.Path())
	}

	for i, custom := range fbo.Custom {
		// keep the naming convention consistent with the existing "osbuild-first-boot"
		name := fmt.Sprintf("osbuild-first-%s", custom.Name)
		if checkName(custom.Name) != nil {
			name = fmt.Sprintf("osbuild-first-custom-%d", i+1)
		}

		// create the executable
		exec := fmt.Sprintf("/usr/local/bin/%s", name)

		f, err := fsnode.NewFile(exec, common.ToPtr(fs.FileMode(0770)), "root", "root", []byte(custom.Contents))
		if err != nil {
			return nil, nil, fmt.Errorf("error creating firstboot file node: %w", err)
		}
		files = append(files, f)

		// prepare data for the systemd unit
		if custom.IgnoreFailure {
			exec = "-" + exec
		}
		executables = append(executables, exec)

	}

	// create the systemd unit
	data := struct {
		Executables []string
	}{
		Executables: executables,
	}
	unitContent, err := renderFirstboot(tmplFirstbootUnit, data)
	if err != nil {
		return nil, nil, fmt.Errorf("error rendering firstboot unit: %w", err)
	}
	unitFile, err := fsnode.NewFile("/etc/systemd/system/osbuild-first-boot.service", common.ToPtr(fs.FileMode(0644)), "root", "root", []byte(unitContent))
	if err != nil {
		return nil, nil, fmt.Errorf("error creating firstboot systemd unit file node: %w", err)
	}
	files = append(files, unitFile)

	return certs, files, nil
}
