// package defs contain the distro definitions used by the "images" library
package defs

import (
	"embed"
	"fmt"
	"io/fs"
	"os"

	"path/filepath"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/experimentalflags"
	"github.com/osbuild/images/pkg/rpmmd"
)

//go:embed */*.yaml
var data embed.FS

var DataFS fs.FS = data

type toplevelYAML struct {
	ImageTypes map[string]imageType `yaml:"image_types"`
	Common     map[string]any       `yaml:".common,omitempty"`
}

func (t *toplevelYAML) ImageType(it distro.ImageType, overrideTypeName string, replacements map[string]string) (*imageType, error) {
	typeName := it.Name()
	if overrideTypeName != "" {
		typeName = overrideTypeName
	}
	typeName = strings.ReplaceAll(typeName, "-", "_")
	arch := it.Arch()
	archName := arch.Name()
	distribution := arch.Distro()
	distroNameVer := distribution.Name()
	distroName, distroVersion := splitDistroNameVer(distroNameVer)

	imgType, ok := t.ImageTypes[typeName]
	if !ok {
		return nil, fmt.Errorf("unknown image type name %q", typeName)
	}

	if imgType.Condition != nil {
		if imgType, ok := imgType.Condition.Architecture[archName]; ok {
			return &imgType, nil
		}
		if imgType, ok := imgType.Condition.DistroName[distroName]; ok {
			return &imgType, nil
		}
		for ltVer, imgType := range imgType.Condition.VersionLessThan {
			if r, ok := replacements[ltVer]; ok {
				ltVer = r
			}
			if common.VersionLessThan(distroVersion, ltVer) {
				return &imgType, nil
			}
		}
		for gteqVer, imgType := range imgType.Condition.VersionGreaterOrEqual {
			if r, ok := replacements[gteqVer]; ok {
				gteqVer = r
			}
			if common.VersionGreaterThanOrEqual(distroVersion, gteqVer) {
				return &imgType, nil
			}
		}
	}

	return &imgType, nil
}

type imageType struct {
	// archStr->partitionTable
	PartitionTables map[string]*disk.PartitionTable `yaml:"partition_table"`
	PackageSets     []packageSet                    `yaml:"package_sets"`
	Condition       *conditionsImageType            `yaml:"condition,omitempty"`
}

type conditionsImageType struct {
	Architecture          map[string]imageType `yaml:"architecture,omitempty"`
	VersionLessThan       map[string]imageType `yaml:"version_less_than,omitempty"`
	VersionGreaterOrEqual map[string]imageType `yaml:"version_greater_or_equal,omitempty"`
	DistroName            map[string]imageType `yaml:"distro_name,omitempty"`
}

type packageSet struct {
	Include   []string              `yaml:"include"`
	Exclude   []string              `yaml:"exclude"`
	Condition *conditionsPackageSet `yaml:"condition,omitempty"`
}

// XXX: unify this with conditionImageType?
type conditionsPackageSet struct {
	Architecture          map[string]packageSet `yaml:"architecture,omitempty"`
	VersionLessThan       map[string]packageSet `yaml:"version_less_than,omitempty"`
	VersionGreaterOrEqual map[string]packageSet `yaml:"version_greater_or_equal,omitempty"`
	DistroName            map[string]packageSet `yaml:"distro_name,omitempty"`
}

// PackageSet loads the PackageSet from the yaml source file discovered via the
// imagetype. By default the imagetype name is used to load the packageset
// but with "overrideTypeName" this can be overriden (useful for e.g.
// installer image types).
func PackageSet(it distro.ImageType, overrideTypeName string, replacements map[string]string) (rpmmd.PackageSet, error) {
	arch := it.Arch()
	archName := arch.Name()
	distribution := arch.Distro()
	distroNameVer := distribution.Name()
	distroName, distroVersion := splitDistroNameVer(distroNameVer)

	// each imagetype can have multiple package sets, so that we can
	// use yaml aliases/anchors to de-duplicate them
	toplevel, err := load(distroNameVer)
	if err != nil {
		return rpmmd.PackageSet{}, err
	}

	imgType, err := toplevel.ImageType(it, overrideTypeName, replacements)
	if err != nil {
		return rpmmd.PackageSet{}, err
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

	return rpmmdPkgSet, nil
}

// PartitionTable returns the partionTable for the given distro/imgType.
func PartitionTable(it distro.ImageType /*, replacements map[string]string*/) (*disk.PartitionTable, error) {
	// XXX: port to add support for replacements
	var replacements map[string]string
	distroNameVer := it.Arch().Distro().Name()

	toplevel, err := load(distroNameVer)
	if err != nil {
		return nil, err
	}

	imgType, err := toplevel.ImageType(it, "", replacements)
	if err != nil {
		return nil, err
	}
	arch := it.Arch()
	partTable, ok := imgType.PartitionTables[arch.Name()]
	if !ok {
		return nil, fmt.Errorf("no partition table for %q", arch)
	}

	return partTable, nil
}

func splitDistroNameVer(distroNameVer string) (string, string) {
	idx := strings.LastIndex(distroNameVer, "-")
	return distroNameVer[:idx], distroNameVer[idx+1:]
}

func load(distroNameVer string) (*toplevelYAML, error) {
	// we need to split from the right for "centos-stream-10" like
	// distro names, sadly go has no rsplit() so we do it manually
	// XXX: we cannot use distroidparser here because of import cycles
	distroName, distroVersion := splitDistroNameVer(distroNameVer)
	distroNameMajorVer := strings.SplitN(distroNameVer, ".", 2)[0]

	// XXX: this is a short term measure, pass a set of
	// searchPaths down the stack instead
	var dataFS fs.FS = DataFS
	if overrideDir := experimentalflags.String("yamldir"); overrideDir != "" {
		logrus.Warnf("using experimental override dir %q", overrideDir)
		dataFS = os.DirFS(overrideDir)
	}

	// XXX: this is only needed temporary until we have a "distros.yaml"
	// that describes some high-level properties of each distro
	// (like their yaml dirs)
	var baseDir string
	switch distroName {
	case "rhel":
		// rhel yaml files are under ./rhel-$majorVer
		baseDir = distroNameMajorVer
	case "centos":
		// centos yaml is just rhel but we have (sadly) no symlinks
		// in "go:embed" so we have to have this slightly ugly
		// workaround
		baseDir = fmt.Sprintf("rhel-%s", distroVersion)
	case "fedora", "test-distro":
		// our other distros just have a single yaml dir per distro
		// and use condition.version_gt etc
		baseDir = distroName
	default:
		return nil, fmt.Errorf("unsupported distro in loader %q (add to loader.go)", distroName)
	}

	f, err := dataFS.Open(filepath.Join(baseDir, "distro.yaml"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	decoder.KnownFields(true)

	// each imagetype can have multiple package sets, so that we can
	// use yaml aliases/anchors to de-duplicate them
	var toplevel toplevelYAML
	if err := decoder.Decode(&toplevel); err != nil {
		return nil, err
	}

	return &toplevel, nil
}
