package spec

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type File struct {
	Includes []string       `yaml:"includes"`
	Spec     map[string]any `yaml:"spec"`
}

func fileExists(dir fs.FS, filepath string) bool {
	f, err := dir.Open(filepath)
	if err != nil {
		return false
	}
	f.Close()
	return true
}

func imageTypeFilePath(distro, arch, imageType string) string {
	return path.Join(distro, arch, imageType+".yaml")
}

func MergeConfigDistro(dir fs.ReadDirFS, distro, arch, imageType string) ([]byte, error) {
	entries, err := dir.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("failed to read dir: %w", err)
	}
	var distros []string
	for _, entry := range entries {
		if fileExists(dir, imageTypeFilePath(entry.Name(), arch, imageType)) || fileExists(dir, imageTypeFilePath(entry.Name(), "generic", imageType)) {
			distros = append(distros, entry.Name())
		}
	}

	bestMatch, err := findBestMatch(distro, distros)
	if err != nil {
		return nil, fmt.Errorf("failed to find best match: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Best match for %s is %s\n", distro, bestMatch)

	if fileExists(dir, imageTypeFilePath(bestMatch, arch, imageType)) {
		return MergeConfig(dir, imageTypeFilePath(bestMatch, arch, imageType))
	}

	return MergeConfig(dir, imageTypeFilePath(bestMatch, "generic", imageType))
}

func MergeConfig(dir fs.FS, filepath string) ([]byte, error) {
	configs, err := traverseConfigs(dir, filepath)
	if err != nil {
		return nil, err
	}

	merged, err := mergeConfigs(configs)
	if err != nil {
		return nil, fmt.Errorf("failed to merge configs: %w", err)
	}

	mergedWithHeader := map[string]any{
		"spec": merged,
	}

	rawMerged, err := yaml.Marshal(mergedWithHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	return rawMerged, nil
}

func parseDistro(distro string) (string, int, int, error) {
	components := strings.Split(distro, "-")

	if len(components) < 2 {
		return "", 0, 0, fmt.Errorf("invalid distro format")
	}

	name := components[0]
	majorStr := components[1]
	minorStr := "0"

	if strings.Contains(majorStr, ".") {
		parts := strings.Split(majorStr, ".")
		if len(parts) != 2 {
			return "", 0, 0, fmt.Errorf("invalid version format")
		}
		majorStr = parts[0]
		minorStr = parts[1]
	}

	major, err := strconv.Atoi(majorStr)
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to parse major version: %w", err)
	}
	minor, err := strconv.Atoi(minorStr)
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to parse minor version: %w", err)
	}
	return name, major, minor, nil
}

func findBestMatch(target string, distros []string) (string, error) {
	var filtered []string
	name, major, minor, err := parseDistro(target)
	if err != nil {
		return "", fmt.Errorf("failed to parse target distro: %w", err)
	}

	for _, distro := range distros {
		distName, distMajor, distMinor, err := parseDistro(distro)
		if err != nil {
			return "", fmt.Errorf("failed to parse distro: %w", err)
		}

		if distName != name {
			continue
		}

		if distMajor > major {
			continue
		}

		if distMajor == major && distMinor > minor {
			continue
		}

		filtered = append(filtered, distro)
	}

	slices.SortFunc(filtered, func(i, j string) int {
		_, iMajor, iMinor, _ := parseDistro(i)
		_, jMajor, jMinor, _ := parseDistro(j)

		if iMajor == jMajor {
			return iMinor - jMinor
		}
		return iMajor - jMajor
	})

	return filtered[len(filtered)-1], nil
}

func traverseConfigs(dir fs.FS, filepath string) ([]map[string]any, error) {
	// TODO: Protect me from infinite loops
	file, err := dir.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filepath, err)
	}
	defer file.Close()

	yamlDecoder := yaml.NewDecoder(file)
	yamlDecoder.KnownFields(true)

	var f File
	err = yamlDecoder.Decode(&f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode file %s: %w", filepath, err)
	}

	var allConfigs []map[string]any

	for _, include := range f.Includes {
		newPath := path.Join(path.Dir(filepath), include)
		configs, err := traverseConfigs(dir, newPath)
		if err != nil {
			return nil, fmt.Errorf("failed to process included file from %s: %w", filepath, err)
		}
		allConfigs = append(allConfigs, configs...)
	}

	allConfigs = append(allConfigs, f.Spec)

	return allConfigs, nil
}

func mergeConfigs(configs []map[string]any) (map[string]any, error) {
	if len(configs) == 0 {
		return nil, fmt.Errorf("no configs to merge")
	}

	merged := configs[0]

	for _, config := range configs[1:] {
		err := mergeTwoConfigs(merged, config)
		if err != nil {
			return nil, fmt.Errorf("failed to merge configs: %w", err)
		}
	}

	return merged, nil
}

func mergeTwoConfigs(a, b map[string]any) error {
	for key, value := range b {
		if _, ok := a[key]; !ok {
			a[key] = value
			continue
		}

		if reflect.TypeOf(a[key]) != reflect.TypeOf(value) {
			return fmt.Errorf("type mismatch for key %s", key)
		}

		if reflect.TypeOf(value).Kind() == reflect.Slice || reflect.TypeOf(value).Kind() == reflect.Array {
			a[key] = append(a[key].([]any), value.([]any)...)
		} else {
			a[key] = value
		}
	}
	return nil
}
