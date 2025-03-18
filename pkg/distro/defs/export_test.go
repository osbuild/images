package defs

import (
	"os"
)

func MockDataFS(path string) (restore func()) {
	saved := DataFS
	DataFS = os.DirFS(path)
	return func() {
		DataFS = saved
	}
}
