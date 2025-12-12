package check_test

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/osbuild/blueprint/pkg/blueprint"
	check "github.com/osbuild/images/cmd/check-host-config/check"
	"github.com/osbuild/images/cmd/check-host-config/mockos"
	"github.com/osbuild/images/internal/buildconfig"
)

func TestFilesCheck(t *testing.T) {
	ctx := mockos.WithExistsFunc(context.Background(), func(name string) bool {
		return true
	})

	check := check.FilesCheck{}
	config := &buildconfig.BuildConfig{
		Blueprint: &blueprint.Blueprint{
			Customizations: &blueprint.Customizations{
				Files: []blueprint.FileCustomization{
					{Path: "/etc/testfile"},
				},
			},
		},
	}

	err := check.Run(ctx, log.New(os.Stdout, "", 0), config)
	if err != nil {
		t.Fatalf("FilesCheck failed: %v", err)
	}
}
