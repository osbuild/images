package ukidirect

import (
	_ "embed"
	"os"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/customizations/fsnode"
)

//go:embed 99-uki-uefi-setup.install
var kernelInstallScript string

func KernelInstallScript() (*fsnode.File, error) {
	return fsnode.NewFile("/etc/kernel/install.d/99-uki-uefi-setup.install", common.ToPtr(os.FileMode(0755)), nil, nil, []byte(kernelInstallScript))
}
