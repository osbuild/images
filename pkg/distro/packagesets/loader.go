package packagesets

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/experimentalflags"
	"github.com/osbuild/images/pkg/rpmmd"
)

//go:embed */*.yaml
var DataFS embed.FS

type packageSet struct {
	Include   []string    `yaml:"include"`
	Exclude   []string    `yaml:"exclude"`
	Condition *conditions `yaml:"condition,omitempty"`
}

type conditions struct {
	Architecture          map[string]packageSet `yaml:"architecture,omitempty"`
	VersionLessThan       map[string]packageSet `yaml:"version_less_than,omitempty"`
	VersionGreaterOrEqual map[string]packageSet `yaml:"version_greater_or_equal,omitempty"`
}

func Load(it distro.ImageType, replacements map[string]string) rpmmd.PackageSet {
	typeName := strings.ReplaceAll(it.Name(), "-", "_")

	arch := it.Arch()
	archName := arch.Name()
	distribution := arch.Distro()
	distroNameVer := distribution.Name()
	// use rsplit() here, for "centos-stream-10" like distro names
	distroName := distroNameVer[:strings.LastIndex(distroNameVer, "-")]
	distroVersion := distribution.OsVersion()

	// XXX: this is a short term measure, pass a set of
	// searchPaths down the stack instead
	var dataFS fs.FS = DataFS
	if overrideDir := experimentalflags.String("yamldir"); overrideDir != "" {
		dataFS = os.DirFS(overrideDir)
	}
	f, err := dataFS.Open(filepath.Join(distroName, "package_sets.yaml"))
	if err != nil {
		panic(err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
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
