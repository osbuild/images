package definition

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"

	"github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// PackageSet is a struct that represent an input for one DNF depsolve.
type PackageSet struct {
	// Include is a list of packages that must be in the image.
	Include []string `yaml:"include"`
	// Exclude is a list of packages that must not be in the image.
	Exclude []string `yaml:"exclude,omitempty"`
}

// Definition is a struct that represents a single image definition.
type Definition struct {
	// Packages is a map of package sets installed inside the image.
	// The accepted keys are currently defined by image types.
	Packages map[string]PackageSet `yaml:"packages"`
}

// File is a struct that represents a single yaml file with an image definition.
type File struct {
	// From is a list of paths to other files that should be included in the final definition.
	// These includes are processed in a DFS post-order.
	From []string `yaml:"from,omitempty"`

	// Def is the actual definition of the image.
	Def Definition `yaml:"def"`
}

func fileExists(dir fs.FS, filepath string) bool {
	_, err := fs.Stat(dir, filepath)
	return err == nil
}

func imageTypeFilePath(distroFamily, distroVersion, arch, imageType string) string {
	return path.Join(distroFamily, distroVersion, arch, imageType+".yaml")
}

// FindBestDefinitionFile finds the best match for the given distroId-distroVersion-arch-imageType combination
// in the given directory, and returns its path relative to the directory.
//
// Example of the algorithm:
// When user wants to build an ami for rhel 10.2 on x86_64, the library searches for the most suitable definition:

// - firstly, it tries to open `rhel/10.2/x86_64/ami.yaml`
// - if it doesn't exist, it tries to open `rhel/10.2/generic/ami.yaml`
// - if it doesn't exist, it tries to open `rhel/X.Y/x86_64/ami.yaml` or `rhel/X.Y/generic/ami.yaml`, with X.Y being the closest older version to 10.2 (10.1 will be prefered over 10.0)
// - if it doesn't exist, an error is returned
func FindBestDefinitionFile(dir fs.ReadDirFS, distroId, distroVersionStr, arch, imageType string) (string, error) {
	distroVersion, err := version.NewVersion(distroVersionStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse requested distro version: %w", err)
	}

	entries, err := dir.ReadDir(distroId)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("distro %s doesn't have any definitions: %w", distroId, err)
		}
		return "", fmt.Errorf("cannot open distro %s: %w", distroId, err)
	}

	var availableVersions []version.Version
	for _, entry := range entries {
		if fileExists(dir, imageTypeFilePath(distroId, entry.Name(), arch, imageType)) || fileExists(dir, imageTypeFilePath(distroId, entry.Name(), "generic", imageType)) {
			version, err := version.NewVersion(entry.Name())
			if err != nil {
				return "", fmt.Errorf("found unparseable version (%s): %w", path.Join(distroId, entry.Name()), err)
			}
			availableVersions = append(availableVersions, *version)
		}
	}

	bestMatch := findBestVersionMatch(*distroVersion, availableVersions)
	if bestMatch == nil {
		return "", fmt.Errorf("no match found for %s, %s, %s, %s", distroId, distroVersion, arch, imageType)
	}

	logrus.Debugf("Best match for %s, %s, %s, %s is %s", distroId, distroVersion, arch, imageType, bestMatch.Original())

	if fileExists(dir, imageTypeFilePath(distroId, bestMatch.Original(), arch, imageType)) {
		return imageTypeFilePath(distroId, bestMatch.Original(), arch, imageType), nil
	}

	return imageTypeFilePath(distroId, bestMatch.Original(), "generic", imageType), nil
}

// findBestVersionMatch finds the latest version that is less than or equal to the target version.
func findBestVersionMatch(targetVersion version.Version, distroVersions []version.Version) *version.Version {
	var filtered []version.Version
	for _, distro := range distroVersions {
		if distro.LessThanOrEqual(&targetVersion) {
			filtered = append(filtered, distro)
		}
	}

	slices.SortFunc(filtered, func(i, j version.Version) int {
		return i.Compare(&j)
	})

	if len(filtered) == 0 {
		return nil
	}

	return &filtered[len(filtered)-1]
}

// MergeConfig processes the given file and all its includes (the top-level from key) and merges them into a single config.
// The returned File always has an empty From field.
//
// Files are processed in a DFS post-order. One file cannot be included multiple times.
//
// The merge rules for fields under the Definition struct are as follows:
// - Packages: If a package set is defined in multiple files, the include and exclude lists are concatenated.
func MergeConfig(dir fs.FS, filepath string) (*File, error) {
	ct := configTraverser{}
	configs, err := ct.traverse(dir, filepath)
	if err != nil {
		return nil, err
	}

	merged, err := mergeDefinitions(configs)
	if err != nil {
		return nil, fmt.Errorf("failed to merge configs: %w", err)
	}

	return &File{
		Def: *merged,
	}, nil
}

type configTraverser struct {
	seen map[string]bool
}

// traverse processes the given file and all its includes (the top-level from key) and returns a list of definitions
// in DFS post-order.
func (c *configTraverser) traverse(dir fs.FS, filepath string) ([]Definition, error) {
	if c.seen == nil {
		c.seen = make(map[string]bool)
	}

	filepath = path.Clean(filepath)
	if c.seen[filepath] {
		return nil, fmt.Errorf("%s is included multiple times", filepath)
	}
	c.seen[filepath] = true

	file, err := dir.Open(filepath)
	if err != nil {
		return nil, err
	}

	defer func() {
		if file != nil {
			file.Close()
		}
	}()

	yamlDecoder := yaml.NewDecoder(file)
	yamlDecoder.KnownFields(true)

	var f File
	err = yamlDecoder.Decode(&f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode file %s: %w", filepath, err)
	}

	// close the file as soon as possible to avoid having too many open files
	file.Close()
	file = nil

	var allDefs []Definition

	for _, include := range f.From {
		newPath := path.Join(path.Dir(filepath), include)
		defs, err := c.traverse(dir, newPath)
		if err != nil {
			return nil, fmt.Errorf("file included from %s:\n%w", filepath, err)
		}
		allDefs = append(allDefs, defs...)
	}

	allDefs = append(allDefs, f.Def)

	return allDefs, nil
}

func mergeDefinitions(defs []Definition) (*Definition, error) {
	if len(defs) == 0 {
		return nil, fmt.Errorf("no definitions to merge")
	}

	merged := defs[0]

	for _, config := range defs[1:] {
		mergeTwoConfigs(&merged, &config)
	}

	return &merged, nil
}

func mergeTwoConfigs(a, b *Definition) {
	// merge packages
	for setName := range b.Packages {

		// a doesn't have the set, just copy it
		if _, ok := a.Packages[setName]; !ok {
			if a.Packages == nil {
				a.Packages = make(map[string]PackageSet)
			}
			a.Packages[setName] = b.Packages[setName]
			continue
		}

		set := a.Packages[setName]
		set.Include = append(set.Include, b.Packages[setName].Include...)
		set.Exclude = append(set.Exclude, b.Packages[setName].Exclude...)
		a.Packages[setName] = set
	}
}
