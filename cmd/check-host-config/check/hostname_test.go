package check_test

import (
	"testing"

	"github.com/osbuild/blueprint/pkg/blueprint"
	check "github.com/osbuild/images/cmd/check-host-config/check"
	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostnameCheck(t *testing.T) {
	test.MockGlobal(t, &check.Exec, func(name string, arg ...string) ([]byte, []byte, int, error) {
		return []byte("test-hostname\n"), nil, 0, nil
	})

	chk, found := check.FindCheckByName("hostname")
	require.True(t, found, "Hostname Check not found")
	config := buildConfig(&blueprint.Customizations{
		Hostname: common.ToPtr("test-hostname"),
	})

	require.NoError(t, chk.Func(chk.Meta, config))
}

func TestHostnameCheckWarning(t *testing.T) {
	test.MockGlobal(t, &check.Exec, func(name string, arg ...string) ([]byte, []byte, int, error) {
		return []byte("changed-by-cloud-init\n"), nil, 0, nil
	})

	chk, found := check.FindCheckByName("hostname")
	require.True(t, found, "Hostname Check not found")
	config := buildConfig(&blueprint.Customizations{
		Hostname: common.ToPtr("test-hostname"),
	})

	err := chk.Func(chk.Meta, config)
	require.Error(t, err)
	assert.True(t, check.IsWarning(err))
}
