package disk_test

import (
	"testing"

	"github.com/osbuild/images/pkg/disk"
)

func TestImplementsInterfacesCompileTimeCheckRaw(t *testing.T) {
	var _ = disk.PayloadEntity(&disk.Raw{})
}
