// Standalone executable for building a test image.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
)

func fail(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

func check(err error) {
	if err != nil {
		fail(err.Error())
	}
}

type BuildConfig struct {
	Name      string               `json:"name"`
	OSTree    *ostree.ImageOptions `json:"ostree,omitempty"`
	Blueprint *blueprint.Blueprint `json:"blueprint,omitempty"`
	Depends   interface{}          `json:"depends,omitempty"` // ignored
}

func loadConfig(path string) BuildConfig {
	fp, err := os.Open(path)
	check(err)
	defer fp.Close()

	dec := json.NewDecoder(fp)
	dec.DisallowUnknownFields()
	var conf BuildConfig

	check(dec.Decode(&conf))
	if dec.More() {
		fail(fmt.Sprintf("multiple configuration objects or extra data found in %q", path))
	}
	return conf
}

func makeManifest(imgType distro.ImageType, config BuildConfig, distribution distro.Distro, repos []rpmmd.RepoConfig, archName string, seedArg int64, cacheRoot string) (manifest.OSBuildManifest, error) {
	cacheDir := filepath.Join(cacheRoot, archName+distribution.Name())

	options := distro.ImageOptions{Size: 0}
	options.OSTree = config.OSTree

	// add RHSM fact to detect changes
	options.Facts = &facts.ImageOptions{
		APIType: facts.TEST_APITYPE,
	}

	var bp blueprint.Blueprint
	if config.Blueprint != nil {
		bp = blueprint.Blueprint(*config.Blueprint)
	}

	manifest, warnings, err := imgType.Manifest(&bp, options, repos, seedArg)
	if err != nil {
		return nil, fmt.Errorf("[ERROR] manifest generation failed: %s", err.Error())
	}
	if len(warnings) > 0 {
		fmt.Fprintf(os.Stderr, "[WARNING]\n%s", strings.Join(warnings, "\n"))
	}

	packageSpecs, repoConfigs, err := depsolve(cacheDir, manifest.GetPackageSetChains(), distribution, archName)
	if err != nil {
		return nil, fmt.Errorf("[ERROR] depsolve failed: %s", err.Error())
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
		return nil, fmt.Errorf("[ERROR] container resolution failed: %s", err.Error())
	}

	commitSpecs, err := resolvePipelineCommits(manifest.GetOSTreeSourceSpecs())
	if err != nil {
		return nil, fmt.Errorf("[ERROR] ostree commit resolution failed: %s\n", err.Error())
	}

	mf, err := manifest.Serialize(packageSpecs, containerSpecs, commitSpecs)
	if err != nil {
		return nil, fmt.Errorf("[ERROR] manifest serialization failed: %s", err.Error())
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
		pkgs, repos, err := solver.Depsolve(pkgSet)
		if err != nil {
			return nil, nil, err
		}
		depsolvedSets[name] = pkgs
		repoSets[name] = repos
	}
	return depsolvedSets, repoSets, nil
}

func save(ms manifest.OSBuildManifest, fpath string) error {
	b, err := json.MarshalIndent(ms, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data for %q: %s\n", fpath, err.Error())
	}
	b = append(b, '\n') // add new line at end of file
	fp, err := os.Create(fpath)
	if err != nil {
		return fmt.Errorf("failed to create output file %q: %s\n", fpath, err.Error())
	}
	defer fp.Close()
	if _, err := fp.Write(b); err != nil {
		return fmt.Errorf("failed to write output file %q: %s\n", fpath, err.Error())
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

func main() {
	// common args
	var outputDir, osbuildStore, rpmCacheRoot string
	flag.StringVar(&outputDir, "output", ".", "artifact output directory")
	flag.StringVar(&osbuildStore, "store", ".osbuild", "osbuild store for intermediate pipeline trees")
	flag.StringVar(&rpmCacheRoot, "rpmmd", "/tmp/rpmmd", "rpm metadata cache directory")

	// image selection args
	var distroName, imgTypeName, configFile string
	flag.StringVar(&distroName, "distro", "", "distribution (required)")
	flag.StringVar(&imgTypeName, "image", "", "image type name (required)")
	flag.StringVar(&configFile, "config", "", "build config file (required)")

	flag.Parse()

	if distroName == "" || imgTypeName == "" || configFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	rngSeed, err := cmdutil.NewRNGSeed()
	check(err)

	testedRepoRegistry, err := reporegistry.NewTestedDefault()
	if err != nil {
		panic(fmt.Sprintf("failed to create repo registry with tested distros: %v", err))
	}
	distroFac := distrofactory.NewDefault()

	config := loadConfig(configFile)

	if err := os.MkdirAll(outputDir, 0777); err != nil {
		fail(fmt.Sprintf("failed to create target directory: %s", err.Error()))
	}

	distribution := distroFac.GetDistro(distroName)
	if distribution == nil {
		fail(fmt.Sprintf("invalid or unsupported distribution: %q", distroName))
	}

	archName := arch.Current().String()
	arch, err := distribution.GetArch(archName)
	if err != nil {
		fail(fmt.Sprintf("invalid arch name %q for distro %q: %s\n", archName, distroName, err.Error()))
	}

	buildName := fmt.Sprintf("%s-%s-%s-%s", u(distroName), u(archName), u(imgTypeName), u(config.Name))
	buildDir := filepath.Join(outputDir, buildName)
	if err := os.MkdirAll(buildDir, 0777); err != nil {
		fail(fmt.Sprintf("failed to create target directory: %s", err.Error()))
	}

	imgType, err := arch.GetImageType(imgTypeName)
	if err != nil {
		fail(fmt.Sprintf("invalid image type %q for distro %q and arch %q: %s\n", imgTypeName, distroName, archName, err.Error()))
	}

	// get repositories
	repos, err := testedRepoRegistry.ReposByArchName(distroName, archName, true)
	if err != nil {
		panic(fmt.Sprintf("failed to get repositories for %s/%s: %v", distroName, archName, err))
	}
	repos = filterRepos(repos, imgTypeName)
	if len(repos) == 0 {
		fail(fmt.Sprintf("no repositories defined for %s/%s\n", distroName, archName))
	}

	fmt.Printf("Generating manifest for %s: ", config.Name)
	mf, err := makeManifest(imgType, config, distribution, repos, archName, rngSeed, rpmCacheRoot)
	check(err)
	fmt.Print("DONE\n")

	manifestPath := filepath.Join(buildDir, "manifest.json")
	check(save(mf, manifestPath))

	fmt.Printf("Building manifest: %s\n", manifestPath)

	jobOutput := filepath.Join(outputDir, buildName)
	_, err = osbuild.RunOSBuild(mf, osbuildStore, jobOutput, imgType.Exports(), nil, nil, false, os.Stderr)
	check(err)

	fmt.Printf("Jobs done. Results saved in\n%s\n", outputDir)
}
