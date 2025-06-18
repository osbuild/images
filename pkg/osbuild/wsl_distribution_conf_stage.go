package osbuild

type WSLDistributionConfStageOptions struct {
	OOBE     WSLDistributionConfOOBEOptions     `json:"oobe,omitempty"`
	Shortcut WSLDistributionConfShortcutOptions `json:"shortcut,omitempty"`
}

type WSLDistributionConfOOBEOptions struct {
	DefaultUID  *int   `json:"default_uid,omitempty"`
	DefaultName string `json:"default_name,omitempty"`
}

type WSLDistributionConfShortcutOptions struct {
	Enabled bool   `json:"enabled,omitempty"`
	Icon    string `json:"icon,omitempty"`
}

func (WSLDistributionConfStageOptions) isStageOptions() {}

func NewWSLDistributionConfStage(options *WSLDistributionConfStageOptions) *Stage {
	return &Stage{
		Type:    "org.osbuild.wsl-distribution.conf",
		Options: options,
	}
}
