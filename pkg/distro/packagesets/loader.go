package packagesets

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/rpmmd"
)

//go:embed */*.yaml
var data embed.FS

var DataFS fs.FS = data

type toplevelYAML struct {
	ImageTypes map[string]imageType `yaml:"image_types"`
	Common     map[string]any       `yaml:".common,omitempty"`
}

type imageType struct {
	PackageSets []packageSet `yaml:"package_sets"`
}

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

// Load loads the PackageSet from the yaml source file discovered via the
// imagetype. By default the imagetype name is used to load the packageset
// but with "overrideTypeName" this can be overriden (useful for e.g.
// installer image types).
func Load(it distro.ImageType, overrideTypeName string, replacements map[string]string) rpmmd.PackageSet {
	typeName := it.Name()
	if overrideTypeName != "" {
		typeName = overrideTypeName
	}
	typeName = strings.ReplaceAll(typeName, "-", "_")

	arch := it.Arch()
	archName := arch.Name()
	distribution := arch.Distro()
	distroNameVer := distribution.Name()
	// we need to split from the right for "centos-stream-10" like
	// distro names, sadly go has no rsplit() so we do it manually
	// XXX: we cannot use distroidparser here because of import cycles
	distroName := distroNameVer[:strings.LastIndex(distroNameVer, "-")]
	distroVersion := strings.SplitN(distroNameVer, "-", 2)[1]
	distroNameMajorVer := strings.SplitN(distroNameVer, ".", 2)[0]

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
		f, err := DataFS.Open(p)
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

	// each imagetype can have multiple package sets, so that we can
	// use yaml aliases/anchors to de-duplicate them
	var toplevel toplevelYAML
	if err := decoder.Decode(&toplevel); err != nil {
		panic(err)
	}

	imgType, ok := toplevel.ImageTypes[typeName]
	if !ok {
		panic(fmt.Errorf("unknown image type name %q", typeName))
	}

	var rpmmdPkgSet rpmmd.PackageSet
	for _, pkgSet := range imgType.PackageSets {
		rpmmdPkgSet = rpmmdPkgSet.Append(rpmmd.PackageSet{
			Include: pkgSet.Include,
			Exclude: pkgSet.Exclude,
		})

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
	}
	// mostly for tests
	sort.Strings(rpmmdPkgSet.Include)
	sort.Strings(rpmmdPkgSet.Exclude)

	return rpmmdPkgSet
}
