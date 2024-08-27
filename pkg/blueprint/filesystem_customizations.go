package blueprint

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/osbuild/images/pkg/datasizes"
	"github.com/osbuild/images/pkg/pathpolicy"
)

type FilesystemCustomization struct {
	Mountpoint string `json:"mountpoint" toml:"mountpoint"`
	MinSize    uint64 `json:"minsize,omitempty" toml:"minsize,omitempty"`
	Label      string `json:"label,omitempty" toml:"label,omitempty"`
	Type       string `json:"type,omitempty" toml:"type,omitempty"`
}

func (fsc *FilesystemCustomization) UnmarshalTOML(data interface{}) error {
	d, _ := data.(map[string]interface{})

	switch d["mountpoint"].(type) {
	case string:
		fsc.Mountpoint = d["mountpoint"].(string)
	default:
		return fmt.Errorf("TOML unmarshal: mountpoint must be string, got %v of type %T", d["mountpoint"], d["mountpoint"])
	}

	switch d["type"].(type) {
	case nil:
		// empty allowed
	case string:
		fsc.Type = d["type"].(string)
	default:
		return fmt.Errorf("TOML unmarshal: type must be string, got %v of type %T", d["type"], d["type"])
	}

	switch d["label"].(type) {
	case nil:
		// empty allowed
	case string:
		fsc.Label = d["label"].(string)
	default:
		return fmt.Errorf("TOML unmarshal: label must be string, got %v of type %T", d["label"], d["label"])
	}

	switch d["minsize"].(type) {
	case int64:
		minSize := d["minsize"].(int64)
		if minSize < 0 {
			return fmt.Errorf("TOML unmarshal: minsize cannot be negative")
		}
		fsc.MinSize = uint64(minSize)
	case string:
		minSize, err := datasizes.Parse(d["minsize"].(string))
		if err != nil {
			return fmt.Errorf("TOML unmarshal: minsize is not valid filesystem size (%w)", err)
		}
		fsc.MinSize = minSize
	default:
		return fmt.Errorf("TOML unmarshal: minsize must be integer or string, got %v of type %T", d["minsize"], d["minsize"])
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
		fsc.Mountpoint = d["mountpoint"].(string)
	default:
		return fmt.Errorf("JSON unmarshal: mountpoint must be string, got %v of type %T", d["mountpoint"], d["mountpoint"])
	}

	switch d["type"].(type) {
	case nil:
		// empty allowed
	case string:
		fsc.Type = d["type"].(string)
	default:
		return fmt.Errorf("JSON unmarshal: type must be string, got %v of type %T", d["type"], d["type"])
	}

	switch d["label"].(type) {
	case nil:
		// empty allowed
	case string:
		fsc.Label = d["label"].(string)
	default:
		return fmt.Errorf("JSON unmarshal: label must be string, got %v of type %T", d["label"], d["label"])
	}

	// The JSON specification only mentions float64 and Go defaults to it: https://go.dev/blog/json
	switch d["minsize"].(type) {
	case float64:
		fsc.MinSize = uint64(d["minsize"].(float64))
	case string:
		minSize, err := datasizes.Parse(d["minsize"].(string))
		if err != nil {
			return fmt.Errorf("JSON unmarshal: minsize is not valid filesystem size (%w)", err)
		}
		fsc.MinSize = minSize
	default:
		return fmt.Errorf("JSON unmarshal: minsize must be float64 number or string, got %v of type %T", d["minsize"], d["minsize"])
	}

	return nil
}

// CheckMountpointsPolicy checks if the mountpoints are allowed by the policy
func CheckMountpointsPolicy(mountpoints []FilesystemCustomization, mountpointAllowList *pathpolicy.PathPolicies) error {
	var errs []error
	for _, m := range mountpoints {
		if err := mountpointAllowList.Check(m.Mountpoint); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("The following errors occurred while setting up custom mountpoints:\n%w", errors.Join(errs...))
	}

	return nil
}
