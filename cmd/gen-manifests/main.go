// Standalone executable for generating all test manifests in parallel.
// Collects list of image types from the distro list.  Must be run from the
// root of the repository and reads test/data/repositories for repositories
// test/config-map.json to match image types with configuration files.
// Collects errors and failures and prints them after all jobs are finished.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gobwas/glob"

	"github.com/osbuild/images/internal/buildconfig"
	"github.com/osbuild/images/internal/cmdutil"
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distrofactory"
	"github.com/osbuild/images/pkg/dnfjson"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/ostree"
	"github.com/osbuild/images/pkg/reporegistry"
	"github.com/osbuild/images/pkg/rhsm/facts"
	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/osbuild/images/pkg/sbom"
)

type buildRequest struct {
	Distro       string                   `json:"distro,omitempty"`
	Arch         string                   `json:"arch,omitempty"`
	ImageType    string                   `json:"image-type,omitempty"`
	Repositories []rpmmd.RepoConfig       `json:"repositories,omitempty"`
	Config       *buildconfig.BuildConfig `json:"config"`
}

type BuildDependency struct {
	Config    string `json:"config"`
	ImageType string `json:"image-type"`
}

// BuildConfigs is a nested map representing the configs to use for each
// distro/arch/image-type. If any component is empty, it maps to all values.
type BuildConfigs map[string]map[string]map[string][]*buildconfig.BuildConfig

func (bc BuildConfigs) Insert(distro, arch, imageType string, cfg *buildconfig.BuildConfig) {
	distroCfgs := bc[distro]
	if distroCfgs == nil {
		distroCfgs = make(map[string]map[string][]*buildconfig.BuildConfig)
	}

	distroArchCfgs := distroCfgs[arch]
	if distroArchCfgs == nil {
		distroArchCfgs = make(map[string][]*buildconfig.BuildConfig)
	}

	distroArchItCfgs := distroArchCfgs[imageType]
	if distroArchItCfgs == nil {
		distroArchItCfgs = make([]*buildconfig.BuildConfig, 0)
	}

	distroArchItCfgs = append(distroArchItCfgs, cfg)
	distroArchCfgs[imageType] = distroArchItCfgs
	distroCfgs[arch] = distroArchCfgs
	bc[distro] = distroCfgs
}

func (bc BuildConfigs) Get(distro, arch, imageType string) []*buildconfig.BuildConfig {
	configs := make([]*buildconfig.BuildConfig, 0)
	for distroName, distroCfgs := range bc {
		distroGlob := glob.MustCompile(distroName)
		if distroGlob.Match(distro) {
			for archName, distroArchCfgs := range distroCfgs {
				archGlob := glob.MustCompile(archName)
				if archGlob.Match(arch) {
					for itName, distroArchItCfgs := range distroArchCfgs {
						itGlob := glob.MustCompile(itName)
						if itGlob.Match(imageType) {
							configs = append(configs, distroArchItCfgs...)
						}
					}
				}
			}
		}
	}
	return configs
}

func loadConfigMap(configPath string) BuildConfigs {
	type configFilters struct {
		ImageTypes []string `json:"image-types"`
		Distros    []string `json:"distros"`
		Arches     []string `json:"arches"`
	}
	type configMap map[string]configFilters

	fp, err := os.Open(configPath)
	if err != nil {
		panic(fmt.Sprintf("failed to open config map %q: %s", configPath, err.Error()))
	}
	defer fp.Close()
	data, err := io.ReadAll(fp)
	if err != nil {
		panic(fmt.Sprintf("failed to read config map %q: %s", configPath, err.Error()))
	}
	var cfgMap configMap
	if err := json.Unmarshal(data, &cfgMap); err != nil {
		panic(fmt.Sprintf("failed to unmarshal config map %q: %s", configPath, err.Error()))
	}

	emptyFallback := func(list []string) []string {
		if len(list) == 0 {
			// empty list means everything so let's add the * to catch
			// everything with glob
			return []string{"*"}
		}
		return list
	}

	// load each config from its path
	cm := make(BuildConfigs)
	for path, filters := range cfgMap {
		// config paths can be relative to the location of the config map
		if !filepath.IsAbs(path) {
			cfgDir := filepath.Dir(configPath)
			path = filepath.Join(cfgDir, path)
		}
		config, err := buildconfig.New(path)
		if err != nil {
			panic(err)
		}
		for _, d := range emptyFallback(filters.Distros) {
			for _, a := range emptyFallback(filters.Arches) {
				for _, it := range emptyFallback(filters.ImageTypes) {
					cm.Insert(d, a, it, config)
				}
			}
		}
	}

	return cm
}

// loadImgConfig loads a single image config from a file and returns a
// a BuildConfigs map with the config mapped to all distros, arches, and
// image types.
func loadImgConfig(configPath string) BuildConfigs {
	cm := make(BuildConfigs)
	config, err := buildconfig.New(configPath)
	if err != nil {
		panic(err)
	}
	cm.Insert("*", "*", "*", config)
	return cm
}

type manifestJob func(chan string) error

func makeManifestJob(
	bc *buildconfig.BuildConfig,
	imgType distro.ImageType,
	distribution distro.Distro,
	repos []rpmmd.RepoConfig,
	archName string,
	cacheRoot string,
	path string,
	content map[string]bool,
	metadata bool,
) manifestJob {
	name := bc.Name
	distroName := distribution.Name()
	filename := fmt.Sprintf("%s-%s-%s-%s.json", u(distroName), u(archName), u(imgType.Name()), u(name))
	cacheDir := filepath.Join(cacheRoot, archName+distribution.Name())

	// ensure that each file has a unique seed based on filename
	seedArg, err := cmdutil.SeedArgFor(bc, imgType.Name(), distribution.Name(), archName)
	if err != nil {
		panic(err)
	}

	options := bc.Options

	var bp blueprint.Blueprint
	if bc.Blueprint != nil {
		bp = *bc.Blueprint
	}

	// add RHSM fact to detect changes
	options.Facts = &facts.ImageOptions{
		APIType: facts.TEST_APITYPE,
	}

	job := func(msgq chan string) (err error) {
		defer func() {
			msg := fmt.Sprintf("Finished job %s", filename)
			if err != nil {
				msg += " [failed]"
			}
			msgq <- msg
		}()
		msgq <- fmt.Sprintf("Starting job %s", filename)

		manifest, _, err := imgType.Manifest(&bp, options, repos, seedArg)
		if err != nil {
			err = fmt.Errorf("[%s] failed: %s", filename, err)
			return
		}

		var packageSpecs map[string][]rpmmd.PackageSpec
		var repoConfigs map[string][]rpmmd.RepoConfig
		if content["packages"] {
			packageSpecs, repoConfigs, err = depsolve(cacheDir, manifest.GetPackageSetChains(), distribution, archName)
			if err != nil {
				err = fmt.Errorf("[%s] depsolve failed: %s", filename, err.Error())
				return
			}
			if packageSpecs == nil {
				err = fmt.Errorf("[%s] nil package specs", filename)
				return
			}
		} else {
			packageSpecs, repoConfigs = mockDepsolve(manifest.GetPackageSetChains(), repos, archName)
		}
		_ = repoConfigs

		var containerSpecs map[string][]container.Spec
		if content["containers"] {
			containerSpecs, err = resolvePipelineContainers(manifest.GetContainerSourceSpecs(), archName)
			if err != nil {
				return fmt.Errorf("[%s] container resolution failed: %s", filename, err.Error())
			}
		} else {
			containerSpecs = mockResolveContainers(manifest.GetContainerSourceSpecs())
		}

		var commitSpecs map[string][]ostree.CommitSpec
		if content["commits"] {
			commitSpecs, err = resolvePipelineCommits(manifest.GetOSTreeSourceSpecs())
			if err != nil {
				return fmt.Errorf("[%s] ostree commit resolution failed: %s", filename, err.Error())
			}
		} else {
			commitSpecs = mockResolveCommits(manifest.GetOSTreeSourceSpecs())
		}

		mf, err := manifest.Serialize(packageSpecs, containerSpecs, commitSpecs, repoConfigs)
		if err != nil {
			return fmt.Errorf("[%s] manifest serialization failed: %s", filename, err.Error())
		}

		request := buildRequest{
			Distro:       distribution.Name(),
			Arch:         archName,
			ImageType:    imgType.Name(),
			Repositories: repos,
			Config:       bc,
		}
		err = save(mf, packageSpecs, containerSpecs, commitSpecs, request, path, filename, metadata)
		return
	}
	return job
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

func mockResolveContainers(containerSources map[string][]container.SourceSpec) map[string][]container.Spec {
	containerSpecs := make(map[string][]container.Spec, len(containerSources))
	for plName, sourceSpecs := range containerSources {
		specs := make([]container.Spec, len(sourceSpecs))
		for idx, src := range sourceSpecs {
			digest := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(src.Name+src.Source+"digest")))
			id := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(src.Name+src.Source+"imageid")))
			listDigest := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(src.Name+src.Source+"list-digest")))
			name := src.Name
			if name == "" {
				name = src.Source
			}
			spec := container.Spec{
				Source:     src.Source,
				Digest:     digest,
				TLSVerify:  src.TLSVerify,
				ImageID:    id,
				LocalName:  name,
				ListDigest: listDigest,
			}
			specs[idx] = spec
		}
		containerSpecs[plName] = specs
	}
	return containerSpecs
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

func mockResolveCommits(commitSources map[string][]ostree.SourceSpec) map[string][]ostree.CommitSpec {
	commits := make(map[string][]ostree.CommitSpec, len(commitSources))
	for name, commitSources := range commitSources {
		commitSpecs := make([]ostree.CommitSpec, len(commitSources))
		for idx, commitSource := range commitSources {
			commitSpecs[idx] = cmdutil.MockOSTreeResolve(commitSource)
		}
		commits[name] = commitSpecs
	}
	return commits
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

func mockDepsolve(packageSets map[string][]rpmmd.PackageSet, repos []rpmmd.RepoConfig, archName string) (map[string][]rpmmd.PackageSpec, map[string][]rpmmd.RepoConfig) {
	depsolvedSets := make(map[string][]rpmmd.PackageSpec)
	repoSets := make(map[string][]rpmmd.RepoConfig)

	for name, pkgSetChain := range packageSets {
		specSet := make([]rpmmd.PackageSpec, 0)
		for _, pkgSet := range pkgSetChain {
			for _, pkgName := range pkgSet.Include {
				checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(pkgName)))
				// generate predictable but non-empty
				// release/version numbers
				ver := strconv.Itoa(int(pkgName[0]) % 9)
				rel := strconv.Itoa(int(pkgName[1]) % 9)
				spec := rpmmd.PackageSpec{
					Name:           pkgName,
					Epoch:          0,
					Version:        ver,
					Release:        rel + ".fk1",
					Arch:           archName,
					RemoteLocation: fmt.Sprintf("https://example.com/repo/packages/%s", pkgName),
					Checksum:       "sha256:" + checksum,
				}
				specSet = append(specSet, spec)
			}
			for _, excludeName := range pkgSet.Exclude {
				pkgName := fmt.Sprintf("exclude:%s", excludeName)
				checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(pkgName)))
				spec := rpmmd.PackageSpec{
					Name:           pkgName,
					Epoch:          0,
					Version:        "0",
					Release:        "0",
					Arch:           "noarch",
					RemoteLocation: fmt.Sprintf("https://example.com/repo/packages/%s", pkgName),
					Checksum:       "sha256:" + checksum,
				}
				specSet = append(specSet, spec)
			}
		}

		// generate pseudo packages for the repos
		for _, repo := range repos {
			// the test repos have the form:
			//   https://rpmrepo..../el9/cs9-x86_64-rt-20240915
			// drop the date as it's not needed for this level of
			// mocks
			baseURL := repo.BaseURLs[0]
			if idx := strings.LastIndex(baseURL, "-"); idx > 0 {
				baseURL = baseURL[:idx]
			}
			url, err := url.Parse(baseURL)
			if err != nil {
				panic(err)
			}
			url.Host = "example.com"
			url.Path = fmt.Sprintf("passed-arch:%s/passed-repo:%s", archName, url.Path)
			specSet = append(specSet, rpmmd.PackageSpec{
				Name:           url.String(),
				RemoteLocation: url.String(),
				Checksum:       "sha256:" + fmt.Sprintf("%x", sha256.Sum256([]byte(url.String()))),
			})
		}

		depsolvedSets[name] = specSet
		repoSets[name] = repos
	}

	return depsolvedSets, repoSets
}

func save(ms manifest.OSBuildManifest, pkgs map[string][]rpmmd.PackageSpec, containers map[string][]container.Spec, commits map[string][]ostree.CommitSpec, cr buildRequest, path, filename string, metadata bool) error {
	var data interface{}
	if metadata {
		data = struct {
			BuidRequest   buildRequest                   `json:"build-request"`
			Manifest      manifest.OSBuildManifest       `json:"manifest"`
			RPMMD         map[string][]rpmmd.PackageSpec `json:"rpmmd"`
			Containers    map[string][]container.Spec    `json:"containers,omitempty"`
			OSTreeCommits map[string][]ostree.CommitSpec `json:"ostree-commits,omitempty"`
			NoImageInfo   bool                           `json:"no-image-info"`
		}{
			cr, ms, pkgs, containers, commits, true,
		}
	} else {
		data = ms
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data for %q: %s\n", filename, err.Error())
	}
	b = append(b, '\n') // add new line at end of file
	fpath := filepath.Join(path, filename)
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

func u(s string) string {
	return strings.Replace(s, "-", "_", -1)
}

func main() {
	// common args
	var outputDir, cacheRoot, configPath, configMapPath string
	var nWorkers int
	var metadata, skipNoconfig, skipNorepos bool
	flag.StringVar(&outputDir, "output", "test/data/manifests/", "manifest store directory")
	flag.IntVar(&nWorkers, "workers", 16, "number of workers to run concurrently")
	flag.StringVar(&cacheRoot, "cache", "/tmp/rpmmd", "rpm metadata cache directory")
	flag.BoolVar(&metadata, "metadata", true, "store metadata in the file")
	flag.StringVar(&configPath, "config", "", "image config file to use for all images (overrides -config-map)")
	flag.StringVar(&configMapPath, "config-map", "test/config-map.json", "configuration file mapping image types to configs")
	flag.BoolVar(&skipNoconfig, "skip-noconfig", false, "skip distro-arch-image configurations that have no config (otherwise fail)")
	flag.BoolVar(&skipNorepos, "skip-norepos", false, "skip distro-arch-image configurations that have no repositories (otherwise fail)")

	// content args
	var packages, containers, commits bool
	flag.BoolVar(&packages, "packages", true, "depsolve package sets")
	flag.BoolVar(&containers, "containers", true, "resolve container checksums")
	flag.BoolVar(&commits, "commits", false, "resolve ostree commit IDs")

	// manifest selection args
	var arches, distros, imgTypes cmdutil.MultiValue
	flag.Var(&arches, "arches", "comma-separated list of architectures (globs supported)")
	flag.Var(&distros, "distros", "comma-separated list of distributions (globs supported)")
	flag.Var(&imgTypes, "types", "comma-separated list of image types (globs supported)")

	flag.Parse()

	testedRepoRegistry, err := reporegistry.NewTestedDefault()
	if err != nil {
		panic(fmt.Sprintf("failed to create repo registry with tested distros: %v", err))
	}

	distroFac := distrofactory.NewDefault()
	jobs := make([]manifestJob, 0)

	contentResolve := map[string]bool{
		"packages":   packages,
		"containers": containers,
		"commits":    commits,
	}

	var configs BuildConfigs
	if configPath != "" {
		fmt.Println("'-config' was provided, thus ignoring '-config-map' option")
		configs = loadImgConfig(configPath)
	} else {
		configs = loadConfigMap(configMapPath)
	}

	if err := os.MkdirAll(outputDir, 0770); err != nil {
		panic(fmt.Sprintf("failed to create target directory: %s", err.Error()))
	}

	fmt.Println("Collecting jobs")

	distros, invalidDistros := distros.ResolveArgValues(testedRepoRegistry.ListDistros())
	if len(invalidDistros) > 0 {
		fmt.Fprintf(os.Stderr, "WARNING: invalid distro names: [%s]\n", strings.Join(invalidDistros, ","))
	}
	for _, distroName := range distros {
		distribution := distroFac.GetDistro(distroName)
		if distribution == nil {
			fmt.Fprintf(os.Stderr, "WARNING: invalid distro name %q\n", distroName)
			continue
		}

		distroArches, invalidArches := arches.ResolveArgValues(distribution.ListArches())
		if len(invalidArches) > 0 {
			fmt.Fprintf(os.Stderr, "WARNING: invalid arch names [%s] for distro %q\n", strings.Join(invalidArches, ","), distroName)
		}
		for _, archName := range distroArches {
			arch, err := distribution.GetArch(archName)
			if err != nil {
				// resolveArgValues should prevent this
				panic(fmt.Sprintf("invalid arch name %q for distro %q: %s\n", archName, distroName, err.Error()))
			}

			daImgTypes, invalidImageTypes := imgTypes.ResolveArgValues(arch.ListImageTypes())
			if len(invalidImageTypes) > 0 {
				fmt.Fprintf(os.Stderr, "WARNING: invalid image type names [%s] for distro %q and arch %q\n", strings.Join(invalidImageTypes, ","), distroName, archName)
			}
			for _, imgTypeName := range daImgTypes {
				imgType, err := arch.GetImageType(imgTypeName)
				if err != nil {
					// resolveArgValues should prevent this
					panic(fmt.Sprintf("invalid image type %q for distro %q and arch %q: %s\n", imgTypeName, distroName, archName, err.Error()))
				}

				// get repositories
				repos, err := testedRepoRegistry.ReposByArchName(distroName, archName, true)
				if err != nil {
					panic(fmt.Sprintf("failed to get repositories for %s/%s: %v", distroName, archName, err))
				}
				repos = filterRepos(repos, imgTypeName)
				if len(repos) == 0 {
					fmt.Printf("no repositories defined for %s/%s/%s\n", distroName, archName, imgTypeName)
					if skipNorepos {
						fmt.Println("Skipping")
						continue
					}
					panic("no repositories found, pass --skip-norepos to skip")
				}

				imgTypeConfigs := configs.Get(distroName, archName, imgTypeName)
				if len(imgTypeConfigs) == 0 {
					if skipNoconfig {
						continue
					}
					panic(fmt.Sprintf("no configs defined for image type %q", imgTypeName))
				}

				for _, itConfig := range imgTypeConfigs {
					job := makeManifestJob(itConfig, imgType, distribution, repos, archName, cacheRoot, outputDir, contentResolve, metadata)
					jobs = append(jobs, job)
				}
			}
		}
	}

	nJobs := len(jobs)
	fmt.Printf("Collected %d jobs\n", nJobs)

	// nolint:gosec
	wq := newWorkerQueue(uint32(nWorkers), uint32(nJobs))
	wq.start()
	fmt.Printf("Initialised %d workers\n", nWorkers)
	fmt.Printf("Submitting %d jobs... ", nJobs)
	for _, j := range jobs {
		wq.submitJob(j)
	}
	fmt.Println("done")
	errs := wq.wait()
	exit := 0
	if len(errs) > 0 {
		fmt.Fprintf(os.Stderr, "Encountered %d errors:\n", len(errs))
		for idx, err := range errs {
			fmt.Fprintf(os.Stderr, "%3d: %s\n", idx, err.Error())
		}
		exit = 1
	}
	fmt.Printf("RPM metadata cache kept in %s\n", cacheRoot)
	os.Exit(exit)
}
