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

func TestModularityCheck(t *testing.T) {
	ctx := mockos.WithExecFunc(context.Background(), func(name string, arg ...string) ([]byte, []byte, error) {
		if name == "dnf" && len(arg) >= 3 && arg[0] == "-q" && arg[1] == "module" && arg[2] == "list" {
			return []byte("Last metadata expiration check: 0:00:00 ago\n" +
				"Dependencies resolved.\n" +
				"Module Stream Profiles\n" +
				"nodejs           18        [d]       common [d], development, minimal, s2i\n" +
				"python39         3.9       [d]       build, common [d], devel, minimal\n" +
				"Hint: [d]efault, [e]nabled, [x]disabled, [i]nstalled, [a]ctive\n"), nil, nil
		}
		return nil, nil, nil
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

// Test fixtures for different RHEL versions
// These represent realistic variations in dnf module list --enabled output across versions

// RHEL 7 format (DNF was tech preview, output might be slightly different)
func dnfModuleListOutputRHEL7() string {
	return "Last metadata expiration check: 1:23:45 ago on Mon 01 Jan 2024 12:00:00 PM UTC.\n" +
		"Dependencies resolved.\n" +
		"Module Stream Profiles\n" +
		"nodejs           10        [e]       common [d], development, minimal\n" +
		"python36         3.6       [e]       build, common [d], devel\n" +
		"Hint: [d]efault, [e]nabled, [x]disabled, [i]nstalled, [a]ctive\n"
}

// RHEL 8 format (standard DNF output)
func dnfModuleListOutputRHEL8() string {
	return "Last metadata expiration check: 0:00:00 ago on Mon 01 Jan 2024 12:00:00 PM UTC.\n" +
		"Dependencies resolved.\n" +
		"Module Stream Profiles\n" +
		"nodejs           12        [e]       common [d], development, minimal, s2i\n" +
		"python38         3.8       [e]       build, common [d], devel, minimal\n" +
		"postgresql       12        [e]       client, server [d]\n" +
		"Hint: [d]efault, [e]nabled, [x]disabled, [i]nstalled, [a]ctive\n"
}

// RHEL 9 format (similar to RHEL 8, might have slight variations)
func dnfModuleListOutputRHEL9() string {
	return "Last metadata expiration check: 0:00:00 ago\n" +
		"Dependencies resolved.\n" +
		"Module Stream Profiles\n" +
		"nodejs           18        [d]       common [d], development, minimal, s2i\n" +
		"python39         3.9       [d]       build, common [d], devel, minimal\n" +
		"Hint: [d]efault, [e]nabled, [x]disabled, [i]nstalled, [a]ctive\n"
}

// RHEL 10 format (latest format, might have updated messages)
func dnfModuleListOutputRHEL10() string {
	return "Last metadata expiration check: 0:00:00 ago\n" +
		"Dependencies resolved.\n" +
		"Module Stream Profiles\n" +
		"nodejs           20        [e]       common [d], development, minimal, s2i\n" +
		"python312        3.12      [e]       build, common [d], devel, minimal\n" +
		"postgresql       16        [e]       client, server [d], devel\n" +
		"Use \"dnf module info <module:stream>\" to get more information.\n" +
		"Hint: [d]efault, [e]nabled, [x]disabled, [i]nstalled, [a]ctive\n"
}

// RHEL 9/10 format with multiple modules and different spacing
func dnfModuleListOutputMultiple() string {
	return "Last metadata expiration check: 0:00:00 ago\n" +
		"Dependencies resolved.\n" +
		"Module Stream Profiles\n" +
		"nodejs           18        [e]       common [d], development, minimal, s2i\n" +
		"python39         3.9       [e]       build, common [d], devel, minimal\n" +
		"postgresql       13        [e]       client, server [d]\n" +
		"ruby              3.1       [e]       common [d], devel\n" +
		"Hint: [d]efault, [e]nabled, [x]disabled, [i]nstalled, [a]ctive\n"
}

func TestModularityCheckRHEL7(t *testing.T) {
	ctx := mockos.WithExecFunc(context.Background(), func(name string, arg ...string) ([]byte, []byte, error) {
		if name == "dnf" && len(arg) >= 3 && arg[0] == "-q" && arg[1] == "module" && arg[2] == "list" {
			return []byte(dnfModuleListOutputRHEL7()), nil, nil
		}
		return nil, nil, nil
	})

	chk := check.ModularityCheck{}
	config := &buildconfig.BuildConfig{
		Blueprint: &blueprint.Blueprint{
			EnabledModules: []blueprint.EnabledModule{
				{Name: "nodejs", Stream: "10"},
				{Name: "python36", Stream: "3.6"},
			},
		},
	}

	err := chk.Run(ctx, log.New(os.Stdout, "", 0), config)
	if err != nil {
		t.Fatalf("ModularityCheck failed for RHEL 7: %v", err)
	}
}

func TestModularityCheckRHEL8(t *testing.T) {
	ctx := mockos.WithExecFunc(context.Background(), func(name string, arg ...string) ([]byte, []byte, error) {
		if name == "dnf" && len(arg) >= 3 && arg[0] == "-q" && arg[1] == "module" && arg[2] == "list" {
			return []byte(dnfModuleListOutputRHEL8()), nil, nil
		}
		return nil, nil, nil
	})

	chk := check.ModularityCheck{}
	config := &buildconfig.BuildConfig{
		Blueprint: &blueprint.Blueprint{
			EnabledModules: []blueprint.EnabledModule{
				{Name: "nodejs", Stream: "12"},
				{Name: "python38", Stream: "3.8"},
				{Name: "postgresql", Stream: "12"},
			},
		},
	}

	err := chk.Run(ctx, log.New(os.Stdout, "", 0), config)
	if err != nil {
		t.Fatalf("ModularityCheck failed for RHEL 8: %v", err)
	}
}

func TestModularityCheckRHEL9(t *testing.T) {
	ctx := mockos.WithExecFunc(context.Background(), func(name string, arg ...string) ([]byte, []byte, error) {
		if name == "dnf" && len(arg) >= 3 && arg[0] == "-q" && arg[1] == "module" && arg[2] == "list" {
			return []byte(dnfModuleListOutputRHEL9()), nil, nil
		}
		return nil, nil, nil
	})

	chk := check.ModularityCheck{}
	config := &buildconfig.BuildConfig{
		Blueprint: &blueprint.Blueprint{
			EnabledModules: []blueprint.EnabledModule{
				{Name: "nodejs", Stream: "18"},
				{Name: "python39", Stream: "3.9"},
			},
		},
	}

	err := chk.Run(ctx, log.New(os.Stdout, "", 0), config)
	if err != nil {
		t.Fatalf("ModularityCheck failed for RHEL 9: %v", err)
	}
}

func TestModularityCheckRHEL10(t *testing.T) {
	ctx := mockos.WithExecFunc(context.Background(), func(name string, arg ...string) ([]byte, []byte, error) {
		if name == "dnf" && len(arg) >= 3 && arg[0] == "-q" && arg[1] == "module" && arg[2] == "list" {
			return []byte(dnfModuleListOutputRHEL10()), nil, nil
		}
		return nil, nil, nil
	})

	chk := check.ModularityCheck{}
	config := &buildconfig.BuildConfig{
		Blueprint: &blueprint.Blueprint{
			EnabledModules: []blueprint.EnabledModule{
				{Name: "nodejs", Stream: "20"},
				{Name: "python312", Stream: "3.12"},
				{Name: "postgresql", Stream: "16"},
			},
		},
	}

	err := chk.Run(ctx, log.New(os.Stdout, "", 0), config)
	if err != nil {
		t.Fatalf("ModularityCheck failed for RHEL 10: %v", err)
	}
}

func TestModularityCheckMultipleModules(t *testing.T) {
	ctx := mockos.WithExecFunc(context.Background(), func(name string, arg ...string) ([]byte, []byte, error) {
		if name == "dnf" && len(arg) >= 3 && arg[0] == "-q" && arg[1] == "module" && arg[2] == "list" {
			return []byte(dnfModuleListOutputMultiple()), nil, nil
		}
		return nil, nil, nil
	})

	chk := check.ModularityCheck{}
	config := &buildconfig.BuildConfig{
		Blueprint: &blueprint.Blueprint{
			EnabledModules: []blueprint.EnabledModule{
				{Name: "nodejs", Stream: "18"},
				{Name: "python39", Stream: "3.9"},
				{Name: "postgresql", Stream: "13"},
				{Name: "ruby", Stream: "3.1"},
			},
		},
	}

	err := chk.Run(ctx, log.New(os.Stdout, "", 0), config)
	if err != nil {
		t.Fatalf("ModularityCheck failed for multiple modules: %v", err)
	}
}
