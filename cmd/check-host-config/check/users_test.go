package check_test

import (
	"testing"

	"github.com/osbuild/blueprint/pkg/blueprint"
	check "github.com/osbuild/images/cmd/check-host-config/check"
	"github.com/osbuild/images/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsersCheck(t *testing.T) {
	test.MockGlobal(t, &check.Exec, func(name string, arg ...string) ([]byte, []byte, int, error) {
		if joinArgs(name, arg...) == "id testuser" {
			return []byte("uid=1000(testuser) gid=1000(testuser) groups=1000(testuser)\n"), nil, 0, nil
		}
		return nil, nil, 0, nil
	})

	chk, found := check.FindCheckByName("users")
	require.True(t, found, "Users Check not found")
	config := buildConfig(&blueprint.Customizations{
		User: []blueprint.UserCustomization{
			{Name: "testuser"},
		},
	})

	require.NoError(t, chk.Func(chk.Meta, config))
}

func TestUsersCheckSkip(t *testing.T) {
	chk, found := check.FindCheckByName("users")
	require.True(t, found, "Users Check not found")
	config := buildConfig(&blueprint.Customizations{
		User: []blueprint.UserCustomization{},
	})

	err := chk.Func(chk.Meta, config)
	require.Error(t, err)
	assert.True(t, check.IsSkip(err))
}
