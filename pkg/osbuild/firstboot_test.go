package osbuild

import (
	"io/fs"
	"testing"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/customizations/firstboot"
	"github.com/osbuild/images/pkg/customizations/fsnode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func firstbootTestFile(path, data string) *fsnode.File {
	f, err := fsnode.NewFile(path, common.ToPtr(fs.FileMode(0770)), "root", "root", []byte(data))
	if err != nil {
		panic(err)
	}
	return f
}

func firstbootTestUnit(filename string, unit *UnitSection, service *ServiceSection) *SystemdUnitCreateStageOptions {
	return &SystemdUnitCreateStageOptions{
		Filename: filename,
		UnitType: SystemUnitType,
		UnitPath: UsrUnitPath,
		Config: SystemdUnit{
			Unit: unit,
			Service: &ServiceSection{
				Type:            OneshotServiceType,
				ExecStart:       service.ExecStart,
				ExecStartPre:    service.ExecStartPre,
				RemainAfterExit: true,
			},
			Install: &InstallSection{
				WantedBy: []string{"basic.target"},
			},
		},
	}
}

func TestGenFirstbootFromOptions(t *testing.T) {
	tests := []struct {
		name      string
		fbo       *firstboot.FirstbootOptions
		wantCerts []string
		wantFiles []*fsnode.File
		wantUnits []*SystemdUnitCreateStageOptions
	}{
		{
			name: "nil",
			fbo:  nil,
		},
		{
			name: "empty-scripts",
			fbo:  &firstboot.FirstbootOptions{},
		},
		{
			name: "single-script",
			fbo: &firstboot.FirstbootOptions{
				Scripts: []firstboot.Script{
					{
						Filename: "osbuild-first-setup",
						Contents: "#!/bin/bash\necho setup\n",
					},
				},
			},
			wantFiles: []*fsnode.File{
				firstbootTestFile(firstbootMarkerPath("osbuild-first-setup.service"), ""),
				firstbootTestFile("/usr/local/bin/osbuild-first-setup", "#!/bin/bash\necho setup\n"),
			},
			wantUnits: []*SystemdUnitCreateStageOptions{
				firstbootTestUnit("osbuild-first-setup.service",
					&UnitSection{
						ConditionPathExists: []string{firstbootMarkerPath("osbuild-first-setup.service")},
						Wants:               []string{"network-online.target"},
						After:               []string{"network-online.target", "osbuild-first-boot.service"},
					},
					&ServiceSection{
						ExecStartPre: []string{"/usr/bin/rm " + firstbootMarkerPath("osbuild-first-setup.service")},
						ExecStart:    []string{"/usr/local/bin/osbuild-first-setup"},
					},
				),
			},
		},
		{
			name: "multiple-scripts",
			fbo: &firstboot.FirstbootOptions{
				Scripts: []firstboot.Script{
					{
						Filename:      "osbuild-first-satellite",
						Contents:      "#!/usr/bin/bash\ncurl https://sat.example.com/register",
						IgnoreFailure: true,
						Certs:         []string{"cert1", "cert2"},
					},
					{
						Filename:      "osbuild-first-aap",
						Contents:      "#!/usr/bin/bash\ncurl -i --data 'host_config_key=host-config-key' 'https://aap.example.com/api/v2/job_templates/9/callback/'\n",
						IgnoreFailure: true,
						Certs:         []string{"cert3", "cert4"},
						After:         []string{"sshd.service"},
					},
					{
						Filename: "osbuild-first-custom-1",
						Contents: "echo 'unnamed'",
					},
				},
			},
			wantCerts: []string{"cert1", "cert2", "cert3", "cert4"},
			wantFiles: []*fsnode.File{
				firstbootTestFile(firstbootMarkerPath("osbuild-first-satellite.service"), ""),
				firstbootTestFile("/usr/local/bin/osbuild-first-satellite", "#!/usr/bin/bash\ncurl https://sat.example.com/register"),
				firstbootTestFile(firstbootMarkerPath("osbuild-first-aap.service"), ""),
				firstbootTestFile("/usr/local/bin/osbuild-first-aap", "#!/usr/bin/bash\ncurl -i --data 'host_config_key=host-config-key' 'https://aap.example.com/api/v2/job_templates/9/callback/'\n"),
				firstbootTestFile(firstbootMarkerPath("osbuild-first-custom-1.service"), ""),
				firstbootTestFile("/usr/local/bin/osbuild-first-custom-1", "echo 'unnamed'"),
			},
			wantUnits: []*SystemdUnitCreateStageOptions{
				firstbootTestUnit("osbuild-first-satellite.service",
					&UnitSection{
						ConditionPathExists: []string{firstbootMarkerPath("osbuild-first-satellite.service")},
						Wants:               []string{"network-online.target"},
						After:               []string{"network-online.target", "osbuild-first-boot.service"},
					},
					&ServiceSection{
						ExecStartPre: []string{"/usr/bin/rm " + firstbootMarkerPath("osbuild-first-satellite.service")},
						ExecStart:    []string{"-/usr/local/bin/osbuild-first-satellite"},
					},
				),
				firstbootTestUnit("osbuild-first-aap.service",
					&UnitSection{
						ConditionPathExists: []string{firstbootMarkerPath("osbuild-first-aap.service")},
						Wants:               []string{"network-online.target"},
						After: []string{
							"network-online.target",
							"osbuild-first-boot.service",
							"sshd.service",
							"osbuild-first-satellite.service",
						},
					},
					&ServiceSection{
						ExecStartPre: []string{"/usr/bin/rm " + firstbootMarkerPath("osbuild-first-aap.service")},
						ExecStart:    []string{"-/usr/local/bin/osbuild-first-aap"},
					},
				),
				firstbootTestUnit("osbuild-first-custom-1.service",
					&UnitSection{
						ConditionPathExists: []string{firstbootMarkerPath("osbuild-first-custom-1.service")},
						Wants:               []string{"network-online.target"},
						After: []string{
							"network-online.target",
							"osbuild-first-boot.service",
							"osbuild-first-aap.service",
						},
					},
					&ServiceSection{
						ExecStartPre: []string{"/usr/bin/rm " + firstbootMarkerPath("osbuild-first-custom-1.service")},
						ExecStart:    []string{"/usr/local/bin/osbuild-first-custom-1"},
					},
				),
			},
		},
		{
			name: "ignore-failure-and-before",
			fbo: &firstboot.FirstbootOptions{
				Scripts: []firstboot.Script{
					{
						Filename:      "osbuild-first-ignore-errors",
						Contents:      "echo 'ignore errors'",
						IgnoreFailure: true,
						Before:        []string{"postgresql.service"},
					},
				},
			},
			wantFiles: []*fsnode.File{
				firstbootTestFile(firstbootMarkerPath("osbuild-first-ignore-errors.service"), ""),
				firstbootTestFile("/usr/local/bin/osbuild-first-ignore-errors", "echo 'ignore errors'"),
			},
			wantUnits: []*SystemdUnitCreateStageOptions{
				firstbootTestUnit("osbuild-first-ignore-errors.service",
					&UnitSection{
						ConditionPathExists: []string{firstbootMarkerPath("osbuild-first-ignore-errors.service")},
						Wants:               []string{"network-online.target"},
						After:               []string{"network-online.target", "osbuild-first-boot.service"},
						Before:              []string{"postgresql.service"},
					},
					&ServiceSection{
						ExecStartPre: []string{"/usr/bin/rm " + firstbootMarkerPath("osbuild-first-ignore-errors.service")},
						ExecStart:    []string{"-/usr/local/bin/osbuild-first-ignore-errors"},
					},
				),
			},
		},
		{
			name: "two-scripts-in-order",
			fbo: &firstboot.FirstbootOptions{
				Scripts: []firstboot.Script{
					{
						Filename: "osbuild-first-one",
						Contents: "echo one",
					},
					{
						Filename: "osbuild-first-two",
						Contents: "echo two",
					},
				},
			},
			wantFiles: []*fsnode.File{
				firstbootTestFile(firstbootMarkerPath("osbuild-first-one.service"), ""),
				firstbootTestFile("/usr/local/bin/osbuild-first-one", "echo one"),
				firstbootTestFile(firstbootMarkerPath("osbuild-first-two.service"), ""),
				firstbootTestFile("/usr/local/bin/osbuild-first-two", "echo two"),
			},
			wantUnits: []*SystemdUnitCreateStageOptions{
				firstbootTestUnit("osbuild-first-one.service",
					&UnitSection{
						ConditionPathExists: []string{firstbootMarkerPath("osbuild-first-one.service")},
						Wants:               []string{"network-online.target"},
						After:               []string{"network-online.target", "osbuild-first-boot.service"},
					},
					&ServiceSection{
						ExecStartPre: []string{"/usr/bin/rm " + firstbootMarkerPath("osbuild-first-one.service")},
						ExecStart:    []string{"/usr/local/bin/osbuild-first-one"},
					},
				),
				firstbootTestUnit("osbuild-first-two.service",
					&UnitSection{
						ConditionPathExists: []string{firstbootMarkerPath("osbuild-first-two.service")},
						Wants:               []string{"network-online.target"},
						After: []string{
							"network-online.target",
							"osbuild-first-boot.service",
							"osbuild-first-one.service",
						},
					},
					&ServiceSection{
						ExecStartPre: []string{"/usr/bin/rm " + firstbootMarkerPath("osbuild-first-two.service")},
						ExecStart:    []string{"/usr/local/bin/osbuild-first-two"},
					},
				),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			certs, files, units, err := GenFirstbootFromOptions(tt.fbo)
			require.NoError(t, err)

			if tt.fbo == nil {
				assert.Nil(t, certs)
				assert.Nil(t, files)
				assert.Nil(t, units)
				return
			}

			assert.Equal(t, tt.wantCerts, certs)
			assert.Equal(t, tt.wantUnits, units)
			require.Len(t, files, len(tt.wantFiles))
			for i := range tt.wantFiles {
				assert.Equal(t, tt.wantFiles[i].Path(), files[i].Path())
				assert.Equal(t, tt.wantFiles[i].Data(), files[i].Data())
			}
		})
	}
}

func TestFirstbootMarkerPath(t *testing.T) {
	assert.Equal(t, "/var/local/.osbuild-first-setup", firstbootMarkerPath("osbuild-first-setup.service"))
}
