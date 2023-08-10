package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/osbuild/images/internal/testdistrolist"
)

func main() {
	definitions := map[string]map[string][]string{}
	distroRegistry := testdistrolist.New()

	for _, distroName := range distroRegistry.ListTested() {
		distro := distroRegistry.GetDistro(distroName)
		for _, archName := range distro.ListArches() {
			arch, err := distro.GetArch(archName)
			if err != nil {
				panic(fmt.Sprintf("failed to get arch %q of distro %q listed in aches list", archName, distroName))
			}
			_, ok := definitions[distroName]
			if !ok {
				definitions[distroName] = map[string][]string{}
			}
			definitions[distroName][archName] = arch.ListImageTypes()
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	err := encoder.Encode(definitions)
	if err != nil {
		panic(err)
	}
}
