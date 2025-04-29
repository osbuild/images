package fsnode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/osbuild/images/internal/common"
)

type Directory struct {
	baseFsNode
	ensureParentDirs bool `rename:"ensure_parent_dirs"`
}

func (d *Directory) EnsureParentDirs() bool {
	if d == nil {
		return false
	}
	return d.ensureParentDirs
}

func (d *Directory) UnmarshalJSON(data []byte) error {
	if err := d.baseFsNode.UnmarshalJSON(data); err != nil {
		return err
	}

	// XXX: ideally we would also check here that we do not
	// get extra/mistyped fields
	var m map[string]interface{}
	dec := json.NewDecoder(bytes.NewBuffer(data))
	if err := dec.Decode(&m); err != nil {
		return err
	}
	if ensureParents, ok := m["ensure_parent_dirs"]; ok {
		d.ensureParentDirs, ok = ensureParents.(bool)
		if !ok {
			return fmt.Errorf("unexpected type %T for ensure_parent_dirs (want bool)", ensureParents)
		}
	}

	// validate only known names are used
	fieldNames := fieldNames(d)
	for k := range m {
		if !fieldNames[k] {
			return fmt.Errorf("unknown key %q in dir", k)
		}
	}

	return nil
}

func (d *Directory) UnmarshalYAML(unmarshal func(any) error) error {
	return common.UnmarshalYAMLviaJSON(d, unmarshal)
}

// NewDirectory creates a new directory with the given path, mode, user and group.
// user and group can be either a string (user name/group name), an int64 (UID/GID) or nil.
func NewDirectory(path string, mode *os.FileMode, user interface{}, group interface{}, ensureParentDirs bool) (*Directory, error) {
	baseNode, err := newBaseFsNode(path, mode, user, group)

	if err != nil {
		return nil, err
	}

	return &Directory{
		baseFsNode:       *baseNode,
		ensureParentDirs: ensureParentDirs,
	}, nil
}
