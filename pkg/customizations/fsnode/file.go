package fsnode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/osbuild/images/internal/common"
)

type File struct {
	baseFsNode
	data []byte
}

func (f *File) Data() []byte {
	if f == nil {
		return nil
	}
	return f.data
}

func (f *File) UnmarshalJSON(data []byte) error {
	if err := f.baseFsNode.UnmarshalJSON(data); err != nil {
		return err
	}

	var m map[string]interface{}
	dec := json.NewDecoder(bytes.NewBuffer(data))
	if err := dec.Decode(&m); err != nil {
		return err
	}
	if data, ok := m["data"]; ok {
		dataStr, ok := data.(string)
		if !ok {
			return fmt.Errorf("unexpected type %T for data (want string)", data)
		}
		f.data = []byte(dataStr)
	}
	return nil
}

func (f *File) UnmarshalYAML(unmarshal func(any) error) error {
	return common.UnmarshalYAMLviaJSON(f, unmarshal)
}

// NewFile creates a new file with the given path, data, mode, user and group.
// user and group can be either a string (user name/group name), an int64 (UID/GID) or nil.
func NewFile(path string, mode *os.FileMode, user interface{}, group interface{}, data []byte) (*File, error) {
	baseNode, err := newBaseFsNode(path, mode, user, group)

	if err != nil {
		return nil, err
	}

	return &File{
		baseFsNode: *baseNode,
		data:       data,
	}, nil
}
