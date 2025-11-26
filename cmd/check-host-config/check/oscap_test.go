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

func TestOpenSCAPCheck(t *testing.T) {
	ctx := cos.WithExistsFunc(context.Background(), func(name string) bool {
		return name == "results.xml"
	})

	ctx = cos.WithExecFunc(ctx, func(name string, arg ...string) ([]byte, error) {
		if name == "sudo" && len(arg) >= 2 && arg[0] == "oscap" {
			// Mock oscap command - just return success
			return []byte(""), nil
		}
		if name == "sudo" && len(arg) >= 2 && arg[0] == "chown" {
			// Mock chown command
			return []byte(""), nil
		}
		if name == "xmlstarlet" && len(arg) >= 2 && arg[0] == "sel" {
			// Check which query is being made by looking at the last argument
			lastArg := arg[len(arg)-1]
			if lastArg == "results.xml" {
				// Check the query pattern (second to last arg)
				queryArg := arg[len(arg)-2]
				if queryArg == "//x:score" {
					// Return score query result (85.0 as percentage string)
					return []byte("85.0\n"), nil
				}
				if queryArg == "//x:rule-result[@severity='high']" {
					// Return empty rule results (no failures)
					return []byte(""), nil
				}
			}
		}
		return nil, nil
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
	ctx := cos.WithExistsFunc(context.Background(), func(name string) bool {
		// results.xml does not exist
		return false
	})

	ctx = cos.WithExecFunc(ctx, func(name string, arg ...string) ([]byte, error) {
		if name == "sudo" && len(arg) >= 2 && arg[0] == "oscap" {
			// Mock oscap command failure
			return []byte("oscap error"), nil
		}
		if name == "sudo" && len(arg) >= 2 && arg[0] == "chown" {
			return []byte(""), nil
		}
		return nil, nil
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
	ctx := cos.WithExistsFunc(context.Background(), func(name string) bool {
		return name == "results.xml"
	})

	ctx = cos.WithExecFunc(ctx, func(name string, arg ...string) ([]byte, error) {
		if name == "sudo" && len(arg) >= 2 && arg[0] == "oscap" {
			return []byte(""), nil
		}
		if name == "sudo" && len(arg) >= 2 && arg[0] == "chown" {
			return []byte(""), nil
		}
		if name == "xmlstarlet" && len(arg) >= 2 && arg[0] == "sel" {
			lastArg := arg[len(arg)-1]
			if lastArg == "results.xml" {
				queryArg := arg[len(arg)-2]
				if queryArg == "//x:score" {
					// Return low score (70.0, below baseline of 80.0)
					return []byte("70.0\n"), nil
				}
				if queryArg == "//x:rule-result[@severity='high']" {
					return []byte(""), nil
				}
			}
		}
		return nil, nil
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
	ctx := cos.WithExistsFunc(context.Background(), func(name string) bool {
		return name == "results.xml"
	})

	ctx = cos.WithExecFunc(ctx, func(name string, arg ...string) ([]byte, error) {
		if name == "sudo" && len(arg) >= 2 && arg[0] == "oscap" {
			return []byte(""), nil
		}
		if name == "sudo" && len(arg) >= 2 && arg[0] == "chown" {
			return []byte(""), nil
		}
		if name == "xmlstarlet" && len(arg) >= 2 && arg[0] == "sel" {
			lastArg := arg[len(arg)-1]
			if lastArg == "results.xml" {
				queryArg := arg[len(arg)-2]
				if queryArg == "//x:score" {
					// Return good score (85.0)
					return []byte("85.0\n"), nil
				}
				if queryArg == "//x:rule-result[@severity='high']" {
					// Return rule results with failures
					return []byte("rule1 result=fail\nrule2 result=pass\nrule3 result=fail\n"), nil
				}
			}
		}
		return nil, nil
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
	ctx := cos.WithExistsFunc(context.Background(), func(name string) bool {
		return name == "results.xml"
	})

	ctx = cos.WithExecFunc(ctx, func(name string, arg ...string) ([]byte, error) {
		if name == "sudo" && len(arg) >= 2 && arg[0] == "oscap" {
			return []byte(""), nil
		}
		if name == "sudo" && len(arg) >= 2 && arg[0] == "chown" {
			return []byte(""), nil
		}
		if name == "xmlstarlet" && len(arg) >= 2 && arg[0] == "sel" {
			lastArg := arg[len(arg)-1]
			if lastArg == "results.xml" {
				queryArg := arg[len(arg)-2]
				if queryArg == "//x:score" {
					// Return error when extracting score
					return nil, os.ErrNotExist
				}
			}
		}
		return nil, nil
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
	ctx := cos.WithExistsFunc(context.Background(), func(name string) bool {
		return name == "results.xml"
	})

	ctx = cos.WithExecFunc(ctx, func(name string, arg ...string) ([]byte, error) {
		if name == "sudo" && len(arg) >= 2 && arg[0] == "oscap" {
			return []byte(""), nil
		}
		if name == "sudo" && len(arg) >= 2 && arg[0] == "chown" {
			return []byte(""), nil
		}
		if name == "xmlstarlet" && len(arg) >= 2 && arg[0] == "sel" {
			lastArg := arg[len(arg)-1]
			if lastArg == "results.xml" {
				queryArg := arg[len(arg)-2]
				if queryArg == "//x:score" {
					// Return good score
					return []byte("85.0\n"), nil
				}
				if queryArg == "//x:rule-result[@severity='high']" {
					// Return error when extracting rule results
					return nil, os.ErrNotExist
				}
			}
		}
		return nil, nil
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
