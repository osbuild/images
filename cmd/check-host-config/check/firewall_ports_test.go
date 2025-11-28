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

func TestFirewallPortsCheck(t *testing.T) {
	ctx := cos.WithExecFunc(context.Background(), func(name string, arg ...string) ([]byte, error) {
		if name == "sudo" && len(arg) >= 2 && arg[0] == "firewall-cmd" && arg[1] == "--query-port=80/tcp" {
			return []byte("yes\n"), nil
		}
		return nil, nil
	})

	chk := check.FirewallPortsCheck{}
	config := &buildconfig.BuildConfig{
		Blueprint: &blueprint.Blueprint{
			Customizations: &blueprint.Customizations{
				Firewall: &blueprint.FirewallCustomization{
					Ports: []string{"80:tcp"},
				},
			},
		},
	}

	err := chk.Run(ctx, log.New(os.Stdout, "", 0), config)
	if err != nil {
		t.Fatalf("FirewallPortsCheck failed: %v", err)
	}
}
