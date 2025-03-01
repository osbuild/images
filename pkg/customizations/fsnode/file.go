package fsnode

import (
	"fmt"
	"net/url"
	"os"
	"syscall"

	"github.com/osbuild/images/internal/common"
)

type File struct {
	baseFsNode
	data []byte

	ref string
}

func (f *File) IsDir() bool {
	return false
}

func (f *File) Data() []byte {
	if f == nil {
		return nil
	}
	return f.data
}

func (f *File) Ref() string {
	return f.ref
}

// NewFile creates a new file with the given path, data, mode, user and group.
// user and group can be either a string (user name/group name), an int64 (UID/GID) or nil.
func NewFile(path string, mode *os.FileMode, user interface{}, group interface{}, data []byte) (*File, error) {
	return newFile(path, mode, user, group, data, "")
}

// NewFleForRef creates a new file from the given "ref" (usually a local
// file but potentially a URL later)
func NewFileForRef(targetPath string, mode *os.FileMode, user interface{}, group interface{}, ref string) (*File, error) {
	uri, err := url.Parse(ref)
	if err != nil {
		return nil, err
	}
	switch uri.Scheme {
	case "", "file":
		return newFileForRefLocalFile(targetPath, mode, user, group, uri)
	default:
		return nil, fmt.Errorf("unsupported scheme for %v (try file://)", ref)
	}
}

func newFileForRefLocalFile(targetPath string, mode *os.FileMode, user interface{}, group interface{}, uri *url.URL) (*File, error) {
	st, err := os.Stat(uri.Path)
	if err != nil {
		return nil, fmt.Errorf("cannot include blueprint file reference: %w", err)
	}
	if !st.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", uri.Path)
	}
	if mode == nil {
		mode = common.ToPtr(st.Mode())
	}
	if user == nil {
		user = int64(st.Sys().(*syscall.Stat_t).Uid)
	}
	if group == nil {
		group = int64(st.Sys().(*syscall.Stat_t).Gid)
	}

	return newFile(targetPath, mode, user, group, nil, uri.Path)
}

func newFile(path string, mode *os.FileMode, user interface{}, group interface{}, data []byte, ref string) (*File, error) {
	baseNode, err := newBaseFsNode(path, mode, user, group)

	if err != nil {
		return nil, err
	}

	return &File{
		baseFsNode: *baseNode,
		data:       data,
		ref:        ref,
	}, nil
}
