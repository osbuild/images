package check_test

import (
	"testing"

	"github.com/osbuild/blueprint/pkg/blueprint"
	check "github.com/osbuild/images/cmd/check-host-config/check"
	"github.com/osbuild/images/internal/test"
	"github.com/stretchr/testify/require"
)

func TestFilesCheck(t *testing.T) {
	test.MockGlobal(t, &check.Exists, func(name string) bool {
		return true
	})

	chk, found := check.FindCheckByName("Files Check")
	require.True(t, found, "Files Check not found")
	config := buildConfig(&blueprint.Customizations{
		Files: []blueprint.FileCustomization{
			{Path: "/etc/testfile"},
		},
	})

	require.NoError(t, chk.Func(chk.Meta, config))
}
