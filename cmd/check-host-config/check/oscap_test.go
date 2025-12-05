package check_test

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/osbuild/blueprint/pkg/blueprint"
	check "github.com/osbuild/images/cmd/check-host-config/check"
	"github.com/osbuild/images/cmd/check-host-config/mockos"
	"github.com/osbuild/images/internal/buildconfig"
)

func TestOpenSCAPCheck(t *testing.T) {
	ctx := mockos.WithExistsFunc(context.Background(), func(name string) bool {
		return name == "results.xml"
	})

	ctx = mockos.WithExecFunc(ctx, func(name string, arg ...string) ([]byte, []byte, error) {
		if name == "sudo" && len(arg) >= 2 && arg[0] == "oscap" {
			// Mock oscap command - just return success
			return []byte(""), nil, nil
		}
		if name == "sudo" && len(arg) >= 2 && arg[0] == "chown" {
			// Mock chown command
			return []byte(""), nil, nil
		}
		if name == "xmlstarlet" && len(arg) >= 2 && arg[0] == "sel" {
			// Check which query is being made by looking at the last argument
			lastArg := arg[len(arg)-1]
			if lastArg == "results.xml" {
				// Check the query pattern (second to last arg)
				queryArg := arg[len(arg)-2]
				if queryArg == "//x:score" {
					// Return score query result (85.0 as percentage string)
					return []byte("85.0\n"), nil, nil
				}
				if queryArg == "//x:rule-result[@severity='high']" {
					// Return empty rule results (no failures)
					return []byte(""), nil, nil
				}
			}
		}
		return nil, nil, nil
	})

	chk := check.OpenSCAPCheck{}
	config := &buildconfig.BuildConfig{
		Blueprint: &blueprint.Blueprint{
			Customizations: &blueprint.Customizations{
				OpenSCAP: &blueprint.OpenSCAPCustomization{
					ProfileID:  "xccdf_org.ssgproject.content_profile_ospp",
					DataStream: "/usr/share/xml/scap/ssg/content/ssg-rhel9-ds.xml",
				},
			},
		},
	}

	err := chk.Run(ctx, log.New(os.Stdout, "", 0), config)
	if err != nil {
		t.Fatalf("OpenSCAPCheck failed: %v", err)
	}
}

func TestOpenSCAPCheckSkip(t *testing.T) {
	ctx := context.Background()
	chk := check.OpenSCAPCheck{}
	config := &buildconfig.BuildConfig{
		Blueprint: &blueprint.Blueprint{
			Customizations: &blueprint.Customizations{
				OpenSCAP: nil,
			},
		},
	}

	err := chk.Run(ctx, log.New(os.Stdout, "", 0), config)
	if err == nil {
		t.Fatal("OpenSCAPCheck should have skipped")
	}
	if !check.IsSkip(err) {
		t.Fatalf("OpenSCAPCheck should return Skip error, got: %v", err)
	}
}

func TestOpenSCAPCheckSkipIncomplete(t *testing.T) {
	ctx := context.Background()
	chk := check.OpenSCAPCheck{}
	config := &buildconfig.BuildConfig{
		Blueprint: &blueprint.Blueprint{
			Customizations: &blueprint.Customizations{
				OpenSCAP: &blueprint.OpenSCAPCustomization{
					ProfileID:  "",
					DataStream: "/usr/share/xml/scap/ssg/content/ssg-rhel9-ds.xml",
				},
			},
		},
	}

	err := chk.Run(ctx, log.New(os.Stdout, "", 0), config)
	if err == nil {
		t.Fatal("OpenSCAPCheck should have skipped")
	}
	if !check.IsSkip(err) {
		t.Fatalf("OpenSCAPCheck should return Skip error, got: %v", err)
	}
}

func TestOpenSCAPCheckFailNoResults(t *testing.T) {
	ctx := mockos.WithExistsFunc(context.Background(), func(name string) bool {
		// results.xml does not exist
		return false
	})

	ctx = mockos.WithExecFunc(ctx, func(name string, arg ...string) ([]byte, []byte, error) {
		if name == "sudo" && len(arg) >= 2 && arg[0] == "oscap" {
			// Mock oscap command failure
			return []byte("oscap error"), nil, nil
		}
		if name == "sudo" && len(arg) >= 2 && arg[0] == "chown" {
			return []byte(""), nil, nil
		}
		return nil, nil, nil
	})

	chk := check.OpenSCAPCheck{}
	config := &buildconfig.BuildConfig{
		Blueprint: &blueprint.Blueprint{
			Customizations: &blueprint.Customizations{
				OpenSCAP: &blueprint.OpenSCAPCustomization{
					ProfileID:  "xccdf_org.ssgproject.content_profile_ospp",
					DataStream: "/usr/share/xml/scap/ssg/content/ssg-rhel9-ds.xml",
				},
			},
		},
	}

	err := chk.Run(ctx, log.New(os.Stdout, "", 0), config)
	if err == nil {
		t.Fatal("OpenSCAPCheck should have failed")
	}
	if !check.IsFail(err) {
		t.Fatalf("OpenSCAPCheck should return Fail error, got: %v", err)
	}
}

func TestOpenSCAPCheckFailLowScore(t *testing.T) {
	ctx := mockos.WithExistsFunc(context.Background(), func(name string) bool {
		return name == "results.xml"
	})

	ctx = mockos.WithExecFunc(ctx, func(name string, arg ...string) ([]byte, []byte, error) {
		if name == "sudo" && len(arg) >= 2 && arg[0] == "oscap" {
			return []byte(""), nil, nil
		}
		if name == "sudo" && len(arg) >= 2 && arg[0] == "chown" {
			return []byte(""), nil, nil
		}
		if name == "xmlstarlet" && len(arg) >= 2 && arg[0] == "sel" {
			lastArg := arg[len(arg)-1]
			if lastArg == "results.xml" {
				queryArg := arg[len(arg)-2]
				if queryArg == "//x:score" {
					// Return low score (70.0, below baseline of 80.0)
					return []byte("70.0\n"), nil, nil
				}
				if queryArg == "//x:rule-result[@severity='high']" {
					return []byte(""), nil, nil
				}
			}
		}
		return nil, nil, nil
	})

	chk := check.OpenSCAPCheck{}
	config := &buildconfig.BuildConfig{
		Blueprint: &blueprint.Blueprint{
			Customizations: &blueprint.Customizations{
				OpenSCAP: &blueprint.OpenSCAPCustomization{
					ProfileID:  "xccdf_org.ssgproject.content_profile_ospp",
					DataStream: "/usr/share/xml/scap/ssg/content/ssg-rhel9-ds.xml",
				},
			},
		},
	}

	err := chk.Run(ctx, log.New(os.Stdout, "", 0), config)
	if err == nil {
		t.Fatal("OpenSCAPCheck should have failed")
	}
	if !check.IsFail(err) {
		t.Fatalf("OpenSCAPCheck should return Fail error, got: %v", err)
	}
}

func TestOpenSCAPCheckFailHighSeverityRules(t *testing.T) {
	ctx := mockos.WithExistsFunc(context.Background(), func(name string) bool {
		return name == "results.xml"
	})

	ctx = mockos.WithExecFunc(ctx, func(name string, arg ...string) ([]byte, []byte, error) {
		if name == "sudo" && len(arg) >= 2 && arg[0] == "oscap" {
			return []byte(""), nil, nil
		}
		if name == "sudo" && len(arg) >= 2 && arg[0] == "chown" {
			return []byte(""), nil, nil
		}
		if name == "xmlstarlet" && len(arg) >= 2 && arg[0] == "sel" {
			lastArg := arg[len(arg)-1]
			if lastArg == "results.xml" {
				queryArg := arg[len(arg)-2]
				if queryArg == "//x:score" {
					// Return good score (85.0)
					return []byte("85.0\n"), nil, nil
				}
				if queryArg == "//x:rule-result[@severity='high']" {
					// Return rule results with failures
					return []byte("rule1 result=fail\nrule2 result=pass\nrule3 result=fail\n"), nil, nil
				}
			}
		}
		return nil, nil, nil
	})

	chk := check.OpenSCAPCheck{}
	config := &buildconfig.BuildConfig{
		Blueprint: &blueprint.Blueprint{
			Customizations: &blueprint.Customizations{
				OpenSCAP: &blueprint.OpenSCAPCustomization{
					ProfileID:  "xccdf_org.ssgproject.content_profile_ospp",
					DataStream: "/usr/share/xml/scap/ssg/content/ssg-rhel9-ds.xml",
				},
			},
		},
	}

	err := chk.Run(ctx, log.New(os.Stdout, "", 0), config)
	if err == nil {
		t.Fatal("OpenSCAPCheck should have failed")
	}
	if !check.IsFail(err) {
		t.Fatalf("OpenSCAPCheck should return Fail error, got: %v", err)
	}
}

func TestOpenSCAPCheckFailExtractScore(t *testing.T) {
	ctx := mockos.WithExistsFunc(context.Background(), func(name string) bool {
		return name == "results.xml"
	})

	ctx = mockos.WithExecFunc(ctx, func(name string, arg ...string) ([]byte, []byte, error) {
		if name == "sudo" && len(arg) >= 2 && arg[0] == "oscap" {
			return []byte(""), nil, nil
		}
		if name == "sudo" && len(arg) >= 2 && arg[0] == "chown" {
			return []byte(""), nil, nil
		}
		if name == "xmlstarlet" && len(arg) >= 2 && arg[0] == "sel" {
			lastArg := arg[len(arg)-1]
			if lastArg == "results.xml" {
				queryArg := arg[len(arg)-2]
				if queryArg == "//x:score" {
					// Return error when extracting score
					return nil, nil, os.ErrNotExist
				}
			}
		}
		return nil, nil, nil
	})

	chk := check.OpenSCAPCheck{}
	config := &buildconfig.BuildConfig{
		Blueprint: &blueprint.Blueprint{
			Customizations: &blueprint.Customizations{
				OpenSCAP: &blueprint.OpenSCAPCustomization{
					ProfileID:  "xccdf_org.ssgproject.content_profile_ospp",
					DataStream: "/usr/share/xml/scap/ssg/content/ssg-rhel9-ds.xml",
				},
			},
		},
	}

	err := chk.Run(ctx, log.New(os.Stdout, "", 0), config)
	if err == nil {
		t.Fatal("OpenSCAPCheck should have failed")
	}
	if !check.IsFail(err) {
		t.Fatalf("OpenSCAPCheck should return Fail error, got: %v", err)
	}
}

func TestOpenSCAPCheckFailExtractRules(t *testing.T) {
	ctx := mockos.WithExistsFunc(context.Background(), func(name string) bool {
		return name == "results.xml"
	})

	ctx = mockos.WithExecFunc(ctx, func(name string, arg ...string) ([]byte, []byte, error) {
		if name == "sudo" && len(arg) >= 2 && arg[0] == "oscap" {
			return []byte(""), nil, nil
		}
		if name == "sudo" && len(arg) >= 2 && arg[0] == "chown" {
			return []byte(""), nil, nil
		}
		if name == "xmlstarlet" && len(arg) >= 2 && arg[0] == "sel" {
			lastArg := arg[len(arg)-1]
			if lastArg == "results.xml" {
				queryArg := arg[len(arg)-2]
				if queryArg == "//x:score" {
					// Return good score
					return []byte("85.0\n"), nil, nil
				}
				if queryArg == "//x:rule-result[@severity='high']" {
					// Return error when extracting rule results
					return nil, nil, os.ErrNotExist
				}
			}
		}
		return nil, nil, nil
	})

	chk := check.OpenSCAPCheck{}
	config := &buildconfig.BuildConfig{
		Blueprint: &blueprint.Blueprint{
			Customizations: &blueprint.Customizations{
				OpenSCAP: &blueprint.OpenSCAPCustomization{
					ProfileID:  "xccdf_org.ssgproject.content_profile_ospp",
					DataStream: "/usr/share/xml/scap/ssg/content/ssg-rhel9-ds.xml",
				},
			},
		},
	}

	err := chk.Run(ctx, log.New(os.Stdout, "", 0), config)
	if err == nil {
		t.Fatal("OpenSCAPCheck should have failed")
	}
	if !check.IsFail(err) {
		t.Fatalf("OpenSCAPCheck should return Fail error, got: %v", err)
	}
}

func TestOpenSCAPCheckNullDatastreamRHEL(t *testing.T) {
	// Create a temporary os-release file for testing
	tmpDir := t.TempDir()
	osReleasePath := filepath.Join(tmpDir, "os-release")
	osReleaseContent := `ID=rhel
VERSION_ID="9.0"
VERSION="9.0 (Plow)"
`
	err := os.WriteFile(osReleasePath, []byte(osReleaseContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create test os-release file: %v", err)
	}

	ctx := mockos.WithExistsFunc(context.Background(), func(name string) bool {
		return name == "results.xml"
	})

	// Mock ReadFile to return the test os-release file content when reading /etc/os-release
	ctx = mockos.WithReadFileFunc(ctx, func(filename string) ([]byte, error) {
		if filename == "/etc/os-release" {
			return os.ReadFile(osReleasePath)
		}
		return os.ReadFile(filename)
	})

	ctx = mockos.WithExecFunc(ctx, func(name string, arg ...string) ([]byte, []byte, error) {
		if name == "sudo" && len(arg) >= 2 && arg[0] == "oscap" {
			// Mock oscap command - just return success
			return []byte(""), nil, nil
		}
		if name == "sudo" && len(arg) >= 2 && arg[0] == "chown" {
			// Mock chown command
			return []byte(""), nil, nil
		}
		if name == "xmlstarlet" && len(arg) >= 2 && arg[0] == "sel" {
			lastArg := arg[len(arg)-1]
			if lastArg == "results.xml" {
				queryArg := arg[len(arg)-2]
				if queryArg == "//x:score" {
					// Return score query result (85.0 as percentage string)
					return []byte("85.0\n"), nil, nil
				}
				if queryArg == "//x:rule-result[@severity='high']" {
					// Return empty rule results (no failures)
					return []byte(""), nil, nil
				}
			}
		}
		return nil, nil, nil
	})

	chk := check.OpenSCAPCheck{}
	config := &buildconfig.BuildConfig{
		Blueprint: &blueprint.Blueprint{
			Customizations: &blueprint.Customizations{
				OpenSCAP: &blueprint.OpenSCAPCustomization{
					ProfileID:  "xccdf_org.ssgproject.content_profile_ospp",
					DataStream: "null", // null datastream should trigger fallback
				},
			},
		},
	}

	err = chk.Run(ctx, log.New(os.Stdout, "", 0), config)
	if err != nil {
		t.Fatalf("OpenSCAPCheck failed: %v", err)
	}
}
