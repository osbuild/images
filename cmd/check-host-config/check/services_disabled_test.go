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

func TestServicesDisabledCheck(t *testing.T) {
	ctx := mockos.WithExecFunc(context.Background(), func(name string, arg ...string) ([]byte, []byte, error) {
		if name == "systemctl" && len(arg) >= 2 && arg[0] == "is-enabled" && arg[1] == "test.service" {
			return []byte("disabled\n"), nil, nil
		}
		return nil, nil, nil
	})

	chk := check.ServicesDisabledCheck{}
	config := &buildconfig.BuildConfig{
		Blueprint: &blueprint.Blueprint{
			Customizations: &blueprint.Customizations{
				Services: &blueprint.ServicesCustomization{
					Disabled: []string{"test.service"},
				},
			},
		},
	}

	err := chk.Run(ctx, log.New(os.Stdout, "", 0), config)
	if err != nil {
		t.Fatalf("ServicesDisabledCheck failed: %v", err)
	}
}
