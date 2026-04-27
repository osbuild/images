package osbuild

import (
	"fmt"
	"io/fs"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/customizations/firstboot"
	"github.com/osbuild/images/pkg/customizations/fsnode"
)

// GenFirstbootFromOptions processes the firstboot options and returns a list of CA certificates to
// include in the image, a list of file nodes to create the firstboot scripts, and
// a systemd unit to run the scripts on first boot.
func GenFirstbootFromOptions(fbo *firstboot.FirstbootOptions) ([]string, []*fsnode.File, *SystemdUnitCreateStageOptions, error) {
	if fbo == nil {
		return nil, nil, nil, nil
	}

	var certs []string       // list of CA certificates to include
	var files []*fsnode.File // list of file nodes to create
	var executables []string // list of executables for the systemd unit (Exec=)

	// add the marker file to indicate that firstboot scripts need to be run
	f, err := fsnode.NewFile("/var/local/.osbuild-custom-first-boot", common.ToPtr(fs.FileMode(0770)), "root", "root", []byte{})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error creating firstboot marker node: %w", err)
	}
	files = append(files, f)

	for _, script := range fbo.Scripts {
		// create the executable - filename was already sanitized
		exec := fmt.Sprintf("/usr/local/bin/%s", script.Filename)
		f, err := fsnode.NewFile(exec, common.ToPtr(fs.FileMode(0770)), "root", "root", []byte(script.Contents))
		if err != nil {
			return nil, nil, nil, fmt.Errorf("error creating firstboot file node %q: %w", exec, err)
		}
		files = append(files, f)

		// prepare data for the systemd unit
		if script.IgnoreFailure {
			exec = "-" + exec
		}
		executables = append(executables, exec)

		// add CA certificates to the list
		certs = append(certs, script.Certs...)
	}

	// create the main systemd unit:
	unit := SystemdUnit{
		Unit: &UnitSection{
			ConditionPathExists: []string{"/var/local/.osbuild-custom-first-boot"},
			Wants:               []string{"network-online.target"},
			After:               []string{"network-online.target", "osbuild-first-boot.service"},
		},
		Service: &ServiceSection{
			Type:            OneshotServiceType,
			ExecStart:       executables,
			ExecStartPre:    []string{"/usr/bin/rm /var/local/.osbuild-custom-first-boot"},
			RemainAfterExit: true,
		},
		Install: &InstallSection{
			WantedBy: []string{"basic.target"},
		},
	}

	unitOptions := &SystemdUnitCreateStageOptions{
		Filename: "osbuild-custom-first-boot.service",
		Config:   unit,
		UnitType: SystemUnitType,
		UnitPath: UsrUnitPath,
	}

	return certs, files, unitOptions, nil
}
