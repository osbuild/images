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

func TestServicesEnabledCheck(t *testing.T) {
	ctx := cos.WithExecFunc(context.Background(), func(name string, arg ...string) ([]byte, error) {
		if name == "systemctl" && len(arg) >= 2 && arg[0] == "is-enabled" && arg[1] == "test.service" {
			return []byte("enabled\n"), nil
		}
		return nil, nil
	})

	chk := check.ServicesEnabledCheck{}
	config := &buildconfig.BuildConfig{
		Blueprint: &blueprint.Blueprint{
			Customizations: &blueprint.Customizations{
				Services: &blueprint.ServicesCustomization{
					Enabled: []string{"test.service"},
				},
			},
		},
	}

	err := chk.Run(ctx, log.New(os.Stdout, "", 0), config)
	if err != nil {
		t.Fatalf("ServicesEnabledCheck failed: %v", err)
	}
}

func TestServicesEnabledCheckSkip(t *testing.T) {
	ctx := context.Background()
	chk := check.ServicesEnabledCheck{}
	config := &buildconfig.BuildConfig{
		Blueprint: &blueprint.Blueprint{
			Customizations: &blueprint.Customizations{
				Services: &blueprint.ServicesCustomization{
					Enabled: []string{},
				},
			},
		},
	}

	err := chk.Run(ctx, log.New(os.Stdout, "", 0), config)
	if err == nil {
		t.Fatal("ServicesEnabledCheck should have skipped")
	}
	if !check.IsSkip(err) {
		t.Fatalf("ServicesEnabledCheck should return Skip error, got: %v", err)
	}
}
