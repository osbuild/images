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

func TestUsersCheck(t *testing.T) {
	ctx := cos.WithExecFunc(context.Background(), func(name string, arg ...string) ([]byte, error) {
		if name == "id" && len(arg) > 0 && arg[0] == "testuser" {
			return []byte("uid=1000(testuser) gid=1000(testuser) groups=1000(testuser)\n"), nil
		}
		return nil, nil
	})

	chk := check.UsersCheck{}
	config := &buildconfig.BuildConfig{
		Blueprint: &blueprint.Blueprint{
			Customizations: &blueprint.Customizations{
				User: []blueprint.UserCustomization{
					{Name: "testuser"},
				},
			},
		},
	}

	err := chk.Run(ctx, log.New(os.Stdout, "", 0), config)
	if err != nil {
		t.Fatalf("UsersCheck failed: %v", err)
	}
}

func TestUsersCheckSkip(t *testing.T) {
	ctx := context.Background()
	chk := check.UsersCheck{}
	config := &buildconfig.BuildConfig{
		Blueprint: &blueprint.Blueprint{
			Customizations: &blueprint.Customizations{
				User: []blueprint.UserCustomization{},
			},
		},
	}

	err := chk.Run(ctx, log.New(os.Stdout, "", 0), config)
	if err == nil {
		t.Fatal("UsersCheck should have skipped")
	}
	if !check.IsSkip(err) {
		t.Fatalf("UsersCheck should return Skip error, got: %v", err)
	}
}
