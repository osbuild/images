package disk_test

import (
	"testing"

	"github.com/osbuild/images/pkg/disk"
)

func TestImplementsInterfacesCompileTimeCheckLUKS(t *testing.T) {
	var _ = disk.Container(&disk.LUKSContainer{})
}
