package fsnode

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"github.com/osbuild/images/internal/common"
)

func TestBaseFsNodeValidate(t *testing.T) {
	testCases := []struct {
		Node  baseFsNode
		Error bool
	}{
		// PATH
		// relative path is not allowed
		{
			Node: baseFsNode{
				path: "relative/path/file",
			},
			Error: true,
		},
		// path ending with slash is not allowed
		{
			Node: baseFsNode{
				path: "/dir/with/trailing/slash/",
			},
			Error: true,
		},
		// empty path is not allowed
		{
			Node: baseFsNode{
				path: "",
			},
			Error: true,
		},
		// path must be canonical
		{
			Node: baseFsNode{
				path: "/dir/../file",
			},
			Error: true,
		},
		{
			Node: baseFsNode{
				path: "/dir/./file",
			},
			Error: true,
		},
		// valid paths
		{
			Node: baseFsNode{
				path: "/etc/file",
			},
		},
		{
			Node: baseFsNode{
				path: "/etc/dir",
			},
		},
		// MODE
		// invalid mode
		{
			Node: baseFsNode{
				path: "/etc/file",
				mode: common.ToPtr(os.FileMode(os.ModeDir)),
			},
			Error: true,
		},
		// valid mode
		{
			Node: baseFsNode{
				path: "/etc/file",
				mode: common.ToPtr(os.FileMode(0o644)),
			},
		},
		// USER
		// invalid user
		{
			Node: baseFsNode{
				path: "/etc/file",
				user: "",
			},
			Error: true,
		},
		{
			Node: baseFsNode{
				path: "/etc/file",
				user: "invalid@@@user",
			},
			Error: true,
		},
		{
			Node: baseFsNode{
				path: "/etc/file",
				user: int64(-1),
			},
			Error: true,
		},
		// valid user
		{
			Node: baseFsNode{
				path: "/etc/file",
				user: "osbuild",
			},
		},
		{
			Node: baseFsNode{
				path: "/etc/file",
				user: int64(0),
			},
		},
		// GROUP
		// invalid group
		{
			Node: baseFsNode{
				path:  "/etc/file",
				group: "",
			},
			Error: true,
		},
		{
			Node: baseFsNode{
				path:  "/etc/file",
				group: "invalid@@@group",
			},
			Error: true,
		},
		{
			Node: baseFsNode{
				path:  "/etc/file",
				group: int64(-1),
			},
			Error: true,
		},
		// valid group
		{
			Node: baseFsNode{
				path:  "/etc/file",
				group: "osbuild",
			},
		},
		{
			Node: baseFsNode{
				path:  "/etc/file",
				group: int64(0),
			},
		},
	}

	for idx, testCase := range testCases {
		t.Run(fmt.Sprintf("case #%d", idx), func(t *testing.T) {
			err := testCase.Node.validate()
			if testCase.Error {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFsNodeUnmarshalDir(t *testing.T) {
	inputYAML := `
path: /some/path
mode: 0644
user: 1000
group: group
ensure_parent_dirs: true
`
	var fsn Directory
	err := yaml.Unmarshal([]byte(inputYAML), &fsn)
	assert.NoError(t, err)
	expected, err := NewDirectory("/some/path", common.ToPtr(os.FileMode(0644)), int64(1000), "group", true)
	assert.NoError(t, err)
	assert.Equal(t, expected, &fsn)
}

func TestFsNodeUnmarshalFile(t *testing.T) {
	inputYAML := `
path: /some/path
mode: 0644
user: 1000
group: group
data: some-data
`
	var fsn File
	err := yaml.Unmarshal([]byte(inputYAML), &fsn)
	assert.NoError(t, err)
	expected, err := NewFile("/some/path", common.ToPtr(os.FileMode(0644)), int64(1000), "group", []byte("some-data"))
	assert.NoError(t, err)
	assert.Equal(t, expected, &fsn)
}

func TestFsNodeUnmarshalBadFile(t *testing.T) {
	for _, tc := range []struct {
		inputYAML   string
		expectedErr string
	}{
		{`path: 123`, `unexpected type json.Number for path (want string)`},
		{`mode: -rw-rw-r--`, `unexpected type string for mode (want number)`},
		{`mode: -1`, `mode -1 is outside the allowed range of [0,4294967295]`},
		{`mode: 5_000_000_000`, `mode 5000000000 is outside the allowed range of [0,4294967295]`},
		{`user: 3.14`, `user is a number but not an int: strconv.ParseInt: parsing "3.14": invalid syntax`},
		{`group: 2.71`, `group is a number but not an int: strconv.ParseInt: parsing "2.71": invalid syntax`},
		{"path: /foo\nuser: -1", `user ID must be non-negative`},
		{"path: /foo\ngroup: a!b", `group name "a!b" doesn't conform to validating regex`},
		{"path: /foo\ndata: 1.61", `unexpected type float64 for data (want string)`},
	} {
		var fsn File
		err := yaml.Unmarshal([]byte(tc.inputYAML), &fsn)
		assert.ErrorContains(t, err, tc.expectedErr)
	}
}

func TestFsNodeUnmarshalBadDir(t *testing.T) {
	for _, tc := range []struct {
		inputYAML   string
		expectedErr string
	}{
		{"path: /foo\nensure_parent_dirs: maybe", `unexpected type string for ensure_parent_dirs (want bool)`},
	} {
		var fsn Directory
		err := yaml.Unmarshal([]byte(tc.inputYAML), &fsn)
		assert.ErrorContains(t, err, tc.expectedErr)
	}
}
