package osbuild

import (
	"fmt"
	"testing"

	"github.com/osbuild/images/internal/common"
	"github.com/stretchr/testify/assert"
)

func createSystemdUnit() SystemdUnit {

	var unit = UnitSection{
		Description:              "Create directory and files",
		DefaultDependencies:      common.ToPtr(true),
		ConditionPathExists:      []string{"!/etc/myfile"},
		ConditionPathIsDirectory: []string{"!/etc/mydir"},
		Requires:                 []string{"dbus.service", "libvirtd.service"},
		Wants:                    []string{"local-fs.target"},
	}
	var service = ServiceSection{
		Type:            OneshotServiceType,
		RemainAfterExit: true,
		ExecStartPre:    []string{"echo creating_files"},
		ExecStopPost:    []string{"echo done_creating_files"},
		ExecStart:       []string{"mkdir -p /etc/mydir", "touch /etc/myfiles"},
	}

	var install = InstallSection{
		RequiredBy: []string{"multi-user.target", "boot-complete.target"},
		WantedBy:   []string{"sshd.service"},
	}

	var systemdUnit = SystemdUnit{
		Unit:    &unit,
		Service: &service,
		Install: &install,
	}

	return systemdUnit
}

func TestNewSystemdUnitCreateStage(t *testing.T) {
	systemdServiceConfig := createSystemdUnit()
	var options = SystemdUnitCreateStageOptions{
		Filename: "create-dir-files.service",
		Config:   systemdServiceConfig,
	}
	expectedStage := &Stage{
		Type:    "org.osbuild.systemd.unit.create",
		Options: &options,
	}

	actualStage := NewSystemdUnitCreateStage(&options)
	assert.Equal(t, expectedStage, actualStage)
}

func TestNewSystemdUnitCreateStageInEtc(t *testing.T) {
	systemdServiceConfig := createSystemdUnit()
	var options = SystemdUnitCreateStageOptions{
		Filename: "create-dir-files.service",
		Config:   systemdServiceConfig,
		UnitPath: EtcUnitPath,
		UnitType: Global,
	}
	expectedStage := &Stage{
		Type:    "org.osbuild.systemd.unit.create",
		Options: &options,
	}

	actualStage := NewSystemdUnitCreateStage(&options)
	assert.Equal(t, expectedStage, actualStage)
}

func TestSystemdUnitStageOptionsValidation(t *testing.T) {
	unitSection := &UnitSection{
		Description:         "test-mount",
		DefaultDependencies: common.ToPtr(true),
	}
	mountSection := &MountSection{
		What:    "/dev/test",
		Where:   "/test",
		Type:    "ext4",
		Options: "defaults",
	}
	installSection := &InstallSection{
		WantedBy: []string{"multi-user.target"},
	}
	serviceSection := &ServiceSection{
		Type:            "oneshot",
		RemainAfterExit: true,
		ExecStart:       []string{"true"},
	}
	socketSection := &SocketSection{
		ListenStream: "/run/test/api.socket",
		SocketGroup:  "testgroup",
		SocketMode:   "660",
	}

	type testCase struct {
		options  SystemdUnitCreateStageOptions
		expected error
	}

	testCases := map[string]testCase{
		// OK
		"service-ok": {
			options: SystemdUnitCreateStageOptions{
				Filename: "test.service",
				UnitType: Global,
				UnitPath: EtcUnitPath,
				Config: SystemdUnit{
					Unit:    unitSection,
					Service: serviceSection,
					Install: installSection,
				},
			},
			expected: nil,
		},
		"mount-ok": {
			options: SystemdUnitCreateStageOptions{
				Filename: "test.mount",
				UnitType: Global,
				UnitPath: EtcUnitPath,
				Config: SystemdUnit{
					Unit:    unitSection,
					Mount:   mountSection,
					Install: installSection,
				},
			},
			expected: nil,
		},
		"socket-ok": {
			options: SystemdUnitCreateStageOptions{
				Filename: "test.socket",
				UnitType: Global,
				UnitPath: EtcUnitPath,
				Config: SystemdUnit{
					Unit:    unitSection,
					Install: installSection,
					Socket:  socketSection,
				},
			},
			expected: nil,
		},

		// missing required section
		"service-no-Service": {
			options: SystemdUnitCreateStageOptions{
				Filename: "test.service",
				UnitType: Global,
				UnitPath: EtcUnitPath,
				Config: SystemdUnit{
					Unit:    unitSection,
					Install: installSection,
				},
			},
			expected: fmt.Errorf(`systemd service unit "test.service" requires a Service section`),
		},
		"service-no-Install": {
			options: SystemdUnitCreateStageOptions{
				Filename: "test.service",
				UnitType: Global,
				UnitPath: EtcUnitPath,
				Config: SystemdUnit{
					Unit:    unitSection,
					Service: serviceSection,
				},
			},
			expected: fmt.Errorf(`systemd service unit "test.service" requires an Install section`),
		},
		"mount-no-Mount": {
			options: SystemdUnitCreateStageOptions{
				Filename: "test.mount",
				UnitType: Global,
				UnitPath: EtcUnitPath,
				Config: SystemdUnit{
					Unit:    unitSection,
					Install: installSection,
				},
			},
			expected: fmt.Errorf(`systemd mount unit "test.mount" requires a Mount section`),
		},
		"socket-no-Socket": {
			options: SystemdUnitCreateStageOptions{
				Filename: "test.socket",
				UnitType: Global,
				UnitPath: EtcUnitPath,
				Config: SystemdUnit{
					Unit:    unitSection,
					Install: installSection,
				},
			},
			expected: fmt.Errorf(`systemd socket unit "test.socket" requires a Socket section`),
		},

		// incorrect section for type
		"service-with-mount": {
			options: SystemdUnitCreateStageOptions{
				Filename: "test.service",
				UnitType: Global,
				UnitPath: EtcUnitPath,
				Config: SystemdUnit{
					Unit:    unitSection,
					Service: serviceSection,
					Mount:   mountSection,
					Install: installSection,
				},
			},
			expected: fmt.Errorf(`systemd service unit "test.service" contains invalid section Mount`),
		},
		"service-with-socket": {
			options: SystemdUnitCreateStageOptions{
				Filename: "test.service",
				UnitType: Global,
				UnitPath: EtcUnitPath,
				Config: SystemdUnit{
					Unit:    unitSection,
					Service: serviceSection,
					Socket:  socketSection,
					Install: installSection,
				},
			},
			expected: fmt.Errorf(`systemd service unit "test.service" contains invalid section Socket`),
		},
		"mount-with-service": {
			options: SystemdUnitCreateStageOptions{
				Filename: "test.mount",
				UnitType: Global,
				UnitPath: EtcUnitPath,
				Config: SystemdUnit{
					Unit:    unitSection,
					Mount:   mountSection,
					Service: serviceSection,
					Install: installSection,
				},
			},
			expected: fmt.Errorf(`systemd mount unit "test.mount" contains invalid section Service`),
		},
		"mount-with-socket": {
			options: SystemdUnitCreateStageOptions{
				Filename: "test.mount",
				UnitType: Global,
				UnitPath: EtcUnitPath,
				Config: SystemdUnit{
					Unit:    unitSection,
					Mount:   mountSection,
					Socket:  socketSection,
					Install: installSection,
				},
			},
			expected: fmt.Errorf(`systemd mount unit "test.mount" contains invalid section Socket`),
		},
		"socket-with-Service": {
			options: SystemdUnitCreateStageOptions{
				Filename: "test.socket",
				UnitType: Global,
				UnitPath: EtcUnitPath,
				Config: SystemdUnit{
					Unit:    unitSection,
					Install: installSection,
					Socket:  socketSection,
					Service: serviceSection,
				},
			},
			expected: fmt.Errorf(`systemd socket unit "test.socket" contains invalid section Service`),
		},
		"socket-with-Mount": {
			options: SystemdUnitCreateStageOptions{
				Filename: "test.socket",
				UnitType: Global,
				UnitPath: EtcUnitPath,
				Config: SystemdUnit{
					Unit:    unitSection,
					Install: installSection,
					Socket:  socketSection,
					Mount:   mountSection,
				},
			},
			expected: fmt.Errorf(`systemd socket unit "test.socket" contains invalid section Mount`),
		},

		// bad filename
		"bad-filename": {
			options: SystemdUnitCreateStageOptions{
				Filename: "//not-a-good-path//",
				UnitType: Global,
				UnitPath: EtcUnitPath,
				Config: SystemdUnit{
					Unit:    unitSection,
					Service: serviceSection,
					Install: installSection,
				},
			},
			expected: fmt.Errorf("invalid filename \"//not-a-good-path//\" for systemd unit: does not conform to schema (%s)", filenameRegex),
		},

		// bad extension
		"bad-extension": {
			options: SystemdUnitCreateStageOptions{
				Filename: "test.whatever",
				UnitType: Global,
				UnitPath: EtcUnitPath,
			},
			expected: fmt.Errorf(`invalid filename "test.whatever" for systemd unit: extension must be one of .service, .mount, or .socket`),
		},

		// missing required options
		"mount-no-what": {
			options: SystemdUnitCreateStageOptions{
				Filename: "test.mount",
				UnitType: Global,
				UnitPath: EtcUnitPath,
				Config: SystemdUnit{
					Unit: unitSection,
					Mount: &MountSection{
						Where:   "/test",
						Type:    "ext4",
						Options: "defaults",
					},
					Install: installSection,
				},
			},
			expected: fmt.Errorf(`What option for Mount section of systemd unit "test.mount" is required`),
		},
		"mount-no-where": {
			options: SystemdUnitCreateStageOptions{
				Filename: "test.mount",
				UnitType: Global,
				UnitPath: EtcUnitPath,
				Config: SystemdUnit{
					Unit: unitSection,
					Mount: &MountSection{
						What:    "/dev/test",
						Type:    "ext4",
						Options: "defaults",
					},
					Install: installSection,
				},
			},
			expected: fmt.Errorf(`Where option for Mount section of systemd unit "test.mount" is required`),
		},

		// invalid values
		"service-bad-env-vars": {
			options: SystemdUnitCreateStageOptions{
				Filename: "test.service",
				UnitType: Global,
				UnitPath: EtcUnitPath,
				Config: SystemdUnit{
					Unit: unitSection,
					Service: &ServiceSection{

						Type:            "oneshot",
						RemainAfterExit: true,
						ExecStart:       []string{"true"},
						Environment: []EnvironmentVariable{
							{
								Key:   ":bad_var/",
								Value: "can-be-whatever",
							},
						},
					},
					Install: installSection,
				},
			},
			expected: fmt.Errorf("variable name \":bad_var/\" doesn't conform to schema (%s)", envVarRegex),
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			err := tc.options.validate()
			assert.Equal(t, tc.expected, err)
		})
	}
}
