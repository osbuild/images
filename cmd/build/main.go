// Standalone executable for building a test image.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/osbuild/images/internal/buildconfig"
	"github.com/osbuild/images/internal/cmdutil"
	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distrofactory"
	"github.com/osbuild/images/pkg/dnfjson"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/ostree"
	"github.com/osbuild/images/pkg/reporegistry"
	"github.com/osbuild/images/pkg/rhsm/facts"
	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/osbuild/images/pkg/sbom"
)

func makeManifest(
	config *buildconfig.BuildConfig,
	imgType distro.ImageType,
	distribution distro.Distro,
	repos []rpmmd.RepoConfig,
	archName string,
	cacheRoot string,
) (manifest.OSBuildManifest, error) {
	cacheDir := filepath.Join(cacheRoot, archName+distribution.Name())

	options := config.Options

	// add RHSM fact to detect changes
	options.Facts = &facts.ImageOptions{
		APIType: facts.TEST_APITYPE,
	}

	var bp blueprint.Blueprint
	if config.Blueprint != nil {
		bp = blueprint.Blueprint(*config.Blueprint)
	}
	seedArg, err := cmdutil.SeedArgFor(config, imgType.Name(), distribution.Name(), archName)
	if err != nil {
		return nil, err
	}

	manifest, warnings, err := imgType.Manifest(&bp, options, repos, seedArg)
	if err != nil {
		return nil, fmt.Errorf("[ERROR] manifest generation failed: %w", err)
	}
	if len(warnings) > 0 {
		fmt.Fprintf(os.Stderr, "[WARNING]\n%s", strings.Join(warnings, "\n"))
	}

	packageSpecs, repoConfigs, err := depsolve(cacheDir, manifest.GetPackageSetChains(), distribution, archName)
	if err != nil {
		return nil, fmt.Errorf("[ERROR] depsolve failed: %w", err)
	}
	if packageSpecs == nil {
		return nil, fmt.Errorf("[ERROR] depsolve did not return any packages")
	}
	_ = repoConfigs

	if config.Blueprint != nil {
		bp = blueprint.Blueprint(*config.Blueprint)
	}

	containerSpecs, err := resolvePipelineContainers(manifest.GetContainerSourceSpecs(), archName)
	if err != nil {
		return nil, fmt.Errorf("[ERROR] container resolution failed: %w", err)
	}

	commitSpecs, err := resolvePipelineCommits(manifest.GetOSTreeSourceSpecs())
	if err != nil {
		return nil, fmt.Errorf("[ERROR] ostree commit resolution failed: %w", err)
	}

	mf, err := manifest.Serialize(packageSpecs, containerSpecs, commitSpecs, nil)
	if err != nil {
		return nil, fmt.Errorf("[ERROR] manifest serialization failed: %w", err)
	}

	return mf, nil
}

func resolveContainers(containers []container.SourceSpec, archName string) ([]container.Spec, error) {
	resolver := container.NewResolver(archName)

	for _, c := range containers {
		resolver.Add(c)
	}

	return resolver.Finish()
}

func resolvePipelineContainers(containerSources map[string][]container.SourceSpec, archName string) (map[string][]container.Spec, error) {
	containerSpecs := make(map[string][]container.Spec, len(containerSources))
	for plName, sourceSpecs := range containerSources {
		specs, err := resolveContainers(sourceSpecs, archName)
		if err != nil {
			return nil, err
		}
		containerSpecs[plName] = specs
	}
	return containerSpecs, nil
}

func resolvePipelineCommits(commitSources map[string][]ostree.SourceSpec) (map[string][]ostree.CommitSpec, error) {
	commits := make(map[string][]ostree.CommitSpec, len(commitSources))
	for name, commitSources := range commitSources {
		commitSpecs := make([]ostree.CommitSpec, len(commitSources))
		for idx, commitSource := range commitSources {
			var err error
			commitSource.MTLS = &ostree.MTLS{
				CA:         os.Getenv("OSBUILD_SOURCES_OSTREE_SSL_CA_CERT"),
				ClientCert: os.Getenv("OSBUILD_SOURCES_OSTREE_SSL_CLIENT_CERT"),
				ClientKey:  os.Getenv("OSBUILD_SOURCES_OSTREE_SSL_CLIENT_KEY"),
			}
			commitSource.Proxy = os.Getenv("OSBUILD_SOURCES_OSTREE_PROXY")
			commitSpecs[idx], err = ostree.Resolve(commitSource)
			if err != nil {
				return nil, err
			}
		}
		commits[name] = commitSpecs
	}
	return commits, nil
}

func depsolve(cacheDir string, packageSets map[string][]rpmmd.PackageSet, d distro.Distro, arch string) (map[string][]rpmmd.PackageSpec, map[string][]rpmmd.RepoConfig, error) {
	solver := dnfjson.NewSolver(d.ModulePlatformID(), d.Releasever(), arch, d.Name(), cacheDir)
	depsolvedSets := make(map[string][]rpmmd.PackageSpec)
	repoSets := make(map[string][]rpmmd.RepoConfig)
	for name, pkgSet := range packageSets {
		res, err := solver.Depsolve(pkgSet, sbom.StandardTypeNone)
		if err != nil {
			return nil, nil, err
		}
		depsolvedSets[name] = res.Packages
		repoSets[name] = res.Repos
	}
	return depsolvedSets, repoSets, nil
}

func save(ms manifest.OSBuildManifest, fpath string) error {
	b, err := json.MarshalIndent(ms, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data for %q: %w", fpath, err)
	}
	b = append(b, '\n') // add new line at end of file
	fp, err := os.Create(fpath)
	if err != nil {
		return fmt.Errorf("failed to create output file %q: %w", fpath, err)
	}
	defer fp.Close()
	if _, err := fp.Write(b); err != nil {
		return fmt.Errorf("failed to write output file %q: %w", fpath, err)
	}
	return nil
}

func u(s string) string {
	return strings.Replace(s, "-", "_", -1)
}

func filterRepos(repos []rpmmd.RepoConfig, typeName string) []rpmmd.RepoConfig {
	filtered := make([]rpmmd.RepoConfig, 0)
	for _, repo := range repos {
		if len(repo.ImageTypeTags) == 0 {
			filtered = append(filtered, repo)
		} else {
			for _, tt := range repo.ImageTypeTags {
				if tt == typeName {
					filtered = append(filtered, repo)
					break
				}
			}
		}
	}
	return filtered
}

func run() error {
	// common args
	var outputDir, osbuildStore, rpmCacheRoot string
	flag.StringVar(&outputDir, "output", ".", "artifact output directory")
	flag.StringVar(&osbuildStore, "store", ".osbuild", "osbuild store for intermediate pipeline trees")
	flag.StringVar(&rpmCacheRoot, "rpmmd", "/tmp/rpmmd", "rpm metadata cache directory")

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

	testedRepoRegistry, err := reporegistry.NewTestedDefault()
	if err != nil {
		return fmt.Errorf("failed to create repo registry with tested distros: %v", err)
	}
	distroFac := distrofactory.NewDefault()

	config, err := buildconfig.New(configFile)
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

	// get repositories
	repos, err := testedRepoRegistry.ReposByArchName(distroName, archName, true)
	if err != nil {
		return fmt.Errorf("failed to get repositories for %s/%s: %w", distroName, archName, err)
	}
	repos = filterRepos(repos, imgTypeName)
	if len(repos) == 0 {
		return fmt.Errorf("no repositories defined for %s/%s", distroName, archName)
	}

	fmt.Printf("Generating manifest for %s: ", config.Name)
	mf, err := makeManifest(config, imgType, distribution, repos, archName, rpmCacheRoot)
	if err != nil {
		return err
	}
	fmt.Print("DONE\n")

	manifestPath := filepath.Join(buildDir, "manifest.json")
	if err := save(mf, manifestPath); err != nil {
		return err
	}

	fmt.Printf("Building manifest: %s\n", manifestPath)

	jobOutput := filepath.Join(outputDir, buildName)
	_, err = osbuild.RunOSBuild(mf, osbuildStore, jobOutput, imgType.Exports(), checkpoints, nil, false, os.Stderr)
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
