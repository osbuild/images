package fsnode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"

	"github.com/osbuild/images/internal/common"
)

const usernameRegex = `^[A-Za-z0-9_.][A-Za-z0-9_.-]{0,31}$`
const groupnameRegex = `^[A-Za-z0-9_][A-Za-z0-9_-]{0,31}$`

type baseFsNode struct {
	path  string
	mode  *os.FileMode
	user  interface{}
	group interface{}
}

func (f *baseFsNode) Path() string {
	if f == nil {
		return ""
	}
	return f.path
}

func (f *baseFsNode) Mode() *os.FileMode {
	if f == nil {
		return nil
	}
	return f.mode
}

// User can return either a string (user name) or an int64 (UID)
func (f *baseFsNode) User() interface{} {
	if f == nil {
		return nil
	}
	return f.user
}

// Group can return either a string (group name) or an int64 (GID)
func (f *baseFsNode) Group() interface{} {
	if f == nil {
		return nil
	}
	return f.group
}

func newBaseFsNode(path string, mode *os.FileMode, user interface{}, group interface{}) (*baseFsNode, error) {
	node := &baseFsNode{
		path:  path,
		mode:  mode,
		user:  user,
		group: group,
	}

	err := node.validate()
	if err != nil {
		return nil, err
	}
	return node, nil
}

func (f *baseFsNode) validate() error {
	// Check that the path is valid
	if f.path == "" {
		return fmt.Errorf("path must not be empty")
	}
	if f.path[0] != '/' {
		return fmt.Errorf("path must be absolute")
	}
	if f.path[len(f.path)-1] == '/' {
		return fmt.Errorf("path must not end with a slash")
	}
	if f.path != filepath.Clean(f.path) {
		return fmt.Errorf("path must be canonical")
	}

	// Check that the mode is valid
	if f.mode != nil && *f.mode&os.ModeType != 0 {
		return fmt.Errorf("mode must not contain file type bits")
	}

	// Check that the user and group are valid
	switch user := f.user.(type) {
	case string:
		nameRegex := regexp.MustCompile(usernameRegex)
		if !nameRegex.MatchString(user) {
			return fmt.Errorf("user name %q doesn't conform to validating regex (%s)", user, nameRegex.String())
		}
	case int64:
		if user < 0 {
			return fmt.Errorf("user ID must be non-negative")
		}
	case nil:
		// user is not set
	default:
		return fmt.Errorf("user must be either a string or an int64, got %T", user)
	}

	switch group := f.group.(type) {
	case string:
		nameRegex := regexp.MustCompile(groupnameRegex)
		if !nameRegex.MatchString(group) {
			return fmt.Errorf("group name %q doesn't conform to validating regex (%s)", group, nameRegex.String())
		}
	case int64:
		if group < 0 {
			return fmt.Errorf("group ID must be non-negative")
		}
	case nil:
		// group is not set
	default:
		return fmt.Errorf("group must be either a string or an int64, got %T", group)
	}

	return nil
}
func (f *baseFsNode) UnmarshalJSON(data []byte) (err error) {
	var m map[string]interface{}
	dec := json.NewDecoder(bytes.NewBuffer(data))
	dec.UseNumber()
	if err := dec.Decode(&m); err != nil {
		return err
	}
	if path, ok := m["path"]; ok {
		f.path, ok = path.(string)
		if !ok {
			return fmt.Errorf("unexpected type %T for path (want string)", path)
		}
	}
	if mode, ok := m["mode"]; ok {
		modeNum, ok := mode.(json.Number)
		if !ok {
			return fmt.Errorf("unexpected type %T for mode (want number)", mode)
		}
		fsMode, err := modeNum.Int64()
		if err != nil {
			return fmt.Errorf("mode is not an int: %v", err)
		}
		if fsMode < 0 || fsMode > math.MaxUint32 {
			return fmt.Errorf("mode %v is outside the allowed range of [0,%v]", fsMode, math.MaxUint32)
		}
		//nolint:gosec
		f.mode = common.ToPtr(os.FileMode(uint32(fsMode)))
	}
	switch user := m["user"].(type) {
	case json.Number:
		f.user, err = user.Int64()
		if err != nil {
			return fmt.Errorf("user is a number but not an int: %v", err)
		}
	default:
		f.user = user
	}
	switch group := m["group"].(type) {
	case json.Number:
		f.group, err = group.Int64()
		if err != nil {
			return fmt.Errorf("group is a number but not an int: %v", err)
		}
	default:
		f.group = group
	}

	return f.validate()

}

func (f *baseFsNode) UnmarshalYAML(unmarshal func(any) error) error {
	return common.UnmarshalYAMLviaJSON(f, unmarshal)
}

// fieldNams returns all field names of the given struct pointer, it
// will panic when pointing to something else
func fieldNames(s interface{}) map[string]bool {
	val := reflect.ValueOf(s).Elem()

	fieldNames := map[string]bool{}
	var collect func(v reflect.Value, t reflect.Type)
	collect = func(v reflect.Value, t reflect.Type) {
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			fieldVal := v.Field(i)
			if field.Anonymous {
				collect(fieldVal, fieldVal.Type())
			} else {
				if field.Tag.Get("rename") != "" {
					fieldNames[field.Tag.Get("rename")] = true
				} else {
					fieldNames[field.Name] = true
				}
			}
		}
	}
	collect(val, val.Type())

	return fieldNames
}
