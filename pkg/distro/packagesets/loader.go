package packagesets

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/rpmmd"
	"gopkg.in/yaml.v3"
)

//go:embed */*.yaml
var data embed.FS

var DataFS fs.FS = data

type packageSet struct {
	Include   []string    `yaml:"include"`
	Exclude   []string    `yaml:"exclude"`
	Condition *conditions `yaml:"condition,omitempty"`
}

type conditions struct {
	Architecture          map[string]packageSet `yaml:"architecture,omitempty"`
	VersionLessThan       map[string]packageSet `yaml:"version_less_than,omitempty"`
	VersionGreaterOrEqual map[string]packageSet `yaml:"version_greater_or_equal,omitempty"`
	DistroName            map[string]packageSet `yaml:"distro_name,omitempty"`
}

func Load(it distro.ImageType, replacements map[string]string) rpmmd.PackageSet {
	typeName := strings.ReplaceAll(it.Name(), "-", "_")

	arch := it.Arch()
	archName := arch.Name()
	distribution := arch.Distro()
	distroNameVer := distribution.Name()
	// we need to split from the right for "centos-stream-10" like
	// distro names, sadly go has no rsplit() so we do it manually
	distroName := distroNameVer[:strings.LastIndex(distroNameVer, "-")]
	distroVersion := distribution.OsVersion()

	distroSets, err := DataFS.Open(filepath.Join(distroName, "package_sets.yaml"))
	if err != nil {
		panic(err)
	}

	decoder := yaml.NewDecoder(distroSets)
	decoder.KnownFields(true)

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
