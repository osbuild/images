package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/osbuild/images/pkg/distrofactory"
	"github.com/osbuild/images/pkg/reporegistry"
)

func main() {
	definitions := map[string]map[string][]string{}
	distroFac := distrofactory.NewDefault()

	testedRepoRegistry, err := reporegistry.NewTestedDefault()
	if err != nil {
		panic(fmt.Sprintf("failed to create repo registry with tested distros: %v", err))
	}

	for _, distroName := range testedRepoRegistry.ListDistros() {
		distro := distroFac.GetDistro(distroName)
		if distro == nil {
			fmt.Fprintf(os.Stderr, "WARNING: invalid distro name %q", distroName)
			continue
		}

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
	err = encoder.Encode(definitions)
	if err != nil {
		panic(err)
	}
}
