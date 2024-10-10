package otk_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/otk"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func readFile(path string) map[string]interface{} {
	file, err := os.Open(path)
	if err != nil {
		panic(fmt.Sprintf("failed to open file %q: %s", path, err))
	}
	defer file.Close()

	ds := map[string]interface{}{}
	if err := yaml.NewDecoder(file).Decode(&ds); err != nil {
		panic(fmt.Sprintf("error deserializing customizations file for test: %s", err))
	}

	return ds

}

func TestNewCustomizationsFile(t *testing.T) {
	type testCase struct {
		bp         blueprint.Blueprint
		entrypoint string                 // entrypoint filename
		expected   map[string]interface{} // expected file contents (deserialized from yaml)
		err        error                  // expected error (nil if happy)
	}

	cwd, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("error getting working directory for test: %s", err))
	}

	testCases := map[string]testCase{
		"empty-and-happy-abspath": {
			bp:         blueprint.Blueprint{},
			entrypoint: "/tmp/test.yaml",
			expected: map[string]interface{}{
				"otk.define.customizations": map[string]interface{}{
					"hostname":      "",
					"kernel_append": "",
					"languages":     []interface{}{},
					"keyboard":      "",
				},
				"otk.include": "/tmp/test.yaml",
			},
		},
		"empty-and-happy-relpath": {
			bp:         blueprint.Blueprint{},
			entrypoint: "./rel/test.yaml",
			expected: map[string]interface{}{
				"otk.define.customizations": map[string]interface{}{
					"hostname":      "",
					"kernel_append": "",
					"languages":     []interface{}{},
					"keyboard":      "",
				},
				"otk.include": filepath.Join(cwd, "rel/test.yaml"),
			},
		},
		"full-and-happy": {
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Hostname: common.ToPtr("test-host"),
					Kernel: &blueprint.KernelCustomization{
						Append: "ro",
					},
					Locale: &blueprint.LocaleCustomization{
						Languages: []string{"de_DE.UTF-8", "el_CY.UTF-8"},
						Keyboard:  common.ToPtr("uk"),
					},
				},
				Distro:  "",
				Minimal: false,
			},
			entrypoint: "/etc/otk/test.yaml",
			expected: map[string]interface{}{
				"otk.define.customizations": map[string]interface{}{
					"hostname":      "test-host",
					"kernel_append": "ro",
					"languages":     []interface{}{"de_DE.UTF-8", "el_CY.UTF-8"},
					"keyboard":      "uk",
				},
				"otk.include": "/etc/otk/test.yaml",
			},
		},
	}

	for name := range testCases {
		tc := testCases[name]
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			cfile, err := otk.NewCustomizationsFile(tc.bp, tc.entrypoint)
			defer func() {
				err := cfile.Cleanup()
				if err != nil {
					panic(fmt.Sprintf("error cleaning up customization file %q: %s", cfile.Path(), err))
				}
			}()
			if tc.err == nil {
				// happy
				assert.NoError(err)
				assert.Equal(tc.expected, readFile(cfile.Path()))
			} else {
				// error
				assert.Error(err)
				assert.Equal(tc.err, err)
			}
		})
	}
}
