package manifest_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/stretchr/testify/assert"
)

func writeEntrypoint(data map[string]interface{}, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create entrypoint file %q: %w", path, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			panic(fmt.Sprintf("error closing entrypoint file %q: %s", path, err))
		}
	}()

	if err := yaml.NewEncoder(file).Encode(data); err != nil {
		return fmt.Errorf("failed to write entrypoint file %q: %w", path, err)
	}
	return nil
}

func deserializeManifest(data []byte) map[string]interface{} {
	ds := map[string]interface{}{}

	if err := json.NewDecoder(bytes.NewReader(data)).Decode(&ds); err != nil {
		panic(fmt.Sprintf("error deserializing manifest for test: %s", err))
	}

	return ds
}

func TestOTKSerialize(t *testing.T) {
	type testCase struct {
		otkEntrypoint map[string]interface{} // entrypoint contents
		bp            blueprint.Blueprint
		manifest      map[string]interface{} // expected manifest (deserialized into map)
		err           error                  // expected error (nil if happy)
	}

	testCases := map[string]testCase{
		"happy-empty": {
			otkEntrypoint: map[string]interface{}{
				"otk.version": "1",
				"otk.target.osbuild": map[string]interface{}{
					"pipelines": nil,
				},
			},
			manifest: map[string]interface{}{
				"version":   "2",
				"pipelines": nil,
			},
		},

		"happy-customized": {
			otkEntrypoint: map[string]interface{}{
				"otk.version": "1",
				"otk.target.osbuild": map[string]interface{}{
					"pipelines": []interface{}{
						map[string]interface{}{
							// using the kernel append customization to test
							// that variables are resolved; the location of the
							// variable is not important
							"name": "${kernel_append}",
						},
					},
				},
			},
			bp: blueprint.Blueprint{
				Customizations: &blueprint.Customizations{
					Kernel: &blueprint.KernelCustomization{
						Append: "customization-value",
					},
				},
			},
			manifest: map[string]interface{}{
				"version": "2",
				"pipelines": []interface{}{
					map[string]interface{}{
						"name": "customization-value",
					},
				},
			},
		},

		"happy-empty-var": {
			otkEntrypoint: map[string]interface{}{
				"otk.version": "1",
				"otk.target.osbuild": map[string]interface{}{
					"pipelines": []interface{}{
						map[string]interface{}{
							"name": "${kernel_append}",
						},
					},
				},
			},
			manifest: map[string]interface{}{
				"version": "2",
				"pipelines": []interface{}{
					map[string]interface{}{
						"name": "",
					},
				},
			},
		},
	}

	tmpRoot := t.TempDir()

	for name := range testCases {
		tc := testCases[name]
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			entrypointPath := filepath.Join(tmpRoot, name+".yaml")
			assert.NoError(writeEntrypoint(tc.otkEntrypoint, entrypointPath))

			otkManifest := manifest.NewOTK(entrypointPath, tc.bp)
			serialized, err := otkManifest.Serialize(nil, nil, nil, nil)
			if tc.err == nil {
				// happy
				assert.NoError(err)
				assert.Equal(tc.manifest, deserializeManifest(serialized))
			} else {
				// error
				assert.Error(err)
				assert.Equal(tc.err, err)
			}
		})
	}
}
