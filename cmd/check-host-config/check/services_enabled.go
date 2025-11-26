package check

import (
	"context"
	"strings"
	"time"

	"github.com/osbuild/images/cmd/check-host-config/mockos"
	"github.com/osbuild/images/internal/buildconfig"
)

type ServicesEnabledCheck struct{}

func (s ServicesEnabledCheck) Metadata() Metadata {
	return Metadata{
		Name:                   "Services Enabled Check",
		ShortName:              "srv-enabled",
		Timeout:                30 * time.Second,
		RequiresBlueprint:      true,
		RequiresCustomizations: true,
	}
}

func (s ServicesEnabledCheck) Run(ctx context.Context, log Logger, config *buildconfig.BuildConfig) error {
	services := config.Blueprint.Customizations.Services
	if services == nil || len(services.Enabled) == 0 {
		return Skip("no enabled services to check")
	}

	for _, service := range services.Enabled {
		log.Printf("Checking enabled service: %s\n", service)
		out, _, err := mockos.ExecContext(ctx, log, "systemctl", "is-enabled", service)
		if err != nil {
			return Fail("service is not enabled:", service, "error:", err.Error())
		}

		state := strings.TrimSpace(string(out))
		if state != "enabled" {
			return Fail("service is not enabled:", service, "state:", state)
		}
		log.Printf("Service was enabled service=%s state=%s\n", service, state)
	}

	return Pass()
}
