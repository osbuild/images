package testdistro

import (
	"github.com/google/go-cmp/cmp"
	"github.com/osbuild/images/pkg/distro"
)

// CompareImageTypes considers two image type objects equal if and only if the names of their distro/arch/imagetype
// are. The thinking is that the objects are static, and resolving by these three keys should always give equivalent
// objects. Whether we actually have object equality, is an implementation detail, so we don't want to rely on that.
func CompareImageTypes() cmp.Option {
	return cmp.Comparer(func(x, y distro.ImageType) bool {
		return x.Name() == y.Name() &&
			x.Arch().Name() == y.Arch().Name() &&
			x.Arch().Distro().Name() == y.Arch().Distro().Name()
	})
}
