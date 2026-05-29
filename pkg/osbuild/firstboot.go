package osbuild

import (
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/customizations/firstboot"
	"github.com/osbuild/images/pkg/customizations/fsnode"
)

var firstbootBaseAfter = []string{"network-online.target", "osbuild-first-boot.service"}

func firstbootMarkerPath(unitFilename string) string {
	// e.g. osbuild-first-setup.service -> /var/local/.osbuild-first-setup
	base := strings.TrimSuffix(unitFilename, ".service")
	return fmt.Sprintf("/var/local/.%s", base)
}

// GenFirstbootFromOptions processes the firstboot options and returns a list of CA certificates to
// include in the image, a list of file nodes to create the firstboot scripts, and
// systemd units to run the scripts on first boot.
func GenFirstbootFromOptions(fbo *firstboot.FirstbootOptions) ([]string, []*fsnode.File, []*SystemdUnitCreateStageOptions, error) {
	if fbo == nil {
		return nil, nil, nil, nil
	}

	var certs []string
	var files []*fsnode.File
	var units []*SystemdUnitCreateStageOptions

	var prevUnit string
	for _, script := range fbo.Scripts {
		unitFilename := script.Filename + ".service"
		markerPath := firstbootMarkerPath(unitFilename)

		f, err := fsnode.NewFile(markerPath, common.ToPtr(fs.FileMode(0770)), "root", "root", []byte{})
		if err != nil {
			return nil, nil, nil, fmt.Errorf("error creating firstboot marker node: %w", err)
		}
		files = append(files, f)

		exec := fmt.Sprintf("/usr/local/bin/%s", script.Filename)
		f, err = fsnode.NewFile(exec, common.ToPtr(fs.FileMode(0770)), "root", "root", []byte(script.Contents))
		if err != nil {
			return nil, nil, nil, fmt.Errorf("error creating firstboot file node %q: %w", exec, err)
		}
		files = append(files, f)

		execStart := exec
		if script.IgnoreFailure {
			execStart = "-" + exec
		}

		after := append([]string{}, firstbootBaseAfter...)
		after = append(after, script.After...)
		if prevUnit != "" {
			after = append(after, prevUnit)
		}
		after = dedupeOrdered(after)

		unit := SystemdUnit{
			Unit: &UnitSection{
				ConditionPathExists: []string{markerPath},
				Wants:               []string{"network-online.target"},
				After:               after,
				Before:              slices.Clone(script.Before),
			},
			Service: &ServiceSection{
				Type:            OneshotServiceType,
				ExecStart:       []string{execStart},
				ExecStartPre:    []string{"/usr/bin/rm " + markerPath},
				RemainAfterExit: true,
			},
			Install: &InstallSection{
				WantedBy: []string{"basic.target"},
			},
		}

		units = append(units, &SystemdUnitCreateStageOptions{
			Filename: unitFilename,
			Config:   unit,
			UnitType: SystemUnitType,
			UnitPath: UsrUnitPath,
		})
		prevUnit = unitFilename

		certs = append(certs, script.Certs...)
	}

	return certs, files, units, nil
}

func dedupeOrdered(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
