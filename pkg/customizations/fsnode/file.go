package fsnode

import (
	"os"
)

type File struct {
	Path  string
	Mode  *os.FileMode
	User  any
	Group any

	Data []byte
}

// NewFile creates a new file with the given path, data, mode, user and group.
// user and group can be either a string (user name/group name), an int64 (UID/GID) or nil.
func NewFile(path string, mode *os.FileMode, user interface{}, group interface{}, data []byte) (*File, error) {
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
