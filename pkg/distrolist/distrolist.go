package distrolist

import (
	"fmt"

	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/fedora"
	"github.com/osbuild/images/pkg/distro/rhel7"
	"github.com/osbuild/images/pkg/distro/rhel8"
	"github.com/osbuild/images/pkg/distro/rhel9"
)

type Factory func(name string) distro.Distro

type List struct {
	factories []Factory
}

func New(factories []Factory) List {
	return List{factories: factories}
}

func NewDefault() List {
	return List{factories: []Factory{
		fedora.New,
		rhel7.NewFromID,
		rhel8.NewFromID,
		rhel9.NewFromID,
	}}
}

func (r *List) GetDistro(name string) distro.Distro {
	var match *distro.Distro
	for _, f := range r.factories {
		if d := f(name); d != nil {
			if match != nil {
				panic(fmt.Sprintf("distro ID was matched by multiple distro factories: %v, %v", *match, d))
			}

			match = &d
		}
	}

	if match == nil {
		return nil
	}

	return *match
}
