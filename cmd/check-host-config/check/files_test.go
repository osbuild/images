package check_test

import (
	"errors"
	"io/fs"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/osbuild/blueprint/pkg/blueprint"
	check "github.com/osbuild/images/cmd/check-host-config/check"
	"github.com/osbuild/images/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFileInfo is a mock implementation of os.FileInfo for testing
type mockFileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
	uid     uint32
	gid     uint32
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return m.size }
func (m *mockFileInfo) Mode() fs.FileMode  { return m.mode }
func (m *mockFileInfo) ModTime() time.Time { return m.modTime }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Sys() any {
	return &syscall.Stat_t{
		Uid: m.uid,
		Gid: m.gid,
	}
}

func TestFilesCheck(t *testing.T) {
	tests := []struct {
		name           string
		filePath       string
		fileMode       fs.FileMode
		fileUid        uint32
		fileGid        uint32
		config         blueprint.FileCustomization
		statError      error
		readFileData   []byte
		readFileError  error
		lookupUid      map[string]uint32 // map username -> uid for mocking
		lookupGid      map[string]uint32 // map groupname -> gid for mocking
		lookupUidError error
		lookupGidError error
		wantError      bool
		wantFail       bool
	}{
		{
			name:     "basic matching file",
			filePath: "/etc/testfile",
			fileMode: 0644,
			fileUid:  0,
			fileGid:  0,
			config: blueprint.FileCustomization{
				Path:  "/etc/testfile",
				Mode:  "0644",
				User:  int64(0),
				Group: int64(0),
			},
			wantError: false,
		},
		{
			name:     "matching file with different uid/gid",
			filePath: "/etc/testfile",
			fileMode: 0644,
			fileUid:  1000,
			fileGid:  1000,
			config: blueprint.FileCustomization{
				Path:  "/etc/testfile",
				Mode:  "0644",
				User:  int64(1000),
				Group: int64(1000),
			},
			wantError: false,
		},
		{
			name:     "matching file with content",
			filePath: "/etc/testfile",
			fileMode: 0644,
			fileUid:  0,
			fileGid:  0,
			config: blueprint.FileCustomization{
				Path:  "/etc/testfile",
				Mode:  "0644",
				User:  int64(0),
				Group: int64(0),
				Data:  "test content",
			},
			readFileData: []byte("test content"),
			wantError:    false,
		},
		{
			name:     "non-matching mode",
			filePath: "/etc/testfile",
			fileMode: 0644,
			fileUid:  0,
			fileGid:  0,
			config: blueprint.FileCustomization{
				Path:  "/etc/testfile",
				Mode:  "0755",
				User:  int64(0),
				Group: int64(0),
			},
			wantError: true,
			wantFail:  true,
		},
		{
			name:     "non-matching uid",
			filePath: "/etc/testfile",
			fileMode: 0644,
			fileUid:  0,
			fileGid:  0,
			config: blueprint.FileCustomization{
				Path:  "/etc/testfile",
				Mode:  "0644",
				User:  int64(1000),
				Group: int64(0),
			},
			wantError: true,
			wantFail:  true,
		},
		{
			name:     "non-matching gid",
			filePath: "/etc/testfile",
			fileMode: 0644,
			fileUid:  0,
			fileGid:  0,
			config: blueprint.FileCustomization{
				Path:  "/etc/testfile",
				Mode:  "0644",
				User:  int64(0),
				Group: int64(1000),
			},
			wantError: true,
			wantFail:  true,
		},
		{
			name:     "non-matching content",
			filePath: "/etc/testfile",
			fileMode: 0644,
			fileUid:  0,
			fileGid:  0,
			config: blueprint.FileCustomization{
				Path:  "/etc/testfile",
				Mode:  "0644",
				User:  int64(0),
				Group: int64(0),
				Data:  "test content",
			},
			readFileData: []byte("different content"),
			wantError:    true,
			wantFail:     true,
		},
		{
			name:     "read file error",
			filePath: "/etc/testfile",
			fileMode: 0644,
			fileUid:  0,
			fileGid:  0,
			config: blueprint.FileCustomization{
				Path:  "/etc/testfile",
				Mode:  "0644",
				User:  int64(0),
				Group: int64(0),
				Data:  "test content",
			},
			readFileError: errors.New("permission denied"),
			wantError:     true,
			wantFail:      true,
		},
		{
			name:     "stat error",
			filePath: "/etc/testfile",
			fileMode: 0644,
			fileUid:  0,
			fileGid:  0,
			config: blueprint.FileCustomization{
				Path:  "/etc/testfile",
				Mode:  "0644",
				User:  int64(0),
				Group: int64(0),
			},
			statError: errors.New("permission denied"),
			wantError: true,
			wantFail:  true,
		},
		{
			name:     "matching file with string username",
			filePath: "/etc/testfile",
			fileMode: 0644,
			fileUid:  1000,
			fileGid:  1000,
			config: blueprint.FileCustomization{
				Path:  "/etc/testfile",
				Mode:  "0644",
				User:  "testuser",
				Group: "testgroup",
			},
			lookupUid: map[string]uint32{"testuser": 1000},
			lookupGid: map[string]uint32{"testgroup": 1000},
			wantError: false,
		},
		{
			name:     "non-matching string username",
			filePath: "/etc/testfile",
			fileMode: 0644,
			fileUid:  0,
			fileGid:  0,
			config: blueprint.FileCustomization{
				Path:  "/etc/testfile",
				Mode:  "0644",
				User:  "testuser",
				Group: int64(0),
			},
			lookupUid: map[string]uint32{"testuser": 1000},
			wantError: true,
			wantFail:  true,
		},
		{
			name:     "non-matching string groupname",
			filePath: "/etc/testfile",
			fileMode: 0644,
			fileUid:  0,
			fileGid:  0,
			config: blueprint.FileCustomization{
				Path:  "/etc/testfile",
				Mode:  "0644",
				User:  int64(0),
				Group: "testgroup",
			},
			lookupGid: map[string]uint32{"testgroup": 1000},
			wantError: true,
			wantFail:  true,
		},
		{
			name:     "lookup uid error",
			filePath: "/etc/testfile",
			fileMode: 0644,
			fileUid:  0,
			fileGid:  0,
			config: blueprint.FileCustomization{
				Path:  "/etc/testfile",
				Mode:  "0644",
				User:  "nonexistent",
				Group: int64(0),
			},
			lookupUidError: errors.New("user not found"),
			wantError:      true,
			wantFail:       true,
		},
		{
			name:     "lookup gid error",
			filePath: "/etc/testfile",
			fileMode: 0644,
			fileUid:  0,
			fileGid:  0,
			config: blueprint.FileCustomization{
				Path:  "/etc/testfile",
				Mode:  "0644",
				User:  int64(0),
				Group: "nonexistent",
			},
			lookupGidError: errors.New("group not found"),
			wantError:      true,
			wantFail:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test.MockGlobal(t, &check.Exists, func(name string) bool {
				return true
			})

			test.MockGlobal(t, &check.Stat, func(name string) (os.FileInfo, error) {
				if name == tt.filePath {
					if tt.statError != nil {
						return nil, tt.statError
					}
					return &mockFileInfo{
						name: "testfile",
						mode: tt.fileMode,
						uid:  tt.fileUid,
						gid:  tt.fileGid,
					}, nil
				}
				return nil, errors.New("file not found")
			})

			if tt.config.Data != "" || tt.readFileData != nil || tt.readFileError != nil {
				test.MockGlobal(t, &check.ReadFile, func(filename string) ([]byte, error) {
					if filename == tt.filePath {
						if tt.readFileError != nil {
							return nil, tt.readFileError
						}
						return tt.readFileData, nil
					}
					return nil, errors.New("file not found")
				})
			}

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

			chk, found := check.FindCheckByName("Files Check")
			require.True(t, found, "Files Check not found")
			config := buildConfig(&blueprint.Customizations{
				Files: []blueprint.FileCustomization{tt.config},
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
