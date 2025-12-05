package check

import (
	"context"
	"strings"
	"time"

	"github.com/osbuild/images/cmd/check-host-config/mockos"
	"github.com/osbuild/images/internal/buildconfig"
)

type ServicesMaskedCheck struct{}

func (s ServicesMaskedCheck) Metadata() Metadata {
	return Metadata{
		Name:                   "Services Masked Check",
		ShortName:              "srv-masked",
		Timeout:                30 * time.Second,
		RequiresBlueprint:      true,
		RequiresCustomizations: true,
	}
}

func (s ServicesMaskedCheck) Run(ctx context.Context, log Logger, config *buildconfig.BuildConfig) error {
	services := config.Blueprint.Customizations.Services
	if services == nil || len(services.Masked) == 0 {
		return Skip("no masked services to check")
	}

	// Get list of masked services
	out, _, err := mockos.ExecContext(ctx, log, "systemctl", "list-unit-files", "--state=masked")
	if err != nil {
		return Fail("failed to list masked services:", err.Error())
	}

	maskedList := string(out)

	for _, service := range services.Masked {
		log.Printf("Checking masked service: %s\n", service)
		if !strings.Contains(maskedList, service) {
			return Fail("service is not masked:", service)
		}
		log.Printf("Service was masked service=%s\n", service)
	}

	return Pass()
}
