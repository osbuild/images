// package defs contain the distro definitions used by the "images" library
package defs

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"reflect"
	"slices"

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

type imageType struct {
	PackageSets []packageSet `yaml:"package_sets"`
	// archStr->partitionTable
	PartitionTables map[string]*disk.PartitionTable `yaml:"partition_table"`
	// override specific aspects of the partition table
	PartitionTablesOverrides *partitionTablesOverrides `yaml:"partition_table_override"`
}

type packageSet struct {
	Include   []string          `yaml:"include"`
	Exclude   []string          `yaml:"exclude"`
	Condition *pkgSetConditions `yaml:"condition,omitempty"`
}

type pkgSetConditions struct {
	Architecture          map[string]packageSet `yaml:"architecture,omitempty"`
	VersionLessThan       map[string]packageSet `yaml:"version_less_than,omitempty"`
	VersionGreaterOrEqual map[string]packageSet `yaml:"version_greater_or_equal,omitempty"`
	DistroName            map[string]packageSet `yaml:"distro_name,omitempty"`
}

type partitionTablesOverrides struct {
	Conditional partitionTablesOverwriteConditional `yaml:"condition"`
}

func (po *partitionTablesOverrides) Apply(it distro.ImageType, pt *disk.PartitionTable, replacements map[string]string) error {
	if po == nil {
		return nil
	}
	cond := po.Conditional
	// XXX: should we strings.Replace("-", "_") for distroName too?
	distroName, distroVersion := splitDistroNameVer(it.Arch().Distro().Name())

	if distroNameOverrides, ok := cond.DistroName[distroName]; ok {
		for _, overrideOp := range distroNameOverrides {
			if err := overrideOp.Apply(it, pt); err != nil {
				return err
			}
		}
	}
	for ltVer, ltOverrides := range cond.VersionLessThan {
		if r, ok := replacements[ltVer]; ok {
			ltVer = r
		}
		if common.VersionLessThan(distroVersion, ltVer) {
			for _, overrideOp := range ltOverrides {
				if err := overrideOp.Apply(it, pt); err != nil {
					return err
				}
			}
		}
	}
	for gteqVer, geOverrides := range cond.VersionGreaterOrEqual {
		if r, ok := replacements[gteqVer]; ok {
			gteqVer = r
		}
		if common.VersionGreaterThanOrEqual(distroVersion, gteqVer) {
			for _, overrideOp := range geOverrides {
				if err := overrideOp.Apply(it, pt); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

type partitionTablesOverwriteConditional struct {
	VersionLessThan       map[string][]partitionTablesOverrideOp `yaml:"version_less_than,omitempty"`
	VersionGreaterOrEqual map[string][]partitionTablesOverrideOp `yaml:"version_greater_or_equal,omitempty"`
	DistroName            map[string][]partitionTablesOverrideOp `yaml:"distro_name,omitempty"`
}

type partitionTablesOverrideOp map[string]interface{}

func findElementIndexByJSONTag(t reflect.Type, needle string) int {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag
		if jsonTag, ok := tag.Lookup("json"); ok {
			if strings.Split(jsonTag, ",")[0] == needle {
				return i
			}
		}
	}
	return -1
}

func (op partitionTablesOverrideOp) applyTo(val reflect.Value) ([]string, error) {
	var consumed []string

	for jsonTag, override := range op {
		fieldIdx := findElementIndexByJSONTag(val.Type(), jsonTag)
		if fieldIdx >= 0 {
			field := val.Field(fieldIdx)
			newVal := reflect.ValueOf(override)
			if !newVal.CanConvert(field.Type()) {
				return nil, fmt.Errorf("cannot convert override %q (%T) to %s", override, override, field.Type())
			}
			field.Set(newVal.Convert(field.Type()))
			consumed = append(consumed, jsonTag)
		}
	}
	return consumed, nil
}

var partitionSelectors = map[string]bool{
	// special token(s) that are always part of the overrides
	"partition_index":       true,
	"partition_mount_point": true,
	"partition_selection":   true,
	// XXX: something is broken here, this should lead to errors
	// but it does not for some reason
	//"action":                true,
	//"partition_arch_only": true,
}

func (op partitionTablesOverrideOp) checkAllConsumed(consumed ...[]string) error {
	// collect all consumed overrides
	seen := map[string]bool{}
	for _, cn := range consumed {
		for _, jsonTag := range cn {
			seen[jsonTag] = true
		}
	}
	// check if we have some overrides left that are not applied, this means
	// there was no json tag
	for override := range op {
		if !seen[override] && !partitionSelectors[override] {
			return fmt.Errorf("cannot find %q in partition", override)
		}
	}

	return nil
}

func (op partitionTablesOverrideOp) maybeReturnNotFoundError(notFoundErr error) (int, error) {
	selectModeIf, ok := op["partition_selection"]
	if ok {
		selectMode, ok := selectModeIf.(string)
		if ok {
			if selectMode == "ignore-missing" {
				return -1, nil
			}
		}
	}

	return 0, notFoundErr
}

func (op partitionTablesOverrideOp) findSelectedPart(pt *disk.PartitionTable) (int, error) {
	selectPartIf, ok := op["partition_index"]
	if ok {
		selectPart, ok := selectPartIf.(int)
		if !ok {
			return -1, fmt.Errorf("partition_index must be int, got %T", selectPartIf)
		}
		if selectPart > len(pt.Partitions) {
			return op.maybeReturnNotFoundError(fmt.Errorf("override %q part %v outside of partitionTable %+v", op, selectPart, pt))
		}
		return selectPart, nil
	}
	selectMpIf, ok := op["partition_mount_point"]
	if ok {
		selectMp, ok := selectMpIf.(string)
		if !ok {
			return -1, fmt.Errorf("partition_mount_point must be string, got %T", selectMpIf)
		}
		// XXX: we cannot use disk.FindMountable() here as it does
		// not expose the "entityPath"
		for idx, part := range pt.Partitions {
			mt, ok := part.Payload.(disk.Mountable)
			if !ok {
				continue
			}
			if mt.GetMountpoint() == selectMp {
				return idx, nil
			}
		}
		return op.maybeReturnNotFoundError(fmt.Errorf("cannot find mount_point %q in %+v: note that nested mounts inside luks/lvm/btrfs are currently unsupported, please report a bug if you need this", selectMp, pt))
	}

	return -1, fmt.Errorf("no partition selector found, please provide one of %q", slices.Sorted(maps.Keys(partitionSelectors)))
}

func (op partitionTablesOverrideOp) Apply(it distro.ImageType, pt *disk.PartitionTable) error {
	selectPart, err := op.findSelectedPart(pt)
	if err != nil {
		return err
	}
	// not finding the selection is not always an error
	if selectPart < 0 {
		return nil
	}

	if archOnlyIf, ok := op["partition_arch_only"]; ok {
		if archOnly, ok := archOnlyIf.(string); ok {
			if archOnly != it.Arch().Name() {
				return nil
			}
		}
	}

	if actionIf, ok := op["action"]; ok {
		if action, ok := actionIf.(string); ok {
			if action == "delete" {
				pt.Partitions = slices.Delete(pt.Partitions, selectPart, selectPart+1)
				return nil
			}
		}
	}

	// try to apply to partition
	part := pt.Partitions[selectPart]
	val := reflect.ValueOf(&part).Elem()
	consumedPart, err := op.applyTo(val)
	if err != nil {
		return err
	}
	pt.Partitions[selectPart] = part

	// try to apply to payload
	var consumedPayload []string
	switch payload := part.Payload.(type) {
	case nil:
		// nothing to do
	case *disk.Filesystem:
		val := reflect.ValueOf(&payload).Elem().Elem()
		consumedPayload, err = op.applyTo(val)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported override payload: %T", payload)
	}
	if err := op.checkAllConsumed(consumedPart, consumedPayload); err != nil {
		return err
	}

	return nil
}

// PackageSet loads the PackageSet from the yaml source file discovered via the
// imagetype. By default the imagetype name is used to load the packageset
// but with "overrideTypeName" this can be overriden (useful for e.g.
// installer image types).
func PackageSet(it distro.ImageType, overrideTypeName string, replacements map[string]string) (rpmmd.PackageSet, error) {
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

	// each imagetype can have multiple package sets, so that we can
	// use yaml aliases/anchors to de-duplicate them
	toplevel, err := load(distroNameVer)
	if err != nil {
		return rpmmd.PackageSet{}, err
	}

	imgType, ok := toplevel.ImageTypes[typeName]
	if !ok {
		return rpmmd.PackageSet{}, fmt.Errorf("unknown image type name %q", typeName)
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

var (
	ErrNoPartitionTableForImgType = errors.New("no partition table for image type")
	ErrNoPartitionTableForArch    = errors.New("no partition table for arch")
)

// PartitionTable returns the partionTable for the given distro/imgType.
func PartitionTable(it distro.ImageType, replacements map[string]string) (*disk.PartitionTable, error) {
	distroNameVer := it.Arch().Distro().Name()
	typeName := strings.ReplaceAll(it.Name(), "-", "_")

	toplevel, err := load(distroNameVer)
	if err != nil {
		return nil, err
	}

	imgType, ok := toplevel.ImageTypes[typeName]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrNoPartitionTableForImgType, typeName)
	}
	arch := it.Arch()
	archName := arch.Name()

	pt, ok := imgType.PartitionTables[archName]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrNoPartitionTableForArch, archName)
	}

	if err := imgType.PartitionTablesOverrides.Apply(it, pt, replacements); err != nil {
		return nil, err
	}

	return pt, nil
}

func splitDistroNameVer(distroNameVer string) (string, string) {
	// we need to split from the right for "centos-stream-10" like
	// distro names, sadly go has no rsplit() so we do it manually
	// XXX: we cannot use distroidparser here because of import cycles
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
