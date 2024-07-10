package otkdistro

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/rpmmd"
	"gopkg.in/yaml.v3"
)

type Distro struct {
	id      string
	arches  map[string]distro.Arch
	otkRoot string

	props DistroProperties
}

func New(otkRoot string) (distro.Distro, error) {
	// use path basename as distro id
	id := filepath.Base(otkRoot)

	d := &Distro{
		id:      id,
		otkRoot: otkRoot,
	}

	if err := d.loadProperties(); err != nil {
		return nil, fmt.Errorf("error loading otk distro at %q: %w", otkRoot, err)
	}
	if err := d.readArches(); err != nil {
		return nil, fmt.Errorf("error loading otk distro at %q: %w", otkRoot, err)
	}
	return d, nil
}

type DistroProperties struct {
	Name             string `yaml:"name"`
	Codename         string `yaml:"codename"`
	Product          string `yaml:"product"`
	OSVersion        string `yaml:"os_version"`
	ReleaseVersion   string `yaml:"release_version"`
	ModulePlatformID string `yaml:"module_platform_id"`
	UEFIVendor       string `yaml:"uefi_vendor"`
	OSTreeRefTmpl    string `yaml:"ostree_ref_template"` // TODO: figure out templating
	Runner           string `yaml:"runner"`              // TODO: expose through a method on the interface
}

// loadProperties reads the properties.yaml file in the distro's otkRoot and
// loads the information into the distro fields. Errors are returned if the
// file is missing or one of the required fields are not set.
func (d *Distro) loadProperties() error {
	propsPath := filepath.Join(d.otkRoot, "properties.yaml")
	propsFile, err := os.Open(propsPath)
	if err != nil {
		return fmt.Errorf("error opening props file %q: %w", propsPath, err)
	}

	var props DistroProperties
	decoder := yaml.NewDecoder(propsFile)
	decoder.KnownFields(true) // produce error if there are fields in the file that don't exist in the target struct
	if err := decoder.Decode(&props); err != nil {
		return fmt.Errorf("error reading props file %q: %w", propsPath, err)
	}

	// check that required properties aren't empty
	// TODO: use reflect?
	missingPropError := func(propName string) error {
		return fmt.Errorf("incomplete props file %q: %q is required", propsPath, propName)
	}
	if props.Name == "" {
		return missingPropError("name")
	}
	if props.OSVersion == "" {
		return missingPropError("os_version")
	}
	if props.ReleaseVersion == "" {
		return missingPropError("release_version")
	}
	if props.Runner == "" {
		return missingPropError("runner")
	}

	d.props = props

	return nil
}

func (d *Distro) readArches() error {
	// read otkRoot and discover arches based on expected directory structure
	entries, err := os.ReadDir(d.otkRoot)
	if err != nil {
		return fmt.Errorf("failed to read otk root for distro %q: %s", d.Name(), err)
	}

	d.arches = make(map[string]distro.Arch)
	for _, entry := range entries {
		if !entry.IsDir() {
			// ignore files
			continue
		}
		// assume each subdir is an architecture
		name := entry.Name()
		a := Arch{
			name:    name,
			otkRoot: filepath.Join(d.otkRoot, entry.Name()),
		}
		if err := a.readImageTypes(); err != nil {
			return fmt.Errorf("error reading architecture at %q for otk distro %q: %w", a.otkRoot, d.Name(), err)
		}
		d.arches[a.Name()] = a
	}
	return nil
}

func (d Distro) Name() string {
	return d.props.Name
}

func (d Distro) SymbolicName() string {
	return d.id
}

func (d Distro) Codename() string {
	return d.props.Codename
}

func (d Distro) Releasever() string {
	return d.props.ReleaseVersion
}

func (d Distro) OsVersion() string {
	return d.props.OSVersion
}

func (d Distro) ModulePlatformID() string {
	return d.props.ModulePlatformID
}

func (d Distro) Product() string {
	return d.props.Product
}

func (d Distro) OSTreeRef() string {
	return d.props.OSTreeRefTmpl
}

func (d Distro) ListArches() []string {
	arches := make([]string, 0, len(d.arches))
	for arch := range d.arches {
		arches = append(arches, arch)
	}
	return arches
}

func (d Distro) GetArch(name string) (distro.Arch, error) {
	return d.arches[name], nil
}

type Arch struct {
	distribution *Distro
	name         string
	imageTypes   map[string]distro.ImageType
	otkRoot      string
}

func (a *Arch) readImageTypes() error {
	entries, err := os.ReadDir(a.otkRoot)
	if err != nil {
		return fmt.Errorf("error reading architecture directory %q for otk test distro: %s", a.otkRoot, err)
	}

	a.imageTypes = make(map[string]distro.ImageType)
	for _, entry := range entries {
		if entry.IsDir() {
			// ignore subdirectories
			continue
		}

		switch filepath.Ext(entry.Name()) {
		case ".yml", ".yaml":
			imageType := newOtkImageType(filepath.Join(a.otkRoot, entry.Name()))
			a.imageTypes[imageType.Name()] = &imageType
		}
	}
	return nil
}

func (a Arch) Name() string {
	return a.name
}

func (a Arch) Distro() distro.Distro {
	return a.distribution
}

func (a Arch) ListImageTypes() []string {
	names := make([]string, 0, len(a.imageTypes))
	for name := range a.imageTypes {
		names = append(names, name)
	}
	return names
}

func (a Arch) GetImageType(name string) (distro.ImageType, error) {
	return a.imageTypes[name], nil
}

type otkImageType struct {
	architecture *Arch
	name         string
	otkPath      string
}

func newOtkImageType(path string) otkImageType {
	baseFilename := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	it := otkImageType{
		name:    baseFilename,
		otkPath: path,
	}
	return it
}

func (t *otkImageType) Name() string {
	return t.name
}

func (t *otkImageType) Arch() distro.Arch {
	return t.architecture
}

func (t *otkImageType) Filename() string {
	return ""
}

func (t *otkImageType) MIMEType() string {
	return ""
}

func (t *otkImageType) OSTreeRef() string {
	d := t.Arch().Distro()
	return fmt.Sprintf(d.OSTreeRef(), t.architecture.Name())
}

func (t *otkImageType) ISOLabel() (string, error) {
	return "", nil
}

func (t *otkImageType) Size(size uint64) uint64 {
	return size
}

func (t *otkImageType) BuildPipelines() []string {
	return nil
}

func (t *otkImageType) PayloadPipelines() []string {
	return nil
}

func (t *otkImageType) PayloadPackageSets() []string {
	return nil
}

func (t *otkImageType) PackageSetsChains() map[string][]string {
	return nil
}

func (t *otkImageType) Exports() []string {
	return nil
}

func (t *otkImageType) BootMode() distro.BootMode {
	return distro.BOOT_NONE
}

func (t *otkImageType) PartitionType() string {
	return ""
}

func (t *otkImageType) Manifest(bp *blueprint.Blueprint,
	options distro.ImageOptions,
	repos []rpmmd.RepoConfig,
	seed int64) (manifest.Manifest, []string, error) {

	mf := manifest.NewOTK(t.otkPath)
	return mf, nil, nil
}

func DistroFactory(idStr string) distro.Distro {
	otkRoot := "otk" // TODO: configurable path
	otkDistroRoot := filepath.Join(otkRoot, idStr)
	d, err := New(otkDistroRoot)
	if err != nil {
		if strings.HasPrefix(idStr, "otk") {
			// NOTE: printing error only if the distro name is otk otherwise we
			// get an error for every distro that's checked. When we move to
			// having only otk-based distros, this will be a fatal error.
			logrus.Errorf("failed to load otk distro at %q: %s", otkDistroRoot, err)
		}
		return nil
	}
	return d
}
