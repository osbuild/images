package fsnode

import (
	"fmt"
	"os"
	"testing"

	"github.com/osbuild/images/internal/common"
	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	testCases := []struct {
		path  string
		mode  *os.FileMode
		user  any
		group any
		Error bool
	}{
		// PATH
		// relative path is not allowed
		{
			path:  "relative/path/file",
			Error: true,
		},
		// path ending with slash is not allowed
		{
			path:  "/dir/with/trailing/slash/",
			Error: true,
		},
		// empty path is not allowed
		{
			path:  "",
			Error: true,
		},
		// path must be canonical
		{
			path:  "/dir/../file",
			Error: true,
		},
		{
			path:  "/dir/./file",
			Error: true,
		},
		// valid paths
		{
			path: "/etc/file",
		},
		{
			path: "/etc/dir",
		},
		// MODE
		// invalid mode
		{
			path:  "/etc/file",
			mode:  common.ToPtr(os.FileMode(os.ModeDir)),
			Error: true,
		},
		// valid mode
		{
			path: "/etc/file",
			mode: common.ToPtr(os.FileMode(0o644)),
		},
		// USER
		// invalid user
		{
			path:  "/etc/file",
			user:  "",
			Error: true,
		},
		{
			path:  "/etc/file",
			user:  "invalid@@@user",
			Error: true,
		},
		{
			path:  "/etc/file",
			user:  int64(-1),
			Error: true,
		},
		// valid user
		{
			path: "/etc/file",
			user: "osbuild",
		},
		{
			path: "/etc/file",
			user: int64(0),
		},
		// GROUP
		// invalid group
		{
			path:  "/etc/file",
			group: "",
			Error: true,
		},
		{
			path:  "/etc/file",
			group: "invalid@@@group",
			Error: true,
		},
		{
			path:  "/etc/file",
			group: int64(-1),
			Error: true,
		},
		// valid group
		{
			path:  "/etc/file",
			group: "osbuild",
		},
		{
			path:  "/etc/file",
			group: int64(0),
		},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("case #%d", idx), func(t *testing.T) {
			err := validate(tc.path, tc.mode, tc.user, tc.group)
			if tc.Error {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
