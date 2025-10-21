// Standalone executable for building a test image.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/osbuild/blueprint/pkg/blueprint"
	"github.com/osbuild/images/internal/buildconfig"
	"github.com/osbuild/images/internal/cmdutil"
	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/distrofactory"
	"github.com/osbuild/images/pkg/manifestgen"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/reporegistry"
	"github.com/osbuild/images/pkg/rhsm/facts"
	"github.com/osbuild/images/pkg/rpmmd"
)

func u(s string) string {
	return strings.ReplaceAll(s, "-", "_")
}

func run() error {
	// common args
	var outputDir, osbuildStore, rpmCacheRoot, repositories string
	flag.StringVar(&outputDir, "output", ".", "artifact output directory")
	flag.StringVar(&osbuildStore, "store", ".osbuild", "osbuild store for intermediate pipeline trees")
	flag.StringVar(&rpmCacheRoot, "rpmmd", "/tmp/rpmmd", "rpm metadata cache directory")
	flag.StringVar(&repositories, "repositories", "test/data/repositories", "path to repository file or directory")

	// osbuild checkpoint arg
	var checkpoints cmdutil.MultiValue
	flag.Var(&checkpoints, "checkpoints", "comma-separated list of pipeline names to checkpoint (passed to osbuild --checkpoint)")

	// image selection args
	var distroName, imgTypeName, configFile string
	flag.StringVar(&distroName, "distro", "", "distribution (required)")
	flag.StringVar(&imgTypeName, "type", "", "image type name (required)")
	flag.StringVar(&configFile, "config", "", "build config file (required)")

	flag.Parse()

	if distroName == "" || imgTypeName == "" || configFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	distroFac := distrofactory.NewDefault()
	config, err := buildconfig.New(configFile, nil)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(outputDir, 0777); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	distribution := distroFac.GetDistro(distroName)
	if distribution == nil {
		return fmt.Errorf("invalid or unsupported distribution: %q", distroName)
	}

	archName := arch.Current().String()
	arch, err := distribution.GetArch(archName)
	if err != nil {
		return fmt.Errorf("invalid arch name %q for distro %q: %w", archName, distroName, err)
	}

	buildName := fmt.Sprintf("%s-%s-%s-%s", u(distroName), u(archName), u(imgTypeName), u(config.Name))
	buildDir := filepath.Join(outputDir, buildName)
	if err := os.MkdirAll(buildDir, 0777); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	imgType, err := arch.GetImageType(imgTypeName)
	if err != nil {
		return fmt.Errorf("invalid image type %q for distro %q and arch %q: %w", imgTypeName, distroName, archName, err)
	}

	var reporeg *reporegistry.RepoRegistry
	var overrideRepos []rpmmd.RepoConfig
	if st, err := os.Stat(repositories); err == nil && !st.IsDir() {
		// anything that is not a dir is tried to be loaded as a file
		// to allow "-repositories <arbitrarily-named-file>.json"
		repoConfig, err := rpmmd.LoadRepositoriesFromFile(repositories)
		if err != nil {
			return fmt.Errorf("failed to load repositories from %q: %w", repositories, err)
		}
		overrideRepos = repoConfig[archName]
	} else {
		reporeg, err = reporegistry.New([]string{repositories}, nil)
		if err != nil {
			return fmt.Errorf("failed to load repositories from %q: %w", repositories, err)
		}
	}
	seedArg, err := cmdutil.SeedArgFor(config, imgType.Name(), distribution.Name(), archName)
	if err != nil {
		return err
	}

	fmt.Printf("Generating manifest for %s: ", config.Name)
	manifestOpts := manifestgen.Options{
		Cachedir:       filepath.Join(rpmCacheRoot, archName+distribution.Name()),
		WarningsOutput: os.Stderr,
		OverrideRepos:  overrideRepos,
		CustomSeed:     &seedArg,
	}
	// add RHSM fact to detect changes
	config.Options.Facts = &facts.ImageOptions{
		APIType: facts.TEST_APITYPE,
	}
	if config.Blueprint == nil {
		config.Blueprint = &blueprint.Blueprint{}
	}

	mg, err := manifestgen.New(reporeg, &manifestOpts)
	if err != nil {
		return fmt.Errorf("[ERROR] manifest generator creation failed: %w", err)
	}
	mf, err := mg.Generate(config.Blueprint, imgType, &config.Options)
	if err != nil {
		return fmt.Errorf("[ERROR] manifest generation failed: %w", err)
	}
	fmt.Print("DONE\n")

	manifestPath := filepath.Join(buildDir, "manifest.json")
	// nolint:gosec
	if err := os.WriteFile(manifestPath, mf, 0644); err != nil {
		return fmt.Errorf("failed to write output file %q: %w", manifestPath, err)
	}

	fmt.Printf("Building manifest: %s\n", manifestPath)

	jobOutput := filepath.Join(outputDir, buildName)
	_, err = osbuild.RunOSBuild(mf, imgType.Exports(), checkpoints, os.Stderr, &osbuild.OSBuildOptions{
		StoreDir:   osbuildStore,
		OutputDir:  jobOutput,
		JSONOutput: false,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Jobs done. Results saved in\n%s\n", outputDir)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
