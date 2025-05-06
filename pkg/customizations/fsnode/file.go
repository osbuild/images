package fsnode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/osbuild/images/internal/common"
)

type File struct {
	Path  string       `json:"path,omitempty"`
	Mode  *os.FileMode `json:"mode,omitempty"`
	User  any          `json:"user,omitempty"`
	Group any          `json:"group,omitempty"`

	Data []byte `json:"data,omitempty"`
}

// NewFile creates a new file with the given path, data, mode, user and group.
// user and group can be either a string (user name/group name), an int64 (UID/GID) or nil.
func NewFile(path string, mode *os.FileMode, user any, group any, data []byte) (*File, error) {
	if err := validate(path, mode, user, group); err != nil {
		return nil, err
	}

	return &File{
		Path:  path,
		Mode:  mode,
		User:  user,
		Group: group,
		Data:  data,
	}, nil
}

func (f *File) UnmarshalJSON(data []byte) error {
	// create an alias so that we can custom unmarshal without
	// infinite loop
	type file File
	var v struct {
		file
		// when unmarshaling, support a "text" field for
		// convenience that is converted into "Data []byte"
		// so that we can write nice ImageConfigs without
		// having to "base64" encode the content
		Text string `json:"text,omitempty"`
	}
	dec := json.NewDecoder(bytes.NewBuffer(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&v); err != nil {
		return err
	}

	if err := validate(v.Path, v.Mode, v.User, v.Group); err != nil {
		return err
	}
	if v.Data != nil && v.Text != "" {
		return fmt.Errorf("fsnode file only allows data or text but not both")
	}
	*f = File(v.file)
	f.Data = []byte(v.Text)

	return nil
}

func (f *File) UnmarshalYAML(unmarshal func(any) error) error {
	return common.UnmarshalYAMLviaJSON(f, unmarshal)
}
