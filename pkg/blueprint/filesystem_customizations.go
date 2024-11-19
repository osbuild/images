package blueprint

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/osbuild/images/pkg/datasizes"
	"github.com/osbuild/images/pkg/pathpolicy"
)

type FilesystemCustomization struct {
	Mountpoint string         `json:"mountpoint,omitempty" toml:"mountpoint,omitempty"`
	MinSize    datasizes.Size `json:"minsize,omitempty" toml:"minsize,omitempty"`
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

func (fsc *FilesystemCustomization) UnmarshalJSON(data []byte) error {
	// this is only needed to generate nicer errors with a hint
	// if the custom unmarshal for minsize failed (as encoding/json
	// provides sadly no context), c.f.
	// https://github.com/golang/go/issues/58655
	type filesystemCustomization FilesystemCustomization
	var fc filesystemCustomization
	if err := json.Unmarshal(data, &fc); err != nil {
		if fc.Mountpoint != "" {
			return fmt.Errorf("JSON unmarshal: error decoding minsize value for mountpoint %q: %w", fc.Mountpoint, err)
		}
		return err
	}
	*fsc = FilesystemCustomization(fc)
	return nil
}
