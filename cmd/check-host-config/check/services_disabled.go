package check

import (
	"context"
	"strings"
	"time"

	"github.com/osbuild/images/cmd/check-host-config/cos"
	"github.com/osbuild/images/internal/buildconfig"
)

type ServicesDisabledCheck struct{}

func (s ServicesDisabledCheck) Metadata() Metadata {
	return Metadata{
		Name:                   "Services Disabled Check",
		ShortName:              "srv-disabled",
		Timeout:                30 * time.Second,
		RequiresBlueprint:      true,
		RequiresCustomizations: true,
	}
}

func (s ServicesDisabledCheck) Run(ctx context.Context, log Logger, config *buildconfig.BuildConfig) error {
	services := config.Blueprint.Customizations.Services
	if services == nil || len(services.Disabled) == 0 {
		return Skip("no disabled services to check")
	}

	for _, service := range services.Disabled {
		log.Printf("Checking disabled service: %s\n", service)
		out, err := cos.ExecContext(ctx, log, "systemctl", "is-enabled", service)
		// systemctl is-enabled returns non-zero exit code for disabled services,
		// but still outputs "disabled", so we check the output regardless of error
		state := strings.TrimSpace(string(out))
		if state == "" && err != nil {
			// If we got no output and an error, the service might not exist
			return Fail("service is not disabled:", service, "error:", err.Error())
		}

		if state != "disabled" {
			return Fail("service is not disabled:", service, "state:", state)
		}
		log.Printf("Service was disabled service=%s state=%s\n", service, state)
	}

	return Pass()
}
