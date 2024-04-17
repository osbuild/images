package osbuild

import (
	"testing"

	"github.com/osbuild/images/internal/common"
	"github.com/stretchr/testify/assert"
)

func createSystemdUnit() SystemdServiceUnit {

	var unit = Unit{
		Description:              "Create directory and files",
		DefaultDependencies:      common.ToPtr(true),
		ConditionPathExists:      []string{"!/etc/myfile"},
		ConditionPathIsDirectory: []string{"!/etc/mydir"},
		Requires:                 []string{"dbus.service", "libvirtd.service"},
		Wants:                    []string{"local-fs.target"},
	}
	var service = Service{
		Type:            Oneshot,
		RemainAfterExit: true,
		ExecStartPre:    []string{"echo creating_files"},
		ExecStopPost:    []string{"echo done_creating_files"},
		ExecStart:       []string{"mkdir -p /etc/mydir", "touch /etc/myfiles"},
	}

	var install = Install{
		RequiredBy: []string{"multi-user.target", "boot-complete.target"},
		WantedBy:   []string{"sshd.service"},
	}

	var systemdUnit = SystemdServiceUnit{
		Unit:    &unit,
		Service: &service,
		Install: &install,
	}

	return systemdUnit
}

func TestNewSystemdUnitCreateStage(t *testing.T) {
	systemdServiceConfig := createSystemdUnit()
	var options = SystemdUnitCreateStageOptions{
		Filename: "create-dir-files",
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
		Filename: "create-dir-files",
		Config:   systemdServiceConfig,
		UnitPath: Etc,
		UnitType: Global,
	}
	expectedStage := &Stage{
		Type:    "org.osbuild.systemd.unit.create",
		Options: &options,
	}

	actualStage := NewSystemdUnitCreateStage(&options)
	assert.Equal(t, expectedStage, actualStage)
}
