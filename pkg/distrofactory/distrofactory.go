package distrofactory

import (
	"fmt"

	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/fedora"
	"github.com/osbuild/images/pkg/distro/rhel7"
	"github.com/osbuild/images/pkg/distro/rhel8"
	"github.com/osbuild/images/pkg/distro/rhel9"
	"github.com/osbuild/images/pkg/distro/test_distro"
)

// FactoryFunc is a function that returns a distro.Distro for a given distro
// represented as a string. If the string does not represent a distro, that can
// be detected by the factory, it should return nil.
type FactoryFunc func(idStr string) distro.Distro

// Factory is a list of distro.Distro factories.
type Factory struct {
	factories []FactoryFunc
}

// GetDistro returns the distro.Distro that matches the given distro ID. If no
// distro.Distro matches the given distro ID, it returns nil. If multiple distro
// factories match the given distro ID, it panics.
func (f *Factory) GetDistro(name string) distro.Distro {
	var match distro.Distro
	for _, f := range f.factories {
		if d := f(name); d != nil {
			if match != nil {
				panic(fmt.Sprintf("distro ID was matched by multiple distro factories: %v, %v", match, d))
			}
			match = d
		}
	}

	return match
}

// New returns a Factory of distro.Distro factories for the given distros.
func New(factories ...FactoryFunc) *Factory {
	return &Factory{factories: factories}
}

// NewDefault returns a Factory of distro.Distro factories for all supported
// distros.
func NewDefault() *Factory {
	return New(
		fedora.DistroFactory,
		rhel7.DistroFactory,
		rhel8.DistroFactory,
		rhel9.DistroFactory,
	)
}

// NewTestDefault returns a Factory of distro.Distro factory for the test_distro.
func NewTestDefault() *Factory {
	return New(
		test_distro.DistroFactory,
	)
}
