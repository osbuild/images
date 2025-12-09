package check

import (
	"github.com/osbuild/images/internal/buildconfig"
)

func init() {
	RegisterCheck(Metadata{
		Name:                   "Hostname Check",
		ShortName:              "hostname",
		RequiresBlueprint:      true,
		RequiresCustomizations: true,
	}, hostnameCheck)
}

func hostnameCheck(meta *Metadata, config *buildconfig.BuildConfig) error {
	expected := config.Blueprint.Customizations.Hostname
	if expected == nil || *expected == "" {
		return Skip("no hostname customization")
	}

	hostname, _, _, err := ExecString("hostname")
	if err != nil {
		return err
	}

	// we only emit a warning here since the hostname gets reset by cloud-init and we're not
	// entirely sure how to deal with it yet on the service level
	if hostname != *expected {
		return Warning("hostname does not match, got", hostname, "expected", *expected)
	}

	return Pass()
}
