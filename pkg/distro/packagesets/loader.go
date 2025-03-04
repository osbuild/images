package packagesets

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/rpmmd"
	"gopkg.in/yaml.v3"
)

//go:embed */*.yaml
var Data embed.FS

type packageSet struct {
	Include   []string    `yaml:"include"`
	Exclude   []string    `yaml:"exclude"`
	Condition *conditions `yaml:"condition,omitempty"`

	// XXX: terrible name, needs to convey that this is does not merge
	// conditions
	AddIncExc []packageSet `yaml:"add_includes_excludes,omitempty"`
}

type conditions struct {
	Architecture          map[string]packageSet `yaml:"architecture,omitempty"`
	VersionLessThan       map[string]packageSet `yaml:"version_less_than,omitempty"`
	VersionGreaterOrEqual map[string]packageSet `yaml:"version_greater_or_equal,omitempty"`
	DistroName            map[string]packageSet `yaml:"distro_name,omitempty"`
}

func Load(it distro.ImageType, replacements map[string]string) rpmmd.PackageSet {
	arch := it.Arch()
	distro := arch.Distro()
	return LoadByNames(distro.Name(), arch.Name(), it.Name(), replacements)
}

// XXX: fugly, we need this to workaround the fact that image installers
// has two package sets and we load by image type not by package set
func LoadByNames(distroNameVer, archName, imgType string, replacements map[string]string) rpmmd.PackageSet {
	typeName := strings.ReplaceAll(imgType, "-", "_")

	distroName := strings.SplitN(distroNameVer, "-", 2)[0]
	distroNameMajorVer := strings.SplitN(distroNameVer, ".", 2)[0]
	distroVersion := strings.SplitN(distroNameVer, "-", 2)[1]

	searchPath := []string{
		filepath.Join(distroNameMajorVer, "package_sets.yaml"),
		filepath.Join(distroName, "package_sets.yaml"),
	}
	// XXX: fugly, symlinks would be nice but not supported via
	// go:embed, we need a way so that the distro can declare what
	// its an alias for
	if distroName == "centos" {
		searchPath = []string{
			filepath.Join(fmt.Sprintf("rhel-%s", distroVersion), "package_sets.yaml"),
			filepath.Join("rhel", "package_sets.yaml"),
		}
	}

	var decoder *yaml.Decoder
	for _, p := range searchPath {
		f, err := Data.Open(p)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			panic(err)
		}
		defer f.Close()

		decoder = yaml.NewDecoder(f)
		decoder.KnownFields(true)
		break
	}
	if decoder == nil {
		panic(fmt.Errorf("cannot find package_set in %v", searchPath))
	}

	var pkgSets map[string]packageSet
	if err := decoder.Decode(&pkgSets); err != nil {
		panic(err)
	}

	pkgSet, ok := pkgSets[typeName]
	if !ok {
		panic(fmt.Sprintf("unknown package set name %q", typeName))
	}
	rpmmdPkgSet := rpmmd.PackageSet{
		Include: pkgSet.Include,
		Exclude: pkgSet.Exclude,
	}

	for _, addSet := range pkgSet.AddIncExc {
		rpmmdPkgSet = rpmmdPkgSet.Append(rpmmd.PackageSet{
			Include: addSet.Include,
			Exclude: addSet.Exclude,
		})
	}

	if pkgSet.Condition != nil {
		// process conditions
		if archSet, ok := pkgSet.Condition.Architecture[archName]; ok {
			rpmmdPkgSet = rpmmdPkgSet.Append(rpmmd.PackageSet{
				Include: archSet.Include,
				Exclude: archSet.Exclude,
			})
		}
		if distroNameSet, ok := pkgSet.Condition.DistroName[distroName]; ok {
			rpmmdPkgSet = rpmmdPkgSet.Append(rpmmd.PackageSet{
				Include: distroNameSet.Include,
				Exclude: distroNameSet.Exclude,
			})
		}

		for ltVer, ltSet := range pkgSet.Condition.VersionLessThan {
			if r, ok := replacements[ltVer]; ok {
				ltVer = r
			}
			if common.VersionLessThan(distroVersion, ltVer) {
				rpmmdPkgSet = rpmmdPkgSet.Append(rpmmd.PackageSet{
					Include: ltSet.Include,
					Exclude: ltSet.Exclude,
				})
			}
		}

		for gteqVer, gteqSet := range pkgSet.Condition.VersionGreaterOrEqual {
			if r, ok := replacements[gteqVer]; ok {
				gteqVer = r
			}
			if common.VersionGreaterThanOrEqual(distroVersion, gteqVer) {
				rpmmdPkgSet = rpmmdPkgSet.Append(rpmmd.PackageSet{
					Include: gteqSet.Include,
					Exclude: gteqSet.Exclude,
				})
			}
		}
	}

	return rpmmdPkgSet
}
