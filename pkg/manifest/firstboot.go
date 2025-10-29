package manifest

import (
	"fmt"
	"io/fs"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/customizations/firstboot"
	"github.com/osbuild/images/pkg/customizations/fsnode"
	"github.com/osbuild/images/pkg/osbuild"
)

// parse processes the firstboot options and returns a list of CA certificates to
// include in the image, a list of file nodes to create the firstboot scripts, and
// a systemd unit to run the scripts on first boot.
// TODO RENAME THIS
func parse(fbo *firstboot.FirstbootOptions) ([]string, []*fsnode.File, *osbuild.SystemdUnitCreateStageOptions, error) {
	if fbo == nil {
		return nil, nil, nil, nil
	}

	var certs []string       // list of CA certificates to include
	var files []*fsnode.File // list of file nodes to create
	var executables []string // list of executables for the systemd unit (Exec=)

	// add the marker file to indicate that firstboot scripts need to be run
	f, err := fsnode.NewFile("/var/local/.osbuild-custom-first-boot", common.ToPtr(fs.FileMode(0770)), "root", "root", []byte{})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error creating firstboot file node: %w", err)
	}
	files = append(files, f)

	for _, script := range fbo.Scripts {
		// create the executable
		exec := fmt.Sprintf("/usr/local/bin/%s", script.Filename)
		f, err := fsnode.NewFile(exec, common.ToPtr(fs.FileMode(0770)), "root", "root", []byte(script.Contents))
		if err != nil {
			return nil, nil, nil, fmt.Errorf("error creating firstboot file node: %w", err)
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
