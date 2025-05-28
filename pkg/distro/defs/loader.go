// package defs contain the distro definitions used by the "images" library
package defs

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"text/template"

	"github.com/gobwas/glob"
	"github.com/hashicorp/go-version"
	"golang.org/x/exp/maps"
	"gopkg.in/yaml.v3"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/internal/environment"
	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/customizations/oscap"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/experimentalflags"
	"github.com/osbuild/images/pkg/olog"
	"github.com/osbuild/images/pkg/platform"
	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/osbuild/images/pkg/runner"
)

var (
	ErrImageTypeNotFound          = errors.New("image type not found")
	ErrNoPartitionTableForImgType = errors.New("no partition table for image type")
	ErrNoPartitionTableForArch    = errors.New("no partition table for arch")
)

//go:embed *.yaml */*.yaml
var data embed.FS

var defaultDataFS fs.FS = data

// distrosYAML defines all supported YAML based distributions
type distrosYAML struct {
	Distros []DistroYAML
}

func dataFS() fs.FS {
	// XXX: this is a short term measure, pass a set of
	// searchPaths down the stack instead
	var dataFS fs.FS = defaultDataFS
	if overrideDir := experimentalflags.String("yamldir"); overrideDir != "" {
		olog.Printf("WARNING: using experimental override dir %q", overrideDir)
		dataFS = os.DirFS(overrideDir)
	}
	return dataFS
}

type DistroYAML struct {
	// Match can be used to match multiple versions via a
	// fnmatch/glob style expression. We could also use a
	// regex and do something like:
	//   rhel-(?P<major>[0-9]+)\.(?P<minor>[0-9]+)
	// if we need to be more precise in the future, but for
	// now every match will be split into "$distroname-$major.$minor"
	// (with minor being optional)
	Match string `yaml:"match"`

	// The distro metadata, can contain go text template strings
	// for {{.Major}}, {{.Minor}} which will be expanded by the
	// upper layers.
	Name             string            `yaml:"name"`
	Codename         string            `yaml:"codename"`
	Vendor           string            `yaml:"vendor"`
	Preview          bool              `yaml:"preview"`
	OsVersion        string            `yaml:"os_version"`
	ReleaseVersion   string            `yaml:"release_version"`
	ModulePlatformID string            `yaml:"module_platform_id"`
	Product          string            `yaml:"product"`
	OSTreeRefTmpl    string            `yaml:"ostree_ref_tmpl"`
	Runner           runner.RunnerConf `yaml:"runner"`

	// ISOLabelTmpl can contain {{.Product}},{{.OsVersion}},{{.Arch}},{{.ImgTypeLabel}}
	ISOLabelTmpl string `yaml:"iso_label_tmpl"`

	DefaultFSType disk.FSType `yaml:"default_fs_type"`

	// directory with the actual image defintions, we separate that
	// so that we can point the "centos-10" distro to the "./rhel-10"
	// image types file/directory.
	DefsPath string `yaml:"defs_path"`

	BootstrapContainers map[arch.Arch]string `yaml:"bootstrap_containers"`

	OscapProfilesAllowList []oscap.Profile `yaml:"oscap_profiles_allowlist"`

	imageTypes map[string]ImageTypeYAML
}

func (d *DistroYAML) ImageTypes() map[string]ImageTypeYAML {
	return d.imageTypes
}

func (d *DistroYAML) runTemplates(nameVer string) error {
	id, err := distro.ParseID(nameVer)
	if err != nil {
		return err
	}

	var errs []error
	subs := func(inp string) string {
		var buf bytes.Buffer
		templ, err := template.New("").Parse(inp)
		if err != nil {
			errs = append(errs, err)
			return inp
		}
		if err := templ.Execute(&buf, id); err != nil {
			errs = append(errs, err)
			return inp
		}
		return buf.String()
	}
	d.Name = subs(d.Name)
	d.OsVersion = subs(d.OsVersion)
	d.ReleaseVersion = subs(d.ReleaseVersion)
	d.OSTreeRefTmpl = subs(d.OSTreeRefTmpl)
	d.ModulePlatformID = subs(d.ModulePlatformID)
	d.Runner.Name = subs(d.Runner.Name)
	for a := range d.BootstrapContainers {
		d.BootstrapContainers[a] = subs(d.BootstrapContainers[a])
	}

	return errors.Join(errs...)
}

// NewDistroYAML return the given distro or nil if the distro is not
// found. This mimics the "distrofactory.GetDistro() interface.
//
// Note that eventually we want something like "Distros()" instead
// that returns all known distros but for now we keep compatibility
// with the way distrofactory/reporegistry work which is by defining
// distros via repository files.
func NewDistroYAML(nameVer string) (*DistroYAML, error) {
	f, err := dataFS().Open("distros.yaml")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	decoder.KnownFields(true)

	var distros distrosYAML
	if err := decoder.Decode(&distros); err != nil {
		return nil, err
	}

	var foundDistro *DistroYAML
	for _, distro := range distros.Distros {
		if distro.Name == nameVer {
			foundDistro = &distro
			break
		}

		pat, err := glob.Compile(distro.Match)
		if err != nil {
			return nil, err
		}
		if pat.Match(nameVer) {
			if err := distro.runTemplates(nameVer); err != nil {
				return nil, err
			}

			foundDistro = &distro
			break
		}
	}
	if foundDistro == nil {
		return nil, nil
	}

	// load imageTypes
	f, err = dataFS().Open(filepath.Join(foundDistro.DefsPath, "distro.yaml"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var toplevel imageTypesYAML
	decoder = yaml.NewDecoder(f)
	decoder.KnownFields(true)
	if err := decoder.Decode(&toplevel); err != nil {
		return nil, err
	}
	if len(toplevel.ImageTypes) > 0 {
		foundDistro.imageTypes = make(map[string]ImageTypeYAML, len(toplevel.ImageTypes))
		for name := range toplevel.ImageTypes {
			v := toplevel.ImageTypes[name]
			v.name = name
			foundDistro.imageTypes[name] = v
		}
	}
	return foundDistro, nil
}

// XXX: make this part of DistroYAML
//
// DistroImageConfig returns the distro wide ImageConfig.
//
// Each ImageType gets this as their default ImageConfig.
func DistroImageConfig(distroNameVer string) (*distro.ImageConfig, error) {
	toplevel, err := load(distroNameVer)
	if err != nil {
		return nil, err
	}
	imgConfig := toplevel.ImageConfig.Default

	cond := toplevel.ImageConfig.Condition
	if cond != nil {
		id, err := distro.ParseID(distroNameVer)
		if err != nil {
			return nil, err
		}
		// XXX: we shoudl probably use a similar pattern like
		// for the partition table overrides (via
		// findElementIndexByJSONTag) but this if fine for now
		if distroNameCnf, ok := cond.DistroName[id.Name]; ok {
			imgConfig = distroNameCnf.InheritFrom(imgConfig)
		}
	}

	return imgConfig, nil
}

// imageTypesYAML describes the image types for a given distribution
// family. Note that multiple distros may use the same image types,
// e.g. centos/rhel
type imageTypesYAML struct {
	ImageConfig distroImageConfig        `yaml:"image_config,omitempty"`
	ImageTypes  map[string]ImageTypeYAML `yaml:"image_types"`
	Common      map[string]any           `yaml:".common,omitempty"`
}

type distroImageConfig struct {
	Default   *distro.ImageConfig          `yaml:"default"`
	Condition *distroImageConfigConditions `yaml:"condition,omitempty"`
}

type distroImageConfigConditions struct {
	DistroName map[string]*distro.ImageConfig `yaml:"distro_name,omitempty"`
}

type ImageTypeYAML struct {
	// This maps "pkgsKey" to their package sets. The
	// map key here is a string that can either be:
	// - "os": packages for the os
	// - "installer": packages for the installer
	// - "container": extra package into an iot container
	//
	// - "blueprint": unused AFAICT
	// - "build": unused AFAICT
	// Note that this does not directly maps to pipeline names
	// but we should look into making it so.
	PackageSetsYAML map[string][]packageSet `yaml:"package_sets"`
	// archStr->partitionTable
	PartitionTables map[string]*disk.PartitionTable `yaml:"partition_table"`
	// override specific aspects of the partition table
	PartitionTablesOverrides *partitionTablesOverrides `yaml:"partition_tables_override"`

	ImageConfigYAML     imageConfig     `yaml:"image_config,omitempty"`
	InstallerConfigYAML installerConfig `yaml:"installer_config,omitempty"`

	Filename    string                      `yaml:"filename"`
	MimeType    string                      `yaml:"mime_type"`
	Compression string                      `yaml:"compression"`
	Environment environment.EnvironmentConf `yaml:"environment"`
	Bootable    bool                        `yaml:"bootable"`

	BootISO  bool   `yaml:"boot_iso"`
	ISOLabel string `yaml:"iso_label"`
	// XXX: or iso_variant?
	Variant string `yaml:"variant"`

	RPMOSTree bool `yaml:"rpm_ostree"`

	OSTree struct {
		Name   string `yaml:"name"`
		Remote string `yaml:"remote"`
	} `yaml:"ostree"`

	DefaultSize uint64 `yaml:"default_size"`
	// the image func name: disk,container,live-installer,...
	Image                  string            `yaml:"image_func"`
	BuildPipelines         []string          `yaml:"build_pipelines"`
	PayloadPipelines       []string          `yaml:"payload_pipelines"`
	Exports                []string          `yaml:"exports"`
	RequiredPartitionSizes map[string]uint64 `yaml:"required_partition_sizes"`

	Platforms []platform.PlatformConf `yaml:"platforms"`

	NameAliases []string `yaml:"name_aliases"`

	// name is set by the loader
	name string
}

func (it *ImageTypeYAML) Name() string {
	return it.name
}

type imageConfig struct {
	*distro.ImageConfig `yaml:",inline"`
	Condition           *conditionsImgConf `yaml:"condition,omitempty"`
}

type conditionsImgConf struct {
	Architecture    map[string]*distro.ImageConfig `yaml:"architecture,omitempty"`
	DistroName      map[string]*distro.ImageConfig `yaml:"distro_name,omitempty"`
	VersionLessThan map[string]*distro.ImageConfig `yaml:"version_less_than,omitempty"`
}

type installerConfig struct {
	*distro.InstallerConfig `yaml:",inline"`
	Condition               *conditionsInstallerConf `yaml:"condition,omitempty"`
}

type conditionsInstallerConf struct {
	Architecture    map[string]*distro.InstallerConfig `yaml:"architecture,omitempty"`
	DistroName      map[string]*distro.InstallerConfig `yaml:"distro_name,omitempty"`
	VersionLessThan map[string]*distro.InstallerConfig `yaml:"version_less_than,omitempty"`
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
	Condition *partitionTablesOverwriteCondition `yaml:"condition"`
}

type partitionTablesOverwriteCondition struct {
	DistroName            map[string]map[string]*disk.PartitionTable `yaml:"distro_name,omitempty"`
	VersionGreaterOrEqual map[string]map[string]*disk.PartitionTable `yaml:"version_greater_or_equal,omitempty"`
	VersionLessThan       map[string]map[string]*disk.PartitionTable `yaml:"version_less_than,omitempty"`
}

// XXX: use slices.Backward() once we move to go1.23
// hint: use "git blame" on this comment and just revert
// the commit that adds it and you will have the 1.23 version
func backward[Slice ~[]E, E any](s Slice) []E {
	out := make([]E, 0, len(s))
	for i := len(s) - 1; i >= 0; i-- {
		out = append(out, s[i])
	}
	return out
}

// XXX: use slices.SortedFunc() once we move to go1.23
// hint: use "git blame" on this comment and just revert
// the commit that adds it and you will have the 1.23 version
func versionLessThanSortedKeys[T any](m map[string]T) []string {
	versions := maps.Keys(m)
	slices.SortFunc(versions, func(a, b string) int {
		ver1 := version.Must(version.NewVersion(a))
		ver2 := version.Must(version.NewVersion(b))
		switch {
		case ver1 == ver2:
			return 0
		case ver2.LessThan(ver1):
			return -1
		default:
			return 1
		}
	})
	return versions
}

// PackageSets loads the PackageSets from the yaml source file
// discovered via the imagetype.
func (imgType *ImageTypeYAML) PackageSets(distroNameVer, archName string) (map[string]rpmmd.PackageSet, error) {
	id, err := distro.ParseID(distroNameVer)
	if err != nil {
		return nil, err
	}

	res := make(map[string]rpmmd.PackageSet)
	for key, pkgSets := range imgType.PackageSetsYAML {
		var rpmmdPkgSet rpmmd.PackageSet
		for _, pkgSet := range pkgSets {
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
				if distroNameSet, ok := pkgSet.Condition.DistroName[id.Name]; ok {
					rpmmdPkgSet = rpmmdPkgSet.Append(rpmmd.PackageSet{
						Include: distroNameSet.Include,
						Exclude: distroNameSet.Exclude,
					})
				}
				// note that we don't need to order here, as
				// packageSets are strictly additive the order
				// is irrelevant
				for ltVer, ltSet := range pkgSet.Condition.VersionLessThan {
					if common.VersionLessThan(id.VersionString(), ltVer) {
						rpmmdPkgSet = rpmmdPkgSet.Append(rpmmd.PackageSet{
							Include: ltSet.Include,
							Exclude: ltSet.Exclude,
						})
					}
				}

				for gteqVer, gteqSet := range pkgSet.Condition.VersionGreaterOrEqual {
					if common.VersionGreaterThanOrEqual(id.VersionString(), gteqVer) {
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
		res[key] = rpmmdPkgSet
	}

	return res, nil
}

// PartitionTable returns the partionTable for the given distro/imgType.
func (imgType *ImageTypeYAML) PartitionTable(distroNameVer, archName string) (*disk.PartitionTable, error) {
	if imgType.PartitionTables == nil {
		return nil, fmt.Errorf("%w: %q", ErrNoPartitionTableForImgType, distroNameVer)
	}
	pt, ok := imgType.PartitionTables[archName]
	if !ok {
		return nil, fmt.Errorf("%w (%q): %q", ErrNoPartitionTableForArch, distroNameVer, archName)
	}

	if imgType.PartitionTablesOverrides != nil {
		cond := imgType.PartitionTablesOverrides.Condition
		id, err := distro.ParseID(distroNameVer)
		if err != nil {
			return nil, err
		}

		for _, ltVer := range versionLessThanSortedKeys(cond.VersionLessThan) {
			ltOverrides := cond.VersionLessThan[ltVer]
			if common.VersionLessThan(id.VersionString(), ltVer) {
				if newPt, ok := ltOverrides[archName]; ok {
					pt = newPt
				}
			}
		}

		for _, gteqVer := range backward(versionLessThanSortedKeys(cond.VersionGreaterOrEqual)) {
			geOverrides := cond.VersionGreaterOrEqual[gteqVer]
			if common.VersionGreaterThanOrEqual(id.VersionString(), gteqVer) {
				if newPt, ok := geOverrides[archName]; ok {
					pt = newPt
				}
			}
		}

		if distroNameOverrides, ok := cond.DistroName[id.Name]; ok {
			if newPt, ok := distroNameOverrides[archName]; ok {
				pt = newPt
			}
		}
	}

	return pt, nil
}

// ImageConfig returns the image type specific ImageConfig
func (imgType *ImageTypeYAML) ImageConfig(distroNameVer, archName string) *distro.ImageConfig {
	imgConfig := imgType.ImageConfigYAML.ImageConfig
	cond := imgType.ImageConfigYAML.Condition
	if cond != nil {
		id := common.Must(distro.ParseID(distroNameVer))

		if distroNameCnf, ok := cond.DistroName[id.Name]; ok {
			imgConfig = distroNameCnf.InheritFrom(imgConfig)
		}
		if archCnf, ok := cond.Architecture[archName]; ok {
			imgConfig = archCnf.InheritFrom(imgConfig)
		}
		for _, ltVer := range versionLessThanSortedKeys(cond.VersionLessThan) {
			ltOverrides := cond.VersionLessThan[ltVer]
			if common.VersionLessThan(id.VersionString(), ltVer) {
				imgConfig = ltOverrides.InheritFrom(imgConfig)
			}
		}
	}

	return imgConfig
}

// nNonEmpty returns the number of non-empty maps in the given
// input
func nNonEmpty[K comparable, V any](maps ...map[K]V) int {
	var nonEmpty int
	for _, m := range maps {
		if len(m) > 0 {
			nonEmpty++
		}
	}
	return nonEmpty
}

// InstallerConfig returns the InstallerConfig for the given imgType
// Note that on conditions the InstallerConfig is fully replaced, do
// any merging in YAML
func (imgType *ImageTypeYAML) InstallerConfig(distroNameVer, archName string) (*distro.InstallerConfig, error) {
	installerConfig := imgType.InstallerConfigYAML.InstallerConfig
	cond := imgType.InstallerConfigYAML.Condition
	if cond != nil {
		if nNonEmpty(cond.DistroName, cond.Architecture, cond.VersionLessThan) > 1 {
			return nil, fmt.Errorf("only a single conditional allowed in installer config for %v", imgType.name)
		}

		id, err := distro.ParseID(distroNameVer)
		if err != nil {
			return nil, err
		}

		if distroNameCnf, ok := cond.DistroName[id.Name]; ok {
			installerConfig = distroNameCnf
		}
		if archCnf, ok := cond.Architecture[archName]; ok {
			installerConfig = archCnf
		}
		for _, ltVer := range versionLessThanSortedKeys(cond.VersionLessThan) {
			ltOverrides := cond.VersionLessThan[ltVer]
			if common.VersionLessThan(id.VersionString(), ltVer) {
				installerConfig = ltOverrides
			}
		}
	}

	return installerConfig, nil
}
