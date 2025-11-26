package check_test

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/osbuild/blueprint/pkg/blueprint"
	check "github.com/osbuild/images/cmd/check-host-config/check"
	"github.com/osbuild/images/cmd/check-host-config/mockos"
	"github.com/osbuild/images/internal/buildconfig"
)

func TestFirewallServicesEnabledCheck(t *testing.T) {
	ctx := mockos.WithExecFunc(context.Background(), func(name string, arg ...string) ([]byte, []byte, error) {
		if name == "sudo" && len(arg) >= 2 && arg[0] == "firewall-cmd" && arg[1] == "--query-service=ssh" {
			return []byte("yes\n"), nil, nil
		}
		return nil, nil, nil
	})

	chk := check.FirewallServicesEnabledCheck{}
	config := &buildconfig.BuildConfig{
		Blueprint: &blueprint.Blueprint{
			Customizations: &blueprint.Customizations{
				Firewall: &blueprint.FirewallCustomization{
					Services: &blueprint.FirewallServicesCustomization{
						Enabled: []string{"ssh"},
					},
				},
			},
		},
	}

	err := chk.Run(ctx, log.New(os.Stdout, "", 0), config)
	if err != nil {
		t.Fatalf("FirewallServicesEnabledCheck failed: %v", err)
	}
}
