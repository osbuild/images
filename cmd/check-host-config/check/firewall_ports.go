package check

import (
	"context"
	"strings"
	"time"

	"github.com/osbuild/images/cmd/check-host-config/cos"
	"github.com/osbuild/images/internal/buildconfig"
)

type FirewallPortsCheck struct{}

func (f FirewallPortsCheck) Metadata() Metadata {
	return Metadata{
		Name:                   "Firewall Ports Check",
		ShortName:              "fw-ports",
		Timeout:                30 * time.Second,
		RequiresBlueprint:      true,
		RequiresCustomizations: true,
	}
}

func (f FirewallPortsCheck) Run(ctx context.Context, log Logger, config *buildconfig.BuildConfig) error {
	firewall := config.Blueprint.Customizations.Firewall
	if firewall == nil || len(firewall.Ports) == 0 {
		return Skip("no firewall ports to check")
	}

	for _, port := range firewall.Ports {
		// firewall-cmd --query-port uses / as the port/protocol separator, but
		// in the blueprint we use :.
		portQuery := strings.ReplaceAll(port, ":", "/")
		log.Printf("Checking enabled firewall port: %s\n", portQuery)
		// NOTE: sudo works here without password because we test this only on ami
		// initialised with cloud-init, which sets sudo NOPASSWD for the user
		out, err := cos.ExecContext(ctx, log, "sudo", "firewall-cmd", "--query-port="+portQuery)
		if err != nil {
			return Fail("firewall port is not enabled:", port, "error:", err.Error())
		}

		state := strings.TrimSpace(string(out))
		if state != "yes" {
			return Fail("firewall port is not enabled:", port, "state:", state)
		}
		log.Printf("Firewall port was enabled port=%s state=%s\n", portQuery, state)
	}

	return Pass()
}
