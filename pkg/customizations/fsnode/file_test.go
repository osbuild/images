package fsnode

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/osbuild/images/internal/common"
	"github.com/stretchr/testify/assert"
)

func TestFileIsDir(t *testing.T) {
	file, err := NewFile("/etc/file", nil, nil, nil, nil)
	assert.NoError(t, err)
	assert.False(t, file.IsDir())
}

func TestNewFile(t *testing.T) {
	testCases := []struct {
		name     string
		path     string
		data     []byte
		mode     *os.FileMode
		user     interface{}
		group    interface{}
		expected *File
	}{
		{
			name:     "empty-file",
			path:     "/etc/file",
			data:     nil,
			mode:     nil,
			user:     nil,
			group:    nil,
			expected: &File{baseFsNode: baseFsNode{path: "/etc/file", mode: nil, user: nil, group: nil}, data: nil},
		},
		{
			name:     "file-with-data",
			path:     "/etc/file",
			data:     []byte("data"),
			mode:     nil,
			user:     nil,
			group:    nil,
			expected: &File{baseFsNode: baseFsNode{path: "/etc/file", mode: nil, user: nil, group: nil}, data: []byte("data")},
		},
		{
			name:     "file-with-mode",
			path:     "/etc/file",
			data:     nil,
			mode:     common.ToPtr(os.FileMode(0644)),
			user:     nil,
			group:    nil,
			expected: &File{baseFsNode: baseFsNode{path: "/etc/file", mode: common.ToPtr(os.FileMode(0644)), user: nil, group: nil}, data: nil},
		},
		{
			name:     "file-with-user-and-group-string",
			path:     "/etc/file",
			data:     nil,
			mode:     nil,
			user:     "user",
			group:    "group",
			expected: &File{baseFsNode: baseFsNode{path: "/etc/file", mode: nil, user: "user", group: "group"}, data: nil},
		},
		{
			name:     "file-with-user-and-group-int64",
			path:     "/etc/file",
			data:     nil,
			mode:     nil,
			user:     int64(1000),
			group:    int64(1000),
			expected: &File{baseFsNode: baseFsNode{path: "/etc/file", mode: nil, user: int64(1000), group: int64(1000)}, data: nil},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			file, err := NewFile(tc.path, tc.mode, tc.user, tc.group, tc.data)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, file)
		})
	}
}

func TestNewFileForRef(t *testing.T) {
	testFile1 := filepath.Join(t.TempDir(), "test1.txt")
	err := os.WriteFile(testFile1, nil, 0511)
	assert.NoError(t, err)

	file, err := NewFileForRef("/target/path", nil, nil, nil, testFile1)
	assert.NoError(t, err)
	assert.Equal(t, testFile1, file.Ref())
	assert.Equal(t, "/target/path", file.Path())
	assert.Equal(t, os.FileMode(0511), file.Mode().Perm())
	assert.Equal(t, int64(os.Getuid()), file.User())
	assert.Equal(t, int64(os.Getgid()), file.Group())
}

func TestNewFileForRefBadRefs(t *testing.T) {
	tmpdir := t.TempDir()

	for _, tc := range []struct {
		ref         string
		expectedErr string
	}{
		{"/not/exists", `cannot include blueprint file reference: stat /not/exists: no such file or directory`},
		{"file://%g", `parse "file://%g": invalid URL escape "%g"`},
		{"gopher://foo.txt", "unsupported scheme for gopher://foo.txt (try file://)"},
		{tmpdir, fmt.Sprintf("%s is not a regular file", tmpdir)},
	} {

		_, err := NewFileForRef("/target/path", nil, nil, nil, tc.ref)
		assert.EqualError(t, err, tc.expectedErr)
	}
}
