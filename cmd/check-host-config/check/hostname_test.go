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
	"github.com/osbuild/images/internal/common"
)

func TestHostnameCheck(t *testing.T) {
	ctx := mockos.WithExecFunc(context.Background(), func(name string, arg ...string) ([]byte, []byte, error) {
		return []byte("test-hostname\n"), nil, nil
	})

	check := check.HostnameCheck{}
	config := &buildconfig.BuildConfig{
		Blueprint: &blueprint.Blueprint{
			Customizations: &blueprint.Customizations{
				Hostname: common.ToPtr("test-hostname"),
			},
		},
	}

	err := check.Run(ctx, log.New(os.Stdout, "", 0), config)
	if err != nil {
		t.Fatalf("HostnameCheck failed: %v", err)
	}
}

func TestHostnameCheckWarning(t *testing.T) {
	ctx := mockos.WithExecFunc(context.Background(), func(name string, arg ...string) ([]byte, []byte, error) {
		return []byte("changed-by-cloud-init\n"), nil, nil
	})

	chk := check.HostnameCheck{}
	config := &buildconfig.BuildConfig{
		Blueprint: &blueprint.Blueprint{
			Customizations: &blueprint.Customizations{
				Hostname: common.ToPtr("test-hostname"),
			},
		},
	}

	err := chk.Run(ctx, log.New(os.Stdout, "", 0), config)
	if err == nil {
		t.Fatalf("HostnameCheckWarning should have returned a warning")
	}
	if !check.IsWarning(err) {
		t.Fatalf("HostnameCheckWarning should return Warning error, got: %v", err)
	}
}
