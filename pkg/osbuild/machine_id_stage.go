package osbuild

type MachineIdStageOptions struct {
	// Determines the state of `/etc/machine-id`, valid values are
	// `yes` (reset to `uninitialized`), `no` (empty), `preserve` (keep).
	FirstBoot string `json:"first-boot"`
}

func (MachineIdStageOptions) isStageOptions() {}

func NewMachineIdStageOptions(firstboot string) *MachineIdStageOptions {
	return &MachineIdStageOptions{
		FirstBoot: firstboot,
	}
}

func NewMachineIdStage(options *MachineIdStageOptions) *Stage {
	return &Stage{
		Type:    "org.osbuild.machine-id",
		Options: options,
	}
}
