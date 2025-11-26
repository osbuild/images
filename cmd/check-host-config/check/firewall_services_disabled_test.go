package check_test

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/osbuild/blueprint/pkg/blueprint"
	check "github.com/osbuild/images/cmd/check-host-config/check"
	"github.com/osbuild/images/cmd/check-host-config/cos"
	"github.com/osbuild/images/internal/buildconfig"
)

func TestFirewallServicesDisabledCheck(t *testing.T) {
	ctx := cos.WithExecFunc(context.Background(), func(name string, arg ...string) ([]byte, error) {
		if name == "sudo" && len(arg) >= 2 && arg[0] == "firewall-cmd" && arg[1] == "--query-service=badservice" {
			return []byte("no\n"), nil
		}
		return nil, nil
	})

	chk := check.FirewallServicesDisabledCheck{}
	config := &buildconfig.BuildConfig{
		Blueprint: &blueprint.Blueprint{
			Customizations: &blueprint.Customizations{
				Firewall: &blueprint.FirewallCustomization{
					Services: &blueprint.FirewallServicesCustomization{
						Disabled: []string{"badservice"},
					},
				},
			},
		},
	}

	err := chk.Run(ctx, log.New(os.Stdout, "", 0), config)
	if err != nil {
		t.Fatalf("FirewallServicesDisabledCheck failed: %v", err)
	}
}
