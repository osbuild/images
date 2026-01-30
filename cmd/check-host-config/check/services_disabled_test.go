package check_test

import (
	"testing"

	"github.com/osbuild/blueprint/pkg/blueprint"
	check "github.com/osbuild/images/cmd/check-host-config/check"
	"github.com/osbuild/images/internal/test"
	"github.com/stretchr/testify/require"
)

func TestServicesDisabledCheck(t *testing.T) {
	test.MockGlobal(t, &check.Exec, func(name string, arg ...string) ([]byte, []byte, int, error) {
		if joinArgs(name, arg...) == "systemctl is-enabled test.service" {
			return []byte("disabled\n"), nil, 0, nil
		}
		return nil, nil, 0, nil
	})

	chk, found := check.FindCheckByName("srv-disabled")
	require.True(t, found, "Services Disabled Check not found")
	config := buildConfig(&blueprint.Customizations{
		Services: &blueprint.ServicesCustomization{
			Disabled: []string{"test.service"},
		},
	})

	require.NoError(t, chk.Func(chk.Meta, config))
}
