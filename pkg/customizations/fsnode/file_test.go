package fsnode

import (
	"os"
	"testing"

	"github.com/osbuild/images/internal/common"
	"github.com/stretchr/testify/assert"
)

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
			expected: &File{Path: "/etc/file", Mode: nil, User: nil, Group: nil, Data: nil},
		},
		{
			name:     "file-with-data",
			path:     "/etc/file",
			data:     []byte("data"),
			mode:     nil,
			user:     nil,
			group:    nil,
			expected: &File{Path: "/etc/file", Mode: nil, User: nil, Group: nil, Data: []byte("data")},
		},
		{
			name:     "file-with-mode",
			path:     "/etc/file",
			data:     nil,
			mode:     common.ToPtr(os.FileMode(0644)),
			user:     nil,
			group:    nil,
			expected: &File{Path: "/etc/file", Mode: common.ToPtr(os.FileMode(0644)), User: nil, Group: nil, Data: nil},
		},
		{
			name:     "file-with-user-and-group-string",
			path:     "/etc/file",
			data:     nil,
			mode:     nil,
			user:     "user",
			group:    "group",
			expected: &File{Path: "/etc/file", Mode: nil, User: "user", Group: "group", Data: nil},
		},
		{
			name:     "file-with-user-and-group-int64",
			path:     "/etc/file",
			data:     nil,
			mode:     nil,
			user:     int64(1000),
			group:    int64(1000),
			expected: &File{Path: "/etc/file", Mode: nil, User: int64(1000), Group: int64(1000), Data: nil},
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
