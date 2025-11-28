package check

import (
	"context"
	"strings"
	"time"

	"github.com/osbuild/images/cmd/check-host-config/cos"
	"github.com/osbuild/images/internal/buildconfig"
)

type FirewallServicesDisabledCheck struct{}

func (f FirewallServicesDisabledCheck) Metadata() Metadata {
	return Metadata{
		Name:                   "Firewall Services Disabled Check",
		ShortName:              "fw-srv-disabled",
		Timeout:                30 * time.Second,
		RequiresBlueprint:      true,
		RequiresCustomizations: true,
	}
}

func (f FirewallServicesDisabledCheck) Run(ctx context.Context, log Logger, config *buildconfig.BuildConfig) error {
	firewall := config.Blueprint.Customizations.Firewall
	if firewall == nil || firewall.Services == nil || len(firewall.Services.Disabled) == 0 {
		return Skip("no disabled firewall services to check")
	}

	for _, service := range firewall.Services.Disabled {
		log.Printf("Checking disabled firewall service: %s\n", service)
		// NOTE: sudo works here without password because we test this only on ami
		// initialised with cloud-init, which sets sudo NOPASSWD for the user
		out, err := cos.ExecContext(ctx, log, "sudo", "firewall-cmd", "--query-service="+service)
		state := strings.TrimSpace(string(out))

		if state == "" && err != nil {
			return Fail("firewall service is not disabled:", service, "error:", err.Error())
		}

		if state != "no" {
			return Fail("firewall service is not disabled:", service, "state:", state)
		}

		log.Printf("Firewall service was disabled service=%s state=%s\n", service, state)
	}

	return Pass()
}
