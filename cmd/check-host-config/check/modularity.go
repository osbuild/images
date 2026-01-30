package check

import (
	"log"
	"strings"

	"github.com/osbuild/images/internal/buildconfig"
)

func init() {
	RegisterCheck(Metadata{
		Name:                   "modularity",
		RequiresBlueprint: true,
		TempDisabled:      "https://github.com/osbuild/images/issues/2061",
	}, modularityCheck)
}

func modularityCheck(meta *Metadata, config *buildconfig.BuildConfig) error {
	// Verify modules that are enabled on a system, if any. Modules can either be enabled separately
	// or they can be installed through packages directly. We test both cases here.
	//
	// Caveat is that when a module is enabled yet _no_ packages are installed from it this breaks.
	// Let's not do that in the test?

	// Collect expected modules from enabled_modules and packages
	var expectedModules []string

	// From enabled_modules
	for _, mod := range config.Blueprint.EnabledModules {
		expectedModules = append(expectedModules, mod.Name+":"+mod.Stream)
	}

	// From packages that start with @ and contain :
	for _, pkg := range config.Blueprint.Packages {
		if strings.HasPrefix(pkg.Name, "@") && strings.Contains(pkg.Name, ":") {
			// Remove @ prefix
			moduleName := strings.TrimPrefix(pkg.Name, "@")
			expectedModules = append(expectedModules, moduleName)
		}
	}

	if len(expectedModules) == 0 {
		return Skip("no modules to check")
	}

	log.Println("Checking enabled modules")

	// Get list of enabled modules from dnf (use -q to suppress download progress output)
	stdout, _, _, err := Exec("dnf", "-q", "module", "list", "--enabled")
	if err != nil {
		return Fail("failed to list enabled modules:", err)
	}

	// Parse dnf output: skip first 3 lines (header) and last 2 lines (footer)
	lines := strings.Split(string(stdout), "\n")
	if len(lines) < 5 {
		return Fail("dnf module list returned nothing")
	}

	enabledModules := make(map[string]bool)
	for i := 3; i < len(lines)-2; i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		// Format: "name stream" or "name    stream"
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			moduleKey := fields[0] + ":" + fields[1]
			enabledModules[moduleKey] = true
		}
	}

	for _, expected := range expectedModules {
		if !enabledModules[expected] {
			return Fail("module was not enabled:", expected)
		}
		log.Printf("Expected module %q was enabled\n", expected)
	}

	return Pass()
}
