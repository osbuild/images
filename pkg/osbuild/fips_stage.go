package osbuild

import (
	"strings"
)

type FIPSStageOptions struct {
	BootCfg bool `json:"boot_cfg"`
}

func (FIPSStageOptions) isStageOptions() {}

// NewFIPSStage creates FIPSStage
func NewFIPSStage(options *FIPSStageOptions) *Stage {
	return &Stage{
		Type:    "org.osbuild.fips",
		Options: options,
	}
}

func ContainsFIPSKernelOption(kernelOpts []string) bool {
	for _, kernelOption := range kernelOpts {
		if strings.Contains(kernelOption, "fips=1") {
			return true
		}
	}
	return false
}
