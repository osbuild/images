package disk_test

import (
	"testing"

	"github.com/osbuild/images/pkg/disk"
)

func TestImplementsInterfacesCompileTimeCheckFilesystem(t *testing.T) {
	var _ = disk.Mountable(&disk.Filesystem{})
	var _ = disk.UniqueEntity(&disk.Filesystem{})
	var _ = disk.FSTabEntity(&disk.Filesystem{})
}
