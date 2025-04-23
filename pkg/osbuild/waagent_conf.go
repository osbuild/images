package osbuild

type WAAgentConfig struct {
	ProvisioningUseCloudInit *bool `json:"Provisioning.UseCloudInit,omitempty"`
	ProvisioningEnabled      *bool `json:"Provisioning.Enabled,omitempty"`
	// XXX: yaml/json inconsistent :(
	RDFormat *bool `json:"ResourceDisk.Format,omitempty" yaml:"rd_format"`
	// XXX: yaml/json inconsistent :(
	RDEnableSwap *bool `json:"ResourceDisk.EnableSwap,omitempty" yaml:"rd_enable_swap"`
}

type WAAgentConfStageOptions struct {
	Config WAAgentConfig `json:"config"`
}

func (WAAgentConfStageOptions) isStageOptions() {}

func NewWAAgentConfStage(options *WAAgentConfStageOptions) *Stage {
	return &Stage{
		Type:    "org.osbuild.waagent.conf",
		Options: options,
	}
}
