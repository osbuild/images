package fsnode

import (
	"os"
)

type Directory struct {
	Path  string
	Mode  *os.FileMode
	User  any
	Group any

	EnsureParentDirs bool
}

// NewDirectory creates a new directory with the given path, mode, user and group.
// user and group can be either a string (user name/group name), an int64 (UID/GID) or nil.
func NewDirectory(path string, mode *os.FileMode, user interface{}, group interface{}, ensureParentDirs bool) (*Directory, error) {
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
