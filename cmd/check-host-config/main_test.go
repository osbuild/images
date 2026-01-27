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
//nolint:gosec // G303: Temporary files need to be consitently named
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
		chk check.RegisteredCheck
		c   blueprint.Customizations
	}{
		{
			chk: check.MustFindCheckByName("Kernel Check"),
			c: blueprint.Customizations{
				Kernel: &blueprint.KernelCustomization{
					Name:   "kernel",
					Append: "root=",
				},
			},
		},
		{
			chk: check.MustFindCheckByName("Kernel Check"),
			c: blueprint.Customizations{
				Kernel: &blueprint.KernelCustomization{
					Name: "kernel-debug",
				},
			},
		},
		{
			chk: check.MustFindCheckByName("Directories Check"),
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
			chk: check.MustFindCheckByName("Files Check"),
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
			chk: check.MustFindCheckByName("Users Check"),
			c: blueprint.Customizations{
				User: []blueprint.UserCustomization{
					{Name: "root"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.chk.Meta.Name, func(t *testing.T) {
			config := &buildconfig.BuildConfig{
				Blueprint: &blueprint.Blueprint{
					Customizations: &tt.c,
				},
			}
			err := tt.chk.Func(tt.chk.Meta, config)
			if errors.Is(err, check.ErrCheckSkipped) {
				t.Logf("Check %s skipped", tt.chk.Meta.Name)
				return
			} else if err != nil {
				t.Fatalf("Check %s failed: %v", tt.chk.Meta.Name, err)
			}
		})
	}
}
