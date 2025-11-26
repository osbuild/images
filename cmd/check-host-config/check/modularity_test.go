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

func TestModularityCheck(t *testing.T) {
	ctx := cos.WithExecFunc(context.Background(), func(name string, arg ...string) ([]byte, error) {
		if name == "dnf" && len(arg) >= 2 && arg[0] == "module" && arg[1] == "list" {
			return []byte("Last metadata expiration check: 0:00:00 ago\n" +
				"Dependencies resolved.\n" +
				"Module Stream Profiles\n" +
				"nodejs           18        [d]       common [d], development, minimal, s2i\n" +
				"python39         3.9       [d]       build, common [d], devel, minimal\n" +
				"Hint: [d]efault, [e]nabled, [x]disabled, [i]nstalled, [a]ctive\n"), nil
		}
		return nil, nil
	})

	chk := check.ModularityCheck{}
	config := &buildconfig.BuildConfig{
		Blueprint: &blueprint.Blueprint{
			EnabledModules: []blueprint.EnabledModule{
				{Name: "nodejs", Stream: "18"},
			},
		},
	}

	err := chk.Run(ctx, log.New(os.Stdout, "", 0), config)
	if err != nil {
		t.Fatalf("ModularityCheck failed: %v", err)
	}
}

func TestModularityCheckSkip(t *testing.T) {
	ctx := context.Background()
	chk := check.ModularityCheck{}
	config := &buildconfig.BuildConfig{
		Blueprint: &blueprint.Blueprint{
			EnabledModules: []blueprint.EnabledModule{},
			Packages:       []blueprint.Package{},
		},
	}

	err := chk.Run(ctx, log.New(os.Stdout, "", 0), config)
	if err == nil {
		t.Fatal("ModularityCheck should have skipped")
	}
	if !check.IsSkip(err) {
		t.Fatalf("ModularityCheck should return Skip error, got: %v", err)
	}
}
