package fsnode

import (
	"fmt"
	"os"
	"testing"

	"gopkg.in/yaml.v3"

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

func TestFsNodeUnmarshalFile(t *testing.T) {
	inputYAML := `
path: /some/path
mode: 0644
user: 1000
group: group
text: some-data
`
	var fsn File
	err := yaml.Unmarshal([]byte(inputYAML), &fsn)
	assert.NoError(t, err)
	expected, err := NewFile("/some/path", common.ToPtr(os.FileMode(0644)), float64(1000), "group", []byte("some-data"))
	assert.NoError(t, err)
	assert.Equal(t, expected, &fsn)
}

func TestFsNodeUnmarshalBadFile(t *testing.T) {
	for _, tc := range []struct {
		inputYAML   string
		expectedErr string
	}{
		{`path: 123`, `json: cannot unmarshal number into Go struct field .file.path of type string`},
		{`mode: -rw-rw-r--`, ` json: cannot unmarshal string into Go struct field .file.mode of type fs.FileMode`},
		{`mode: -1`, `cannot unmarshal number -1 into Go struct field .file.mode of type fs.FileMode`},
		{`mode: 5_000_000_000`, `json: cannot unmarshal number 5000000000 into Go struct field .file.mode of type fs.FileMode`},
		{"path: /foo\nuser: 3.14", `user ID must be int`},
		{"path: /foo\ngroup: 2.71", `group ID must be int`},
		{"path: /foo\nuser: -1", `user ID must be non-negative`},
		{"path: /foo\ngroup: a!b", `group name "a!b" doesn't conform to validating regex`},
		{"path: /foo\ntext: 1.61", `cannot unmarshal number into Go struct field .text of type string`},
		{"path: /foo\nextra: field", `unknown field "extra"`},
	} {
		var fsn File
		err := yaml.Unmarshal([]byte(tc.inputYAML), &fsn)
		assert.ErrorContains(t, err, tc.expectedErr)
	}
}
