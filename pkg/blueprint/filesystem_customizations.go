package blueprint

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/pathpolicy"
)

type FilesystemCustomization struct {
	Mountpoint Mountpoint `json:"mountpoint,omitempty" toml:"mountpoint,omitempty"`
	MinSize    MinSize    `json:"minsize,omitempty" toml:"minsize,omitempty"`
}

type Mountpoint string

func (mt *Mountpoint) UnmarshalTOML(d interface{}) error {
	switch d.(type) {
	case string:
		*mt = Mountpoint(d.(string))
	default:
		return fmt.Errorf("TOML unmarshal: mountpoint must be string, got %v of type %T", d, d)
	}
	return nil
}

type MinSize uint64

func (ms *MinSize) UnmarshalTOML(d interface{}) error {
	switch d.(type) {
	case int64:
		*ms = MinSize(d.(int64))
	case string:
		minSize, err := common.DataSizeToUint64(d.(string))
		if err != nil {
			return fmt.Errorf("TOML unmarshal: minsize is not valid filesystem size (%w)", err)
		}
		*ms = MinSize(minSize)
	default:
		return fmt.Errorf("TOML unmarshal: minsize must be integer or string, got %v of type %T", d, d)
	}

	return nil
}

func (fsc *FilesystemCustomization) UnmarshalJSON(data []byte) error {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	d, _ := v.(map[string]interface{})

	switch d["mountpoint"].(type) {
	case string:
		fsc.Mountpoint = Mountpoint(d["mountpoint"].(string))
	default:
		return fmt.Errorf("JSON unmarshal: mountpoint must be string, got %v of type %T", d["mountpoint"], d["mountpoint"])
	}

	// The JSON specification only mentions float64 and Go defaults to it: https://go.dev/blog/json
	switch d["minsize"].(type) {
	case float64:
		fsc.MinSize = MinSize(d["minsize"].(float64))
	case string:
		minSize, err := common.DataSizeToUint64(d["minsize"].(string))
		if err != nil {
			return fmt.Errorf("JSON unmarshal: minsize is not valid filesystem size (%w)", err)
		}
		fsc.MinSize = MinSize(minSize)
	default:
		return fmt.Errorf("JSON unmarshal: minsize must be float64 number or string, got %v of type %T", d["minsize"], d["minsize"])
	}

	return nil
}

// CheckMountpointsPolicy checks if the mountpoints are allowed by the policy
func CheckMountpointsPolicy(mountpoints []FilesystemCustomization, mountpointAllowList *pathpolicy.PathPolicies) error {
	var errs []error
	for _, m := range mountpoints {
		if err := mountpointAllowList.Check(string(m.Mountpoint)); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("The following errors occurred while setting up custom mountpoints:\n%w", errors.Join(errs...))
	}

	return nil
}
