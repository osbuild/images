package fsnode

import (
	"bytes"
	"encoding/json"
	"os"

	"github.com/osbuild/images/internal/common"
)

type Directory struct {
	Path  string       `json:"path,omitempty"`
	Mode  *os.FileMode `json:"mode,omitempty"`
	User  any          `json:"user,omitempty"`
	Group any          `json:"group,omitempty"`

	EnsureParentDirs bool `json:"ensure_parent_dirs,omitempty"`
}

// NewDirectory creates a new directory with the given path, mode, user and group.
// user and group can be either a string (user name/group name), an int64 (UID/GID) or nil.
func NewDirectory(path string, mode *os.FileMode, user any, group any, ensureParentDirs bool) (*Directory, error) {
	if err := validate(path, mode, user, group); err != nil {
		return nil, err
	}

	return &Directory{
		Path:             path,
		Mode:             mode,
		User:             user,
		Group:            group,
		EnsureParentDirs: ensureParentDirs,
	}, nil
}

func (d *Directory) UnmarshalJSON(data []byte) error {
	type dir Directory

	var v dir
	dec := json.NewDecoder(bytes.NewBuffer(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&v); err != nil {
		return err
	}
	*d = Directory(v)

	return validate(d.Path, d.Mode, d.User, d.Group)
}

func (d *Directory) UnmarshalYAML(unmarshal func(any) error) error {
	return common.UnmarshalYAMLviaJSON(d, unmarshal)
}
