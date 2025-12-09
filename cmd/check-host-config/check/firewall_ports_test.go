package check_test

import (
	"testing"

	"github.com/osbuild/blueprint/pkg/blueprint"
	check "github.com/osbuild/images/cmd/check-host-config/check"
	"github.com/osbuild/images/internal/test"
	"github.com/stretchr/testify/require"
)

func TestFirewallPortsCheck(t *testing.T) {
	test.MockGlobal(t, &check.Exec, func(name string, arg ...string) ([]byte, []byte, int, error) {
		if joinArgs(name, arg...) == "sudo firewall-cmd --query-port=80/tcp" {
			return []byte("yes\n"), nil, 0, nil
		}
		return nil, nil, 0, nil
	})

	chk, found := check.FindCheckByName("Firewall Ports Check")
	require.True(t, found, "Firewall Ports Check not found")
	config := buildConfig(&blueprint.Customizations{
		Firewall: &blueprint.FirewallCustomization{
			Ports: []string{"80:tcp"},
		},
	})

	require.NoError(t, chk.Func(chk.Meta, config))
}
