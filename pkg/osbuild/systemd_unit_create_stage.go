package osbuild

import (
	"fmt"
	"regexp"
)

type SystemdServiceType string
type SystemdUnitPath string

const (
	SimpleServiceType       SystemdServiceType = "simple"
	ExecServiceType         SystemdServiceType = "exec"
	ForkingServiceType      SystemdServiceType = "forking"
	OneshotServiceType      SystemdServiceType = "oneshot"
	DbusServiceType         SystemdServiceType = "dbus"
	NotifyServiceType       SystemdServiceType = "notify"
	NotifyReloadServiceType SystemdServiceType = "notify-reload"
	IdleServiceType         SystemdServiceType = "idle"

	EtcUnitPath SystemdUnitPath = "etc"
	UsrUnitPath SystemdUnitPath = "usr"
)

type UnitSection struct {
	Description              string   `json:"Description,omitempty"`
	DefaultDependencies      *bool    `json:"DefaultDependencies,omitempty"`
	ConditionPathExists      []string `json:"ConditionPathExists,omitempty"`
	ConditionPathIsDirectory []string `json:"ConditionPathIsDirectory,omitempty"`
	Requires                 []string `json:"Requires,omitempty"`
	Wants                    []string `json:"Wants,omitempty"`
	After                    []string `json:"After,omitempty"`
	Before                   []string `json:"Before,omitempty"`
}

type ServiceSection struct {
	Type            SystemdServiceType    `json:"Type,omitempty"`
	RemainAfterExit bool                  `json:"RemainAfterExit,omitempty"`
	ExecStartPre    []string              `json:"ExecStartPre,omitempty"`
	ExecStopPost    []string              `json:"ExecStopPost,omitempty"`
	ExecStart       []string              `json:"ExecStart,omitempty"`
	Environment     []EnvironmentVariable `json:"Environment,omitempty"`
	EnvironmentFile []string              `json:"EnvironmentFile,omitempty"`
}

type MountSection struct {
	What    string `json:"What"`
	Where   string `json:"Where"`
	Type    string `json:"Type,omitempty"`
	Options string `json:"Options,omitempty"`
}

type SocketSection struct {
	Service                string `json:"Service,omitempty"`
	ListenStream           string `json:"ListenStream,omitempty"`
	ListenDatagram         string `json:"ListenDatagram,omitempty"`
	ListenSequentialPacket string `json:"ListenSequentialPacket,omitempty"`
	ListenFifo             string `json:"ListenFifo,omitempty"`
	SocketUser             string `json:"SocketUser,omitempty"`
	SocketGroup            string `json:"SocketGroup,omitempty"`
	SocketMode             string `json:"SocketMode,omitempty"`
	DirectoryMode          string `json:"DirectoryMode,omitempty"`
	Accept                 string `json:"Accept,omitempty"`
	RuntimeDirectory       string `json:"RuntimeDirectory,omitempty"`
	RemoveOnStop           string `json:"RemoveOnStop,omitempty"`
}

type InstallSection struct {
	RequiredBy []string `json:"RequiredBy,omitempty"`
	WantedBy   []string `json:"WantedBy,omitempty"`
}

type SystemdServiceUnit struct {
	Unit    *UnitSection    `json:"Unit"`
	Service *ServiceSection `json:"Service"`
	Mount   *MountSection   `json:"Mount,omitempty"`
	Socket  *SocketSection  `json:"Socket,omitempty"`
	Install *InstallSection `json:"Install"`
}

type SystemdUnitCreateStageOptions struct {
	Filename string             `json:"filename"`
	UnitType unitType           `json:"unit-type,omitempty"` // unitType defined in ./systemd_unit_stage.go
	UnitPath SystemdUnitPath    `json:"unit-path,omitempty"`
	Config   SystemdServiceUnit `json:"config"`
}

func (SystemdUnitCreateStageOptions) isStageOptions() {}

func (o *SystemdUnitCreateStageOptions) validate() error {
	fre := regexp.MustCompile(filenameRegex)
	if !fre.MatchString(o.Filename) {
		return fmt.Errorf("filename %q doesn't conform to schema (%s)", o.Filename, filenameRegex)
	}

	if o.Config.Install == nil {
		return fmt.Errorf("Install section of systemd unit is required")
	}

	vre := regexp.MustCompile(envVarRegex)
	if service := o.Config.Service; service != nil {
		for _, envVar := range service.Environment {
			if !vre.MatchString(envVar.Key) {
				return fmt.Errorf("variable name %q doesn't conform to schema (%s)", envVar.Key, envVarRegex)
			}
		}
	}

	if o.Config.Mount != nil {
		if o.Config.Mount.What == "" {
			return fmt.Errorf("What option for Mount section of systemd unit is required")
		}
		if o.Config.Mount.Where == "" {
			return fmt.Errorf("Where option for Mount section of systemd unit is required")
		}
	}

	return nil
}

func NewSystemdUnitCreateStage(options *SystemdUnitCreateStageOptions) *Stage {
	if err := options.validate(); err != nil {
		panic(err)
	}
	return &Stage{
		Type:    "org.osbuild.systemd.unit.create",
		Options: options,
	}
}
