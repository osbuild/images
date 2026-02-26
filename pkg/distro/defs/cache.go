package defs

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sync"

	"github.com/brunoga/deep"
	"github.com/osbuild/images/data/distrodefs"
	"github.com/osbuild/images/pkg/experimentalflags"
	"github.com/osbuild/images/pkg/olog"
	"go.yaml.in/yaml/v3"
)

// this can be overridden in tests
var defaultDataFS fs.FS = distrodefs.Data

func dataFS() fs.FS {
	// XXX: this is a short term measure, pass a set of
	// searchPaths down the stack instead
	dataFS := defaultDataFS
	if overrideDir := experimentalflags.String("yamldir"); overrideDir != "" {
		olog.Printf("WARNING: using experimental override dir %q", overrideDir)
		dataFS = os.DirFS(overrideDir)
	}
	return dataFS
}

func globFiles(pattern string) ([]string, error) {
	return fs.Glob(dataFS(), pattern)
}

// yamlCacheKey identifies a cached decode by path and value type.
// The same path can be decoded into different types (e.g. distrosYAML vs imageTypesYAML).
type yamlCacheKey struct {
	path string
	typ  string
}

var yamlCache struct {
	mu         sync.Mutex
	prototypes map[fs.FS]map[yamlCacheKey]any
}

// cachedDecodeYAML decodes the YAML file at the given path into the given value.
// The file is decoded only once and the result is cached. The cache is unique
// for each filesystem (it can be mocked in tests) and for each value type.
// Each returned value is a deep copy of the cached prototype.
func cachedDecodeYAML[T any](path string, v *T) error {
	yamlCache.mu.Lock()
	defer yamlCache.mu.Unlock()

	dfs := dataFS()
	if yamlCache.prototypes == nil {
		yamlCache.prototypes = make(map[fs.FS]map[yamlCacheKey]any)
	}
	if yamlCache.prototypes[dfs] == nil {
		yamlCache.prototypes[dfs] = make(map[yamlCacheKey]any)
	}

	var tmp T
	path = filepath.Clean(path)
	key := yamlCacheKey{path: path, typ: reflect.TypeOf(tmp).String()}
	prototype, ok := yamlCache.prototypes[dfs][key]
	if ok {
		tmp, ok = prototype.(T)
		if !ok {
			return fmt.Errorf("cachedDecodeYAML: prototype for %s is not of type %T", path, v)
		}

		*v = deep.MustCopy(tmp)
		return nil
	}

	f, err := dfs.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	decoder.KnownFields(true)
	if err := decoder.Decode(&tmp); err != nil {
		return err
	}
	yamlCache.prototypes[dfs][key] = tmp

	*v = deep.MustCopy(tmp)
	return nil
}
