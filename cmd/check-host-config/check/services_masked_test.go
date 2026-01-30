package check_test

import (
	"testing"

	"github.com/osbuild/blueprint/pkg/blueprint"
	check "github.com/osbuild/images/cmd/check-host-config/check"
	"github.com/osbuild/images/internal/test"
	"github.com/stretchr/testify/require"
)

func TestServicesMaskedCheck(t *testing.T) {
	test.MockGlobal(t, &check.Exec, func(name string, arg ...string) ([]byte, []byte, int, error) {
		if joinArgs(name, arg...) == "systemctl list-unit-files --state=masked" {
			return []byte("UNIT FILE\t\t\t\t\tSTATE\n" +
				"test.service\t\t\t\t\tmasked\n" +
				"other.service\t\t\t\t\tenabled\n"), nil, 0, nil
		}
		return nil, nil, 0, nil
	})

	chk, found := check.FindCheckByName("srv-masked")
	require.True(t, found, "Services Masked Check not found")
	config := buildConfig(&blueprint.Customizations{
		Services: &blueprint.ServicesCustomization{
			Masked: []string{"test.service"},
		},
	})

	require.NoError(t, chk.Func(chk.Meta, config))
}
