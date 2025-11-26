package check_test

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"testing"

	check "github.com/osbuild/images/cmd/check-host-config/check"
	"github.com/osbuild/images/cmd/check-host-config/mockos"
)

func TestParseOSRelease(t *testing.T) {
	// Create a temporary directory and file
	tmpDir := t.TempDir()
	osReleasePath := filepath.Join(tmpDir, "os-release")

	// Write a sample os-release file
	content := `NAME="Red Hat Enterprise Linux"
VERSION="9.0 (Plow)"
ID=rhel
ID_LIKE="fedora"
VERSION_ID="9.0"
PRETTY_NAME="Red Hat Enterprise Linux 9.0 (Plow)"
ANSI_COLOR="0;31"
CPE_NAME="cpe:/o:redhat:enterprise_linux:9::baseos"
HOME_URL="https://www.redhat.com/"
BUG_REPORT_URL="https://bugzilla.redhat.com/"
`
	err := os.WriteFile(osReleasePath, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create test os-release file: %v", err)
	}

	ctx := mockos.WithReadFileFunc(context.Background(), func(filename string) ([]byte, error) {
		return os.ReadFile(filename)
	})
	logger := log.New(os.Stdout, "", 0)
	release, err := check.ParseOSRelease(ctx, logger, osReleasePath)
	if err != nil {
		t.Fatalf("ParseOSRelease failed: %v", err)
	}

	if release.ID != "rhel" {
		t.Errorf("Expected ID='rhel', got '%s'", release.ID)
	}
	if release.VersionID != "9.0" {
		t.Errorf("Expected VersionID='9.0', got '%s'", release.VersionID)
	}
	if release.Version != "9.0 (Plow)" {
		t.Errorf("Expected Version='9.0 (Plow)', got '%s'", release.Version)
	}
}

func TestGetDatastreamFilename(t *testing.T) {
	tests := []struct {
		name     string
		release  *check.OSRelease
		expected string
		wantErr  bool
	}{
		{
			name: "RHEL 9",
			release: &check.OSRelease{
				ID:        "rhel",
				VersionID: "9.0",
			},
			expected: "/usr/share/xml/scap/ssg/content/ssg-rhel9-ds.xml",
			wantErr:  false,
		},
		{
			name: "RHEL 8",
			release: &check.OSRelease{
				ID:        "rhel",
				VersionID: "8.6",
			},
			expected: "/usr/share/xml/scap/ssg/content/ssg-rhel8-ds.xml",
			wantErr:  false,
		},
		{
			name: "CentOS 9",
			release: &check.OSRelease{
				ID:        "centos",
				VersionID: "9",
			},
			expected: "/usr/share/xml/scap/ssg/content/ssg-cs9-ds.xml",
			wantErr:  false,
		},
		{
			name: "CentOS 8",
			release: &check.OSRelease{
				ID:        "centos",
				VersionID: "8.5",
			},
			expected: "/usr/share/xml/scap/ssg/content/ssg-centos8-ds.xml",
			wantErr:  false,
		},
		{
			name: "Fedora",
			release: &check.OSRelease{
				ID:        "fedora",
				VersionID: "39",
			},
			expected: "/usr/share/xml/scap/ssg/content/ssg-fedora-ds.xml",
			wantErr:  false,
		},
		{
			name: "Unsupported OS",
			release: &check.OSRelease{
				ID:        "ubuntu",
				VersionID: "22.04",
			},
			wantErr: true,
		},
		{
			name: "Unsupported version",
			release: &check.OSRelease{
				ID:        "rhel",
				VersionID: "7",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename, err := check.GetDatastreamFilename(tt.release)
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetDatastreamFilename() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("GetDatastreamFilename() error = %v", err)
				return
			}
			if filename != tt.expected {
				t.Errorf("GetDatastreamFilename() = %v, want %v", filename, tt.expected)
			}
		})
	}
}
