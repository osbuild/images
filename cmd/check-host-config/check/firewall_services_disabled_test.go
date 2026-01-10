package check_test

import (
	"testing"

	"github.com/osbuild/blueprint/pkg/blueprint"
	check "github.com/osbuild/images/cmd/check-host-config/check"
	"github.com/osbuild/images/internal/test"
	"github.com/stretchr/testify/require"
)

func TestFirewallServicesDisabledCheck(t *testing.T) {
	test.MockGlobal(t, &check.Exec, func(name string, arg ...string) ([]byte, []byte, error) {
		if joinArgs(name, arg...) == "sudo firewall-cmd --query-service=badservice" {
			return []byte("no\n"), nil, nil
		}
		return nil, nil, nil
	})

	chk, found := check.FindCheckByName("Firewall Services Disabled Check")
	require.True(t, found, "Firewall Services Disabled Check not found")
	config := buildConfig(&blueprint.Customizations{
		Firewall: &blueprint.FirewallCustomization{
			Services: &blueprint.FirewallServicesCustomization{
				Disabled: []string{"badservice"},
			},
		},
	})

	require.NoError(t, chk.Func(chk.Meta, config))
}
