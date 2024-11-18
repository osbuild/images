package blueprint

import (
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
