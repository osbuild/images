package testdistrolist

import (
	"fmt"
	"os"
	"strings"

	"github.com/osbuild/images/pkg/distrolist"
)

type DistroList struct {
	distrolist.List
}

func New() DistroList {
	return DistroList{
		distrolist.NewDefault(),
	}
}

func (l *DistroList) ListTested() []string {
	files, err := os.ReadDir("test/data/repositories")
	if err != nil {
		panic(fmt.Sprintf("error when enumerating the test repositories: %v", err))
	}

	var distros []string
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		d := strings.TrimSuffix(f.Name(), ".json")
		distros = append(distros, d)
	}

	return distros
}
