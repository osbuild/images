package osbuild

import (
	"fmt"
	"slices"
	"strings"
)

type HMACStageOptions struct {
	Paths     []string `json:"paths"`
	Algorithm string   `json:"algorithm"`
}

func (o *HMACStageOptions) isStageOptions() {}

func (o *HMACStageOptions) validate() error {
	if len(o.Paths) == 0 {
		return fmt.Errorf("'paths' is a required property")
	}
	if o.Algorithm == "" {
		return fmt.Errorf("'algorithm' is a required property")
	}

	algorithms := []string{
		"sha1",
		"sha224",
		"sha256",
		"sha384",
		"sha512",
	}

	if !slices.Contains(algorithms, o.Algorithm) {
		return fmt.Errorf("'%s' is not one of [%s]", o.Algorithm, strings.Join(algorithms, ", "))
	}

	return nil
}

func NewHMACStage(options *HMACStageOptions) *Stage {
	return &Stage{
		Type:    "org.osbuild.hmac",
		Options: options,
	}
}
