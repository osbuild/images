package fedora

import (
	"fmt"
	"sort"

	// we cannot use "maps" yet, as it needs go1.23
	"golang.org/x/exp/maps"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/customizations/oscap"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distro/defs"
	"github.com/osbuild/images/pkg/platform"
)

const (
	// package set names

	// main/common os image package set name
	osPkgsKey = "os"

	// container package set name
	containerPkgsKey = "container"

	// installer package set name
	installerPkgsKey = "installer"

	// blueprint package set name
	blueprintPkgsKey = "blueprint"
)

var (
	oscapProfileAllowList = []oscap.Profile{
		oscap.Ospp,
		oscap.PciDss,
		oscap.Standard,
	}
)

// distribution implements the distro.Distro interface
var _ = distro.Distro(&distribution{})

type distribution struct {
	defs.DistroYAML

	arches map[string]*architecture
	// XXX: move into defs.DistroYAML
	defaultImageConfig *distro.ImageConfig
}

func getISOLabelFunc(variant string) isoLabelFunc {
	const ISO_LABEL = "%s-%s-%s-%s"

	return func(t *imageType) string {
		return fmt.Sprintf(ISO_LABEL, t.Arch().Distro().Product(), t.Arch().Distro().OsVersion(), variant, t.Arch().Name())
	}

}

func newDistro(version int) (distro.Distro, error) {
	distros := common.Must(defs.Distros())
	nameVer := fmt.Sprintf("fedora-%d", version)
	distroYAML, ok := distros[nameVer]
	if !ok {
		err := fmt.Errorf("cannot find %s in %q", nameVer, maps.Keys(distros))
		panic(err)
	}

	rd := &distribution{
		DistroYAML: distroYAML,
		// move into distroYAML
		defaultImageConfig: common.Must(defs.DistroImageConfig(nameVer)),
		arches:             make(map[string]*architecture),
	}

	its, err := defs.ImageTypes(rd.Name())
	if err != nil {
		return nil, err
	}
	for _, imgTypeYAML := range its {
		// use as marker for images that are not converted to
		// YAML yet
		if imgTypeYAML.Filename == "" {
			continue
		}
		for _, pl := range imgTypeYAML.Platforms {
			ar, ok := rd.arches[pl.Arch.String()]
			if !ok {
				ar = newArchitecture(rd, pl.Arch.String())
				rd.arches[pl.Arch.String()] = ar
			}
			it := newImageTypeFrom(rd, ar, imgTypeYAML)
			if err := ar.addImageType(&pl, it); err != nil {
				return nil, err
			}
		}
	}

	return rd, nil
}

func (d *distribution) Name() string {
	return d.DistroYAML.Name
}

func (d *distribution) Codename() string {
	return d.DistroYAML.Codename
}

func (d *distribution) Releasever() string {
	return d.DistroYAML.ReleaseVersion
}

func (d *distribution) OsVersion() string {
	return d.DistroYAML.ReleaseVersion
}

func (d *distribution) Product() string {
	return d.DistroYAML.Product
}

func (d *distribution) ModulePlatformID() string {
	return d.DistroYAML.ModulePlatformID
}

func (d *distribution) OSTreeRef() string {
	return d.DistroYAML.OSTreeRefTmpl
}

func (d *distribution) ListArches() []string {
	archNames := make([]string, 0, len(d.arches))
	for name := range d.arches {
		archNames = append(archNames, name)
	}
	sort.Strings(archNames)
	return archNames
}

func (d *distribution) GetArch(name string) (distro.Arch, error) {
	arch, exists := d.arches[name]
	if !exists {
		return nil, fmt.Errorf("invalid architecture: %v", name)
	}
	return arch, nil
}

// architecture implements the distro.Arch interface
var _ = distro.Arch(&architecture{})

type architecture struct {
	distro           *distribution
	name             string
	imageTypes       map[string]distro.ImageType
	imageTypeAliases map[string]string
}

func newArchitecture(rd *distribution, name string) *architecture {
	return &architecture{
		distro:           rd,
		name:             name,
		imageTypes:       make(map[string]distro.ImageType),
		imageTypeAliases: make(map[string]string),
	}
}

func (a *architecture) Name() string {
	return a.name
}

func (a *architecture) ListImageTypes() []string {
	itNames := make([]string, 0, len(a.imageTypes))
	for name := range a.imageTypes {
		itNames = append(itNames, name)
	}
	sort.Strings(itNames)
	return itNames
}

func (a *architecture) GetImageType(name string) (distro.ImageType, error) {
	t, exists := a.imageTypes[name]
	if !exists {
		aliasForName, exists := a.imageTypeAliases[name]
		if !exists {
			return nil, fmt.Errorf("invalid image type: %v", name)
		}
		t, exists = a.imageTypes[aliasForName]
		if !exists {
			panic(fmt.Sprintf("image type '%s' is an alias to a non-existing image type '%s'", name, aliasForName))
		}
	}
	return t, nil
}

func (a *architecture) addImageType(platform platform.Platform, it imageType) error {
	it.arch = a
	it.platform = platform
	a.imageTypes[it.name] = &it
	for _, alias := range it.nameAliases {
		if a.imageTypeAliases == nil {
			a.imageTypeAliases = map[string]string{}
		}
		if existingAliasFor, exists := a.imageTypeAliases[alias]; exists {
			return fmt.Errorf("image type alias '%s' for '%s' is already defined for another image type '%s'", alias, it.name, existingAliasFor)
		}
		a.imageTypeAliases[alias] = it.name
	}
	return nil
}

func (a *architecture) Distro() distro.Distro {
	return a.distro
}

func ParseID(idStr string) (*distro.ID, error) {
	id, err := distro.ParseID(idStr)
	if err != nil {
		return nil, err
	}

	if id.Name != "fedora" {
		return nil, fmt.Errorf("invalid distro name: %s", id.Name)
	}

	if id.MinorVersion != -1 {
		return nil, fmt.Errorf("fedora distro does not support minor versions")
	}

	return id, nil
}

func DistroFactory(idStr string) distro.Distro {
	id, err := ParseID(idStr)
	if err != nil {
		return nil
	}

	return common.Must(newDistro(id.MajorVersion))
}
