package check

import (
	"context"
	"strings"
	"time"

	"github.com/osbuild/images/cmd/check-host-config/mockos"
	"github.com/osbuild/images/internal/buildconfig"
)

type HostnameCheck struct{}

func (h HostnameCheck) Metadata() Metadata {
	return Metadata{
		Name:                   "Hostname Check",
		ShortName:              "hostname",
		Timeout:                5 * time.Second,
		RequiresBlueprint:      true,
		RequiresCustomizations: true,
	}
}

func (h HostnameCheck) Run(ctx context.Context, log Logger, config *buildconfig.BuildConfig) error {
	expected := config.Blueprint.Customizations.Hostname
	if expected == nil || *expected == "" {
		return Skip("no hostname customization")
	}

	out, _, err := mockos.ExecContext(ctx, log, "hostname")
	if err != nil {
		log.Printf("Failed to get hostname: %v\n", err)
		return err
	}

	hostname := strings.TrimSpace(string(out))
	log.Printf("Comparing '%s' with expected hostname '%s'\n", hostname, *expected)
	// we only emit a warning here since the hostname gets reset by cloud-init and we're not
	// entirely sure how to deal with it yet on the service level
	if hostname != *expected {
		return Warning("hostname does not match, got", hostname, "expected", *expected)
	}

	return Pass()
}
