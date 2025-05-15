package defs

import (
	"os"
)

func MockDataFS(path string) (restore func()) {
	saved := defaultDataFS
	defaultDataFS = os.DirFS(path)
	return func() {
		defaultDataFS = saved
	}
}
