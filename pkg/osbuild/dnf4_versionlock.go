package osbuild

import (
	"fmt"
)

const dnf4VersionlockType = "org.osbuild.dnf4.versionlock"

type DNF4VersionlockOptions struct {
	Add []string `json:"add"`
}

func (*DNF4VersionlockOptions) isStageOptions() {}

func (o *DNF4VersionlockOptions) validate() error {
	if len(o.Add) == 0 {
		return fmt.Errorf("%s: at least one package must be included in the 'add' list", dnf4VersionlockType)
	}

	return nil
}

func NewDNF4VersionlockStage(options *DNF4VersionlockOptions) *Stage {
	if err := options.validate(); err != nil {
		panic(err)
	}
	return &Stage{
		Type:    dnf4VersionlockType,
		Options: options,
	}
}
