package otkdistro_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/osbuild/images/pkg/otkdistro"
	"github.com/stretchr/testify/require"
)

func makeFakeDistro(root string) ([]string, error) {
	arches := []string{
		"aarch64",
		"fakearch",
		"x86_64",
	}

	imageTypes := []string{
		"qcow2",
		"container",
	}

	props := map[string]interface{}{
		"name":            "FakeDistro",
		"os_version":      "42.0",
		"release_version": "42",
		"runner":          "org.osbuild.fakedistro42",
	}
	propsfile, err := os.Create(filepath.Join(root, "properties.yaml"))
	if err != nil {
		return nil, fmt.Errorf("error creating properties file for test distro: %w", err)
	}
	if err := yaml.NewEncoder(propsfile).Encode(props); err != nil {
		return nil, fmt.Errorf("error writing properties file for test distro: %w", err)
	}

	// make all otk files have the same content for now
	otkcontent := map[string]interface{}{
		"otk.version": "1",
		"otk.target.osbuild.name": map[string]interface{}{
			"pipelines": []map[string]interface{}{
				{
					"name": "build",
					"stages": []map[string]interface{}{
						{
							"type":    "org.osbuild.rpm",
							"options": nil,
						},
					},
				},
				{
					"name": "os",
					"stages": []map[string]interface{}{
						{
							"type":    "org.osbuild.rpm",
							"options": nil,
						},
					},
				},
			},
		},
	}

	expected := make([]string, 0, len(arches)*len(imageTypes))
	for _, arch := range arches {
		archpath := filepath.Join(root, arch)
		if err := os.Mkdir(archpath, 0o777); err != nil {
			return nil, fmt.Errorf("error creating architecture directory for test distro %q: %w", archpath, err)
		}
		for _, imageType := range imageTypes {
			itpath := filepath.Join(archpath, imageType) + ".yaml"
			itfile, err := os.Create(itpath)
			if err != nil {
				return nil, fmt.Errorf("error creating image type file for test distro %q: %w", itpath, err)
			}
			if err := yaml.NewEncoder(itfile).Encode(otkcontent); err != nil {
				return nil, fmt.Errorf("error writing image type file for test distro %q: %w", itpath, err)
			}
			expected = append(expected, fmt.Sprintf("%s/%s", arch, imageType))
		}
	}

	return expected, nil
}

func TestDistroLoad(t *testing.T) {
	require := require.New(t)

	distroRoot := t.TempDir()
	expected, err := makeFakeDistro(distroRoot)
	require.NoError(err)

	distro, err := otkdistro.New(distroRoot)
	require.NoError(err)
	require.Equal("FakeDistro", distro.Name())
	require.Equal("42.0", distro.OsVersion())
	require.Equal("42", distro.Releasever())
	// TODO: check runner when it's added to the distro interface

	archImageTypes := make([]string, 0)
	for _, archName := range distro.ListArches() {
		arch, err := distro.GetArch(archName)
		require.NoError(err)

		for _, imageTypeName := range arch.ListImageTypes() {
			archImageTypes = append(archImageTypes, fmt.Sprintf("%s/%s", archName, imageTypeName))
		}
	}

	require.ElementsMatch(expected, archImageTypes)
}
