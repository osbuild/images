package defs

import (
	"os"
)

type WhenCondition = whenCondition

func MockDataFS(path string) (restore func()) {
	saved := defaultDataFS
	defaultDataFS = os.DirFS(path)
	return func() {
		defaultDataFS = saved
	}
}

func ResetYamlCache() {
	yamlCache.mu.Lock()
	defer yamlCache.mu.Unlock()

	yamlCache.prototypes = nil
}
