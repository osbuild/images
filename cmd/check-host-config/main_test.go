package main

import (
	"errors"
	"os"
	"testing"

	"github.com/osbuild/blueprint/pkg/blueprint"
	"github.com/osbuild/images/cmd/check-host-config/check"
	"github.com/osbuild/images/internal/buildconfig"
)

// This is a happy-path smoke test that is supposed to be executed in a
// temporary Fedora container. It is ran on our CI/CD. To run it locally (in
// podman), execute `make host-check-test`.
//
//nolint:gosec // G303: Temporary files need to be consistently named
func TestSmokeAll(t *testing.T) {
	if os.Getenv("OSBUILD_TEST_CONTAINER") != "true" {
		t.Skip("Not running in container, skipping host check test")
	}

	// Prepare the container environment (cleanup not needed)
	if err := os.Mkdir("/tmp/dir", 0700); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	if err := os.WriteFile("/tmp/dir/file", []byte("data"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		chk  string
		name string
		c    blueprint.Customizations
	}{
		{
			chk:  "kernel",
			name: "params",
			c: blueprint.Customizations{
				Kernel: &blueprint.KernelCustomization{
					Name:   "kernel",
					Append: "root=",
				},
			},
		},
		{
			chk:  "kernel",
			name: "package",
			c: blueprint.Customizations{
				Kernel: &blueprint.KernelCustomization{
					Name: "kernel-debug",
				},
			},
		},
		{
			chk:  "directories",
			name: "all",
			c: blueprint.Customizations{
				Directories: []blueprint.DirectoryCustomization{
					{Path: "/tmp/dir"},
					{Path: "/tmp/dir", Mode: "0700"},
					{Path: "/tmp/dir", Mode: "0700", User: "root", Group: "root"},
					{Path: "/tmp/dir", Mode: "0700", User: 0, Group: 0},
				},
			},
		},
		{
			chk:  "files",
			name: "all",
			c: blueprint.Customizations{
				Files: []blueprint.FileCustomization{
					{Path: "/tmp/dir/file"},
					{Path: "/tmp/dir/file", Data: "data"},
					{Path: "/tmp/dir/file", Mode: "0600"},
					{Path: "/tmp/dir/file", Mode: "0600", User: "root", Group: "root"},
					{Path: "/tmp/dir/file", Mode: "0600", User: 0, Group: 0},
				},
			},
		},
		{
			chk:  "users",
			name: "root",
			c: blueprint.Customizations{
				User: []blueprint.UserCustomization{
					{Name: "root"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.chk+"/"+tt.name, func(t *testing.T) {
			chk := check.MustFindCheckByName(tt.chk)
			config := &buildconfig.BuildConfig{
				Blueprint: &blueprint.Blueprint{
					Customizations: &tt.c,
				},
			}
			err := chk.Func(chk.Meta, config)
			if errors.Is(err, check.ErrCheckSkipped) {
				t.Logf("Check %s skipped", chk.Meta.Name)
				return
			} else if err != nil {
				t.Fatalf("Check %s failed: %v", chk.Meta.Name, err)
			}
		})
	}
}
