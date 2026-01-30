package check_test

import (
	"testing"

	"github.com/osbuild/blueprint/pkg/blueprint"
	check "github.com/osbuild/images/cmd/check-host-config/check"
	"github.com/osbuild/images/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServicesEnabledCheck(t *testing.T) {
	test.MockGlobal(t, &check.Exec, func(name string, arg ...string) ([]byte, []byte, int, error) {
		if joinArgs(name, arg...) == "systemctl is-enabled test.service" {
			return []byte("enabled\n"), nil, 0, nil
		}
		return nil, nil, 0, nil
	})

	chk, found := check.FindCheckByName("srv-enabled")
	require.True(t, found, "Services Enabled Check not found")
	config := buildConfig(&blueprint.Customizations{
		Services: &blueprint.ServicesCustomization{
			Enabled: []string{"test.service"},
		},
	})

	require.NoError(t, chk.Func(chk.Meta, config))
}

func TestServicesEnabledCheckSkip(t *testing.T) {
	chk, found := check.FindCheckByName("srv-enabled")
	require.True(t, found, "Services Enabled Check not found")
	config := buildConfig(&blueprint.Customizations{
		Services: &blueprint.ServicesCustomization{
			Enabled: []string{},
		},
	})

	err := chk.Func(chk.Meta, config)
	require.Error(t, err)
	assert.True(t, check.IsSkip(err))
}
