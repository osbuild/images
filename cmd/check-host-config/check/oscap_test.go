package check_test

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
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

	// Mock os-release for RHEL 9 (XCCDF 1.2 supported)
	osReleaseContent := `ID=rhel
VERSION_ID="9.0"
VERSION="9.0 (Plow)"
`

	// Mock XML file content with score 85.0 and no high severity failures (XCCDF 1.2 format)
	xmlContent := `<?xml version="1.0"?>
<Benchmark xmlns="http://checklists.nist.gov/xccdf/1.2">
	<TestResult>
		<score>85.0</score>
		<rule-result severity="high" idref="rule1">
			<result>pass</result>
		</rule-result>
		<rule-result severity="medium" idref="rule2">
			<result>fail</result>
		</rule-result>
	</TestResult>
</Benchmark>`

	ctx = mockos.WithReadFileFunc(ctx, func(filename string) ([]byte, error) {
		if filename == "results.xml" {
			return []byte(xmlContent), nil
		}
		if filename == "/etc/os-release" {
			return []byte(osReleaseContent), nil
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

	// Mock os-release for RHEL 9 (XCCDF 1.2 supported)
	osReleaseContent := `ID=rhel
VERSION_ID="9.0"
VERSION="9.0 (Plow)"
`

	// Mock XML file content with low score (70.0, below baseline of 80.0) (XCCDF 1.2 format)
	xmlContent := `<?xml version="1.0"?>
<Benchmark xmlns="http://checklists.nist.gov/xccdf/1.2">
	<TestResult>
		<score>70.0</score>
		<rule-result severity="high" idref="rule1">
			<result>pass</result>
		</rule-result>
	</TestResult>
</Benchmark>`

	ctx = mockos.WithReadFileFunc(ctx, func(filename string) ([]byte, error) {
		if filename == "results.xml" {
			return []byte(xmlContent), nil
		}
		if filename == "/etc/os-release" {
			return []byte(osReleaseContent), nil
		}
		return os.ReadFile(filename)
	})

	ctx = mockos.WithExecFunc(ctx, func(name string, arg ...string) ([]byte, []byte, error) {
		if name == "sudo" && len(arg) >= 2 && arg[0] == "oscap" {
			return []byte(""), nil, nil
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

func TestOpenSCAPCheckFailHighSeverityRules(t *testing.T) {
	ctx := mockos.WithExistsFunc(context.Background(), func(name string) bool {
		return name == "results.xml"
	})

	// Mock os-release for RHEL 9 (XCCDF 1.2 supported)
	osReleaseContent := `ID=rhel
VERSION_ID="9.0"
VERSION="9.0 (Plow)"
`

	// Mock XML file content with good score but high severity failures (XCCDF 1.2 format)
	xmlContent := `<?xml version="1.0"?>
<Benchmark xmlns="http://checklists.nist.gov/xccdf/1.2">
	<TestResult>
		<score>85.0</score>
		<rule-result severity="high" idref="rule1">
			<result>fail</result>
		</rule-result>
		<rule-result severity="high" idref="rule2">
			<result>pass</result>
		</rule-result>
		<rule-result severity="high" idref="rule3">
			<result>fail</result>
		</rule-result>
	</TestResult>
</Benchmark>`

	ctx = mockos.WithReadFileFunc(ctx, func(filename string) ([]byte, error) {
		if filename == "results.xml" {
			return []byte(xmlContent), nil
		}
		if filename == "/etc/os-release" {
			return []byte(osReleaseContent), nil
		}
		return os.ReadFile(filename)
	})

	ctx = mockos.WithExecFunc(ctx, func(name string, arg ...string) ([]byte, []byte, error) {
		if name == "sudo" && len(arg) >= 2 && arg[0] == "oscap" {
			return []byte(""), nil, nil
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

func TestOpenSCAPCheckIgnoreHighSeverityRules(t *testing.T) {
	ctx := mockos.WithExistsFunc(context.Background(), func(name string) bool {
		return name == "results.xml"
	})

	// Mock os-release for RHEL 9 (XCCDF 1.2 supported)
	osReleaseContent := `ID=rhel
VERSION_ID="9.0"
VERSION="9.0 (Plow)"
`

	// Mock XML file content with good score and ignored high severity failure (XCCDF 1.2 format)
	// The ignored rule should not cause the check to fail
	xmlContent := `<?xml version="1.0"?>
<Benchmark xmlns="http://checklists.nist.gov/xccdf/1.2">
	<TestResult>
		<score>85.0</score>
		<rule-result severity="high" idref="xccdf_org.ssgproject.content_rule_ensure_redhat_gpgkey_installed">
			<result>fail</result>
		</rule-result>
		<rule-result severity="high" idref="rule2">
			<result>pass</result>
		</rule-result>
	</TestResult>
</Benchmark>`

	ctx = mockos.WithReadFileFunc(ctx, func(filename string) ([]byte, error) {
		if filename == "results.xml" {
			return []byte(xmlContent), nil
		}
		if filename == "/etc/os-release" {
			return []byte(osReleaseContent), nil
		}
		return os.ReadFile(filename)
	})

	ctx = mockos.WithExecFunc(ctx, func(name string, arg ...string) ([]byte, []byte, error) {
		if name == "sudo" && len(arg) >= 2 && arg[0] == "oscap" {
			return []byte(""), nil, nil
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
	if err != nil {
		t.Fatalf("OpenSCAPCheck should pass when only ignored rules fail, got: %v", err)
	}
}

func TestOpenSCAPCheckIgnoreAndFailHighSeverityRules(t *testing.T) {
	ctx := mockos.WithExistsFunc(context.Background(), func(name string) bool {
		return name == "results.xml"
	})

	// Mock os-release for RHEL 9 (XCCDF 1.2 supported)
	osReleaseContent := `ID=rhel
VERSION_ID="9.0"
VERSION="9.0 (Plow)"
`

	// Mock XML file content with good score, one ignored high severity failure and one non-ignored failure
	// The check should fail because of the non-ignored rule, but the ignored rule should not appear in the error
	xmlContent := `<?xml version="1.0"?>
<Benchmark xmlns="http://checklists.nist.gov/xccdf/1.2">
	<TestResult>
		<score>85.0</score>
		<rule-result severity="high" idref="xccdf_org.ssgproject.content_rule_ensure_redhat_gpgkey_installed">
			<result>fail</result>
		</rule-result>
		<rule-result severity="high" idref="rule_non_ignored">
			<result>fail</result>
		</rule-result>
	</TestResult>
</Benchmark>`

	ctx = mockos.WithReadFileFunc(ctx, func(filename string) ([]byte, error) {
		if filename == "results.xml" {
			return []byte(xmlContent), nil
		}
		if filename == "/etc/os-release" {
			return []byte(osReleaseContent), nil
		}
		return os.ReadFile(filename)
	})

	ctx = mockos.WithExecFunc(ctx, func(name string, arg ...string) ([]byte, []byte, error) {
		if name == "sudo" && len(arg) >= 2 && arg[0] == "oscap" {
			return []byte(""), nil, nil
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
		t.Fatal("OpenSCAPCheck should have failed due to non-ignored rule")
	}
	if !check.IsFail(err) {
		t.Fatalf("OpenSCAPCheck should return Fail error, got: %v", err)
	}
}

func TestOpenSCAPCheckFailExtractScore(t *testing.T) {
	ctx := mockos.WithExistsFunc(context.Background(), func(name string) bool {
		return name == "results.xml"
	})

	// Mock os-release for RHEL 9 (XCCDF 1.2 supported)
	osReleaseContent := `ID=rhel
VERSION_ID="9.0"
VERSION="9.0 (Plow)"
`

	// Mock XML file content without score element (should fail parsing)
	xmlContent := `<?xml version="1.0"?>
<Benchmark xmlns="http://checklists.nist.gov/xccdf/1.2">
	<TestResult>
		<rule-result severity="high" idref="rule1">
			<result>pass</result>
		</rule-result>
	</TestResult>
</Benchmark>`

	ctx = mockos.WithReadFileFunc(ctx, func(filename string) ([]byte, error) {
		if filename == "results.xml" {
			return []byte(xmlContent), nil
		}
		if filename == "/etc/os-release" {
			return []byte(osReleaseContent), nil
		}
		return os.ReadFile(filename)
	})

	ctx = mockos.WithExecFunc(ctx, func(name string, arg ...string) ([]byte, []byte, error) {
		if name == "sudo" && len(arg) >= 2 && arg[0] == "oscap" {
			return []byte(""), nil, nil
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

func TestOpenSCAPCheckFailExtractRules(t *testing.T) {
	ctx := mockos.WithExistsFunc(context.Background(), func(name string) bool {
		return name == "results.xml"
	})

	// Mock os-release for RHEL 9 (XCCDF 1.2 supported)
	osReleaseContent := `ID=rhel
VERSION_ID="9.0"
VERSION="9.0 (Plow)"
`

	// Mock XML file content with invalid XML (should fail parsing)
	xmlContent := `<?xml version="1.0"?>
<Benchmark xmlns="http://checklists.nist.gov/xccdf/1.2">
	<TestResult>
		<score>85.0</score>
		<rule-result severity="high" idref="rule1">
			<result>fail</result>
		</rule-result>
	</TestResult>
</Benchmark>`

	ctx = mockos.WithReadFileFunc(ctx, func(filename string) ([]byte, error) {
		if filename == "results.xml" {
			return []byte(xmlContent), nil
		}
		if filename == "/etc/os-release" {
			return []byte(osReleaseContent), nil
		}
		return os.ReadFile(filename)
	})

	ctx = mockos.WithExecFunc(ctx, func(name string, arg ...string) ([]byte, []byte, error) {
		if name == "sudo" && len(arg) >= 2 && arg[0] == "oscap" {
			return []byte(""), nil, nil
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

	// Mock XML file content with score 85.0 and no high severity failures (XCCDF 1.2 format)
	xmlContent := `<?xml version="1.0"?>
<Benchmark xmlns="http://checklists.nist.gov/xccdf/1.2">
	<TestResult>
		<score>85.0</score>
		<rule-result severity="high" idref="rule1">
			<result>pass</result>
		</rule-result>
		<rule-result severity="medium" idref="rule2">
			<result>fail</result>
		</rule-result>
	</TestResult>
</Benchmark>`

	ctx = mockos.WithReadFileFunc(ctx, func(filename string) ([]byte, error) {
		if filename == "results.xml" {
			return []byte(xmlContent), nil
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

func TestOpenSCAPCheckSkipRHEL7(t *testing.T) {
	ctx := context.Background()

	// Mock os-release for RHEL 7 (should skip)
	osReleaseContent := `ID=rhel
VERSION_ID="7.9"
VERSION="7.9 (Maipo)"
`

	ctx = mockos.WithReadFileFunc(ctx, func(filename string) ([]byte, error) {
		if filename == "/etc/os-release" {
			return []byte(osReleaseContent), nil
		}
		return os.ReadFile(filename)
	})

	chk := check.OpenSCAPCheck{}
	config := &buildconfig.BuildConfig{
		Blueprint: &blueprint.Blueprint{
			Customizations: &blueprint.Customizations{
				OpenSCAP: &blueprint.OpenSCAPCustomization{
					ProfileID:  "xccdf_org.ssgproject.content_profile_ospp",
					DataStream: "/usr/share/xml/scap/ssg/content/ssg-rhel7-ds.xml",
				},
			},
		},
	}

	err := chk.Run(ctx, log.New(os.Stdout, "", 0), config)
	if err == nil {
		t.Fatal("OpenSCAPCheck should have skipped for RHEL 7")
	}
	if !check.IsSkip(err) {
		t.Fatalf("OpenSCAPCheck should return Skip error for RHEL < 8.0, got: %v", err)
	}
	if !strings.Contains(err.Error(), "only XCCDF 1.2 is supported") {
		t.Fatalf("Error message should mention XCCDF 1.2 requirement, got: %v", err)
	}
}

func TestOpenSCAPCheckFailNoTestResult(t *testing.T) {
	ctx := mockos.WithExistsFunc(context.Background(), func(name string) bool {
		return name == "results.xml"
	})

	// Mock os-release for RHEL 9 (XCCDF 1.2 supported)
	osReleaseContent := `ID=rhel
VERSION_ID="9.0"
VERSION="9.0 (Plow)"
`

	// Mock XML file content with Benchmark but no TestResult (should fail)
	xmlContent := `<?xml version="1.0"?>
<Benchmark xmlns="http://checklists.nist.gov/xccdf/1.2">
</Benchmark>`

	ctx = mockos.WithReadFileFunc(ctx, func(filename string) ([]byte, error) {
		if filename == "results.xml" {
			return []byte(xmlContent), nil
		}
		if filename == "/etc/os-release" {
			return []byte(osReleaseContent), nil
		}
		return os.ReadFile(filename)
	})

	ctx = mockos.WithExecFunc(ctx, func(name string, arg ...string) ([]byte, []byte, error) {
		if name == "sudo" && len(arg) >= 2 && arg[0] == "oscap" {
			return []byte(""), nil, nil
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
	if !strings.Contains(err.Error(), "expected exactly one test result") {
		t.Fatalf("Error message should mention expected exactly one test result, got: %v", err)
	}
}

func TestOpenSCAPCheckFailMultipleTestResults(t *testing.T) {
	ctx := mockos.WithExistsFunc(context.Background(), func(name string) bool {
		return name == "results.xml"
	})

	// Mock os-release for RHEL 9 (XCCDF 1.2 supported)
	osReleaseContent := `ID=rhel
VERSION_ID="9.0"
VERSION="9.0 (Plow)"
`

	// Mock XML file content with multiple TestResult elements (should fail)
	xmlContent := `<?xml version="1.0"?>
<Benchmark xmlns="http://checklists.nist.gov/xccdf/1.2">
	<TestResult>
		<score>85.0</score>
		<rule-result severity="high" idref="rule1">
			<result>pass</result>
		</rule-result>
	</TestResult>
	<TestResult>
		<score>90.0</score>
		<rule-result severity="high" idref="rule2">
			<result>pass</result>
		</rule-result>
	</TestResult>
</Benchmark>`

	ctx = mockos.WithReadFileFunc(ctx, func(filename string) ([]byte, error) {
		if filename == "results.xml" {
			return []byte(xmlContent), nil
		}
		if filename == "/etc/os-release" {
			return []byte(osReleaseContent), nil
		}
		return os.ReadFile(filename)
	})

	ctx = mockos.WithExecFunc(ctx, func(name string, arg ...string) ([]byte, []byte, error) {
		if name == "sudo" && len(arg) >= 2 && arg[0] == "oscap" {
			return []byte(""), nil, nil
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
	if !strings.Contains(err.Error(), "expected exactly one test result") {
		t.Fatalf("Error message should mention expected exactly one test result, got: %v", err)
	}
}
