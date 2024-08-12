package blueprint

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/pkg/pathpolicy"
)

func TestCheckMountpointsPolicy(t *testing.T) {
	policy := pathpolicy.NewPathPolicies(map[string]pathpolicy.PathPolicy{
		"/": {Exact: true},
	})

	mps := []FilesystemCustomization{
		{Mountpoint: "/foo"},
		{Mountpoint: "/boot/"},
	}

	expectedErr := `The following errors occurred while setting up custom mountpoints:
path "/foo" is not allowed
path "/boot/" must be canonical`
	err := CheckMountpointsPolicy(mps, policy)
	assert.EqualError(t, err, expectedErr)
}
