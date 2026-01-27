package check_test

import (
	"errors"
	"io/fs"
	"os"
	"testing"

	"github.com/osbuild/blueprint/pkg/blueprint"
	check "github.com/osbuild/images/cmd/check-host-config/check"
	"github.com/osbuild/images/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirectoriesCheck(t *testing.T) {
	tests := []struct {
		name           string
		dirPath        string
		dirMode        fs.FileMode
		dirUid         uint32
		dirGid         uint32
		isDir          bool
		config         blueprint.DirectoryCustomization
		statError      error
		lookupUid      map[string]uint32 // map username -> uid for mocking
		lookupGid      map[string]uint32 // map groupname -> gid for mocking
		lookupUidError error
		lookupGidError error
		wantError      bool
		wantFail       bool
	}{
		{
			name:    "basic matching directory",
			dirPath: "/etc/testdir",
			dirMode: 0755,
			dirUid:  0,
			dirGid:  0,
			isDir:   true,
			config: blueprint.DirectoryCustomization{
				Path:  "/etc/testdir",
				Mode:  "0755",
				User:  int64(0),
				Group: int64(0),
			},
			wantError: false,
		},
		{
			name:    "matching directory with different uid/gid",
			dirPath: "/etc/testdir",
			dirMode: 0755,
			dirUid:  1000,
			dirGid:  1000,
			isDir:   true,
			config: blueprint.DirectoryCustomization{
				Path:  "/etc/testdir",
				Mode:  "0755",
				User:  int64(1000),
				Group: int64(1000),
			},
			wantError: false,
		},
		{
			name:    "matching directory without mode",
			dirPath: "/etc/testdir",
			dirMode: 0755,
			dirUid:  0,
			dirGid:  0,
			isDir:   true,
			config: blueprint.DirectoryCustomization{
				Path: "/etc/testdir",
			},
			wantError: false,
		},
		{
			name:    "non-matching mode",
			dirPath: "/etc/testdir",
			dirMode: 0755,
			dirUid:  0,
			dirGid:  0,
			isDir:   true,
			config: blueprint.DirectoryCustomization{
				Path:  "/etc/testdir",
				Mode:  "0700",
				User:  int64(0),
				Group: int64(0),
			},
			wantError: true,
			wantFail:  true,
		},
		{
			name:    "non-matching uid",
			dirPath: "/etc/testdir",
			dirMode: 0755,
			dirUid:  0,
			dirGid:  0,
			isDir:   true,
			config: blueprint.DirectoryCustomization{
				Path:  "/etc/testdir",
				Mode:  "0755",
				User:  int64(1000),
				Group: int64(0),
			},
			wantError: true,
			wantFail:  true,
		},
		{
			name:    "non-matching gid",
			dirPath: "/etc/testdir",
			dirMode: 0755,
			dirUid:  0,
			dirGid:  0,
			isDir:   true,
			config: blueprint.DirectoryCustomization{
				Path:  "/etc/testdir",
				Mode:  "0755",
				User:  int64(0),
				Group: int64(1000),
			},
			wantError: true,
			wantFail:  true,
		},
		{
			name:    "stat error",
			dirPath: "/etc/testdir",
			dirMode: 0755,
			dirUid:  0,
			dirGid:  0,
			isDir:   true,
			config: blueprint.DirectoryCustomization{
				Path:  "/etc/testdir",
				Mode:  "0755",
				User:  int64(0),
				Group: int64(0),
			},
			statError: errors.New("permission denied"),
			wantError: true,
			wantFail:  true,
		},
		{
			name:    "path is not a directory",
			dirPath: "/etc/testfile",
			dirMode: 0644,
			dirUid:  0,
			dirGid:  0,
			isDir:   false,
			config: blueprint.DirectoryCustomization{
				Path:  "/etc/testfile",
				Mode:  "0644",
				User:  int64(0),
				Group: int64(0),
			},
			wantError: true,
			wantFail:  true,
		},
		{
			name:    "matching directory with string username",
			dirPath: "/etc/testdir",
			dirMode: 0755,
			dirUid:  1000,
			dirGid:  1000,
			isDir:   true,
			config: blueprint.DirectoryCustomization{
				Path:  "/etc/testdir",
				Mode:  "0755",
				User:  "testuser",
				Group: "testgroup",
			},
			lookupUid: map[string]uint32{"testuser": 1000},
			lookupGid: map[string]uint32{"testgroup": 1000},
			wantError: false,
		},
		{
			name:    "non-matching string username",
			dirPath: "/etc/testdir",
			dirMode: 0755,
			dirUid:  0,
			dirGid:  0,
			isDir:   true,
			config: blueprint.DirectoryCustomization{
				Path:  "/etc/testdir",
				Mode:  "0755",
				User:  "testuser",
				Group: int64(0),
			},
			lookupUid: map[string]uint32{"testuser": 1000},
			wantError: true,
			wantFail:  true,
		},
		{
			name:    "non-matching string groupname",
			dirPath: "/etc/testdir",
			dirMode: 0755,
			dirUid:  0,
			dirGid:  0,
			isDir:   true,
			config: blueprint.DirectoryCustomization{
				Path:  "/etc/testdir",
				Mode:  "0755",
				User:  int64(0),
				Group: "testgroup",
			},
			lookupGid: map[string]uint32{"testgroup": 1000},
			wantError: true,
			wantFail:  true,
		},
		{
			name:    "lookup uid error",
			dirPath: "/etc/testdir",
			dirMode: 0755,
			dirUid:  0,
			dirGid:  0,
			isDir:   true,
			config: blueprint.DirectoryCustomization{
				Path:  "/etc/testdir",
				Mode:  "0755",
				User:  "nonexistent",
				Group: int64(0),
			},
			lookupUidError: errors.New("user not found"),
			wantError:      true,
			wantFail:       true,
		},
		{
			name:    "lookup gid error",
			dirPath: "/etc/testdir",
			dirMode: 0755,
			dirUid:  0,
			dirGid:  0,
			isDir:   true,
			config: blueprint.DirectoryCustomization{
				Path:  "/etc/testdir",
				Mode:  "0755",
				User:  int64(0),
				Group: "nonexistent",
			},
			lookupGidError: errors.New("group not found"),
			wantError:      true,
			wantFail:       true,
		},
		{
			name:    "directory does not exist",
			dirPath: "/etc/nonexistent",
			dirMode: 0755,
			dirUid:  0,
			dirGid:  0,
			isDir:   true,
			config: blueprint.DirectoryCustomization{
				Path:  "/etc/nonexistent",
				Mode:  "0755",
				User:  int64(0),
				Group: int64(0),
			},
			wantError: true,
			wantFail:  true,
		},
		{
			name:    "matching directory with mode 0700",
			dirPath: "/etc/testdir",
			dirMode: 0700,
			dirUid:  0,
			dirGid:  0,
			isDir:   true,
			config: blueprint.DirectoryCustomization{
				Path:  "/etc/testdir",
				Mode:  "0700",
				User:  int64(0),
				Group: int64(0),
			},
			wantError: false,
		},
		{
			name:    "matching directory with mode 0777",
			dirPath: "/etc/testdir",
			dirMode: 0777,
			dirUid:  0,
			dirGid:  0,
			isDir:   true,
			config: blueprint.DirectoryCustomization{
				Path:  "/etc/testdir",
				Mode:  "0777",
				User:  int64(0),
				Group: int64(0),
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test.MockGlobal(t, &check.ExistsDir, func(name string) bool {
				if name == tt.dirPath && tt.name != "directory does not exist" {
					return true
				}
				return false
			})

			test.MockGlobal(t, &check.Stat, func(name string) (os.FileInfo, error) {
				if name == tt.dirPath {
					if tt.statError != nil {
						return nil, tt.statError
					}
					return &mockFileInfo{
						name:  "testdir",
						mode:  tt.dirMode,
						uid:   tt.dirUid,
						gid:   tt.dirGid,
						isDir: tt.isDir,
					}, nil
				}
				return nil, errors.New("directory not found")
			})

			// Mock LookupUid if needed
			if tt.lookupUid != nil || tt.lookupUidError != nil {
				test.MockGlobal(t, &check.LookupUID, func(username string) (uint32, error) {
					if tt.lookupUidError != nil {
						return 0, tt.lookupUidError
					}
					if uid, ok := tt.lookupUid[username]; ok {
						return uid, nil
					}
					return 0, errors.New("user not found")
				})
			}

			// Mock LookupGid if needed
			if tt.lookupGid != nil || tt.lookupGidError != nil {
				test.MockGlobal(t, &check.LookupGID, func(groupname string) (uint32, error) {
					if tt.lookupGidError != nil {
						return 0, tt.lookupGidError
					}
					if gid, ok := tt.lookupGid[groupname]; ok {
						return gid, nil
					}
					return 0, errors.New("group not found")
				})
			}

			chk, found := check.FindCheckByName("Directories Check")
			require.True(t, found, "Directories Check not found")
			config := buildConfig(&blueprint.Customizations{
				Directories: []blueprint.DirectoryCustomization{tt.config},
			})

			err := chk.Func(chk.Meta, config)
			if tt.wantError {
				require.Error(t, err)
				if tt.wantFail {
					assert.True(t, check.IsFail(err))
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
