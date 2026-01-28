package distro_test

import (
	"testing"

	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distrofactory"
)

var distroArchPairs = []struct {
	distro string
	arch   string
}{
	{"rhel-10.0", "x86_64"},
	{"fedora-42", "aarch64"},
}

func BenchmarkDistro(b *testing.B) {
	factory := distrofactory.NewDefault()
	if factory == nil {
		b.Fatal("distrofactory.NewDefault() returned nil")
	}

	var d distro.Distro
	for _, pair := range distroArchPairs {
		b.Run("GetDistro/"+pair.distro+"-"+pair.arch, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				d = factory.GetDistro(pair.distro)
			}
		})

		b.Run("GetArch/"+pair.distro+"-"+pair.arch, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = d.GetArch(pair.arch)
			}
		})

		b.Run("GetImageTypes/"+pair.distro+"-"+pair.arch, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = distro.GetImageTypes(d, pair.arch)
			}
		})
	}
}
