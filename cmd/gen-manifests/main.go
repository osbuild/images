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
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gobwas/glob"

	"github.com/osbuild/images/internal/buildconfig"
	"github.com/osbuild/images/internal/cmdutil"
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distrofactory"
	"github.com/osbuild/images/pkg/dnfjson"
	"github.com/osbuild/images/pkg/experimentalflags"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/manifestgen"
	"github.com/osbuild/images/pkg/ostree"
	"github.com/osbuild/images/pkg/rhsm/facts"
	"github.com/osbuild/images/pkg/rpmmd"
	testrepos "github.com/osbuild/images/test/data/repositories"
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

type skipDistro struct {
	Name   string    `json:"name"`
	Reason string    `json:"reason"`
	Date   time.Time `json:"date"`
}

// BuildConfigs is a nested map representing the configs to use for each
// distro/arch/image-type. If any component is empty, it maps to all values.
type BuildConfigs struct {
	confMap  map[string]map[string]map[string][]*buildconfig.BuildConfig
	skipList map[*buildconfig.BuildConfig][]skipDistro
}

func newBuildConfigs() *BuildConfigs {
	return &BuildConfigs{
		confMap:  make(map[string]map[string]map[string][]*buildconfig.BuildConfig),
		skipList: make(map[*buildconfig.BuildConfig][]skipDistro),
	}
}

func (bc BuildConfigs) Insert(distro, arch, imageType string, cfg *buildconfig.BuildConfig) {
	distroCfgs := bc.confMap[distro]
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
	bc.confMap[distro] = distroCfgs
}

func (bc BuildConfigs) needsSkipping(distro string, cfg *buildconfig.BuildConfig) (bool, string) {
	for _, s := range bc.skipList[cfg] {
		if s.Name == distro {
			if time.Since(s.Date) > 90*24*time.Hour {
				err := fmt.Errorf("distro %q is temporarily skipped for more than 90 days (added %q)", s.Name, s.Date)
				panic(err)
			}
			return true, s.Reason
		}
	}

	return false, ""
}

func (bc BuildConfigs) Get(distro, arch, imageType string) []*buildconfig.BuildConfig {

	configs := make([]*buildconfig.BuildConfig, 0)
	for distroName, distroCfgs := range bc.confMap {
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

func loadConfigMap(configPath string) *BuildConfigs {
	type configFilters struct {
		ImageTypes  []string     `json:"image-types"`
		Distros     []string     `json:"distros"`
		SkipDistros []skipDistro `json:"skip-distros"`
		Arches      []string     `json:"arches"`
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
	cm := newBuildConfigs()
	for path, filters := range cfgMap {
		// config paths can be relative to the location of the config map
		if !filepath.IsAbs(path) {
			cfgDir := filepath.Dir(configPath)
			path = filepath.Join(cfgDir, path)
		}
		config, err := buildconfig.New(path, nil)
		if err != nil {
			panic(err)
		}
		for _, d := range emptyFallback(filters.Distros) {
			if len(filters.SkipDistros) > 0 {
				cm.skipList[config] = append(cm.skipList[config], filters.SkipDistros...)
			}
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
func loadImgConfig(configPath string) *BuildConfigs {
	cm := newBuildConfigs()
	config, err := buildconfig.New(configPath, nil)
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
	if experimentalflags.Bool("gen-manifest-mock-bpfile-uris") && bp.Customizations != nil {
		for i, fc := range bp.Customizations.Files {
			// in mock mode, replace all file customizations
			// urls with local ones that we auto-generate
			// with predictable content (from the URL) and
			// a predictable location (under cacheRoot)
			if fc.URI != "" {
				newBpFileUrl := filepath.Join(cacheRoot, "fake-bp-files-with-urls", fmt.Sprintf("%x", sha256.Sum256([]byte(fc.URI))))
				if err := os.MkdirAll(filepath.Dir(newBpFileUrl), 0755); err != nil {
					panic(err)
				}
				if err := os.WriteFile(newBpFileUrl, []byte(fc.URI), 0600); err != nil {
					panic(err)
				}
				bp.Customizations.Files[i].URI = "file://" + newBpFileUrl
			}
		}
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

		manifest, _, err := imgType.Manifest(&bp, options, repos, &seedArg)
		if err != nil {
			err = fmt.Errorf("[%s] failed: %s", filename, err)
			return
		}

		var depsolvedSets map[string]dnfjson.DepsolveResult
		if content["packages"] {
			depsolvedSets, err = manifestgen.DefaultDepsolver(cacheDir, os.Stderr, manifest.GetPackageSetChains(), distribution, archName)
			if err != nil {
				err = fmt.Errorf("[%s] depsolve failed: %s", filename, err.Error())
				return
			}
			for plName, depsolved := range depsolvedSets {
				if depsolved.Packages == nil {
					err = fmt.Errorf("[%s] nil package specs in %v", filename, plName)
					return
				}
			}
		} else {
			depsolvedSets = mockDepsolve(manifest.GetPackageSetChains(), repos, archName)
		}

		var containerSpecs map[string][]container.Spec
		if content["containers"] {
			containerSpecs, err = manifestgen.DefaultContainerResolver(manifest.GetContainerSourceSpecs(), archName)
			if err != nil {
				return fmt.Errorf("[%s] container resolution failed: %s", filename, err.Error())
			}
		} else {
			containerSpecs = mockResolveContainers(manifest.GetContainerSourceSpecs())
		}

		var commitSpecs map[string][]ostree.CommitSpec
		if content["commits"] {
			commitSpecs, err = manifestgen.DefaultCommitResolver(manifest.GetOSTreeSourceSpecs())
			if err != nil {
				return fmt.Errorf("[%s] ostree commit resolution failed: %s", filename, err.Error())
			}
		} else {
			commitSpecs = mockResolveCommits(manifest.GetOSTreeSourceSpecs())
		}

		mf, err := manifest.Serialize(depsolvedSets, containerSpecs, commitSpecs, nil)
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
		err = save(mf, depsolvedSets, containerSpecs, commitSpecs, request, path, filename, metadata)
		return
	}
	return job
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

func mockDepsolve(packageSets map[string][]rpmmd.PackageSet, repos []rpmmd.RepoConfig, archName string) map[string]dnfjson.DepsolveResult {
	depsolvedSets := make(map[string]dnfjson.DepsolveResult)

	for name, pkgSetChain := range packageSets {
		specSet := make([]rpmmd.PackageSpec, 0)
		seenChksumsInc := make(map[string]bool)
		seenChksumsExc := make(map[string]bool)
		for idx, pkgSet := range pkgSetChain {
			include := pkgSet.Include
			slices.Sort(include)
			for _, pkgName := range include {
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
				if seenChksumsInc[spec.Checksum] {
					continue
				}
				seenChksumsInc[spec.Checksum] = true

				specSet = append(specSet, spec)
			}

			exclude := pkgSet.Exclude
			slices.Sort(exclude)
			for _, excludeName := range exclude {
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
				if seenChksumsExc[spec.Checksum] {
					continue
				}
				seenChksumsExc[spec.Checksum] = true

				specSet = append(specSet, spec)
			}

			// generate pseudo packages for the config of each transaction
			var setRepoNames []string
			for _, setRepo := range pkgSet.Repositories {
				setRepoNames = append(setRepoNames, setRepo.Name)
			}
			configPackageName := fmt.Sprintf("%s:transaction-%d-repos:%s", name, idx, strings.Join(setRepoNames, "+"))
			if pkgSet.InstallWeakDeps {
				configPackageName += "-weak"
			}
			depsolveConfigPackage := rpmmd.PackageSpec{
				Name:           configPackageName,
				Epoch:          0,
				Version:        "",
				Release:        "",
				Arch:           "noarch",
				RemoteLocation: fmt.Sprintf("https://example.com/repo/packages/%s", configPackageName),
				Checksum:       fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(configPackageName))),
				Secrets:        "",
				CheckGPG:       false,
				IgnoreSSL:      false,
				Path:           "",
				RepoID:         "",
			}
			specSet = append(specSet, depsolveConfigPackage)
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

		depsolvedSets[name] = dnfjson.DepsolveResult{
			Packages: specSet,
			Repos:    repos,
		}
	}

	return depsolvedSets
}

func save(ms manifest.OSBuildManifest, depsolved map[string]dnfjson.DepsolveResult, containers map[string][]container.Spec, commits map[string][]ostree.CommitSpec, cr buildRequest, path, filename string, metadata bool) error {
	var data interface{}
	if metadata {
		rpmmds := make(map[string][]rpmmd.PackageSpec)
		for plName, res := range depsolved {
			rpmmds[plName] = res.Packages
		}
		data = struct {
			BuidRequest   buildRequest                   `json:"build-request"`
			Manifest      manifest.OSBuildManifest       `json:"manifest"`
			RPMMD         map[string][]rpmmd.PackageSpec `json:"rpmmd"`
			Containers    map[string][]container.Spec    `json:"containers,omitempty"`
			OSTreeCommits map[string][]ostree.CommitSpec `json:"ostree-commits,omitempty"`
			NoImageInfo   bool                           `json:"no-image-info"`
		}{
			cr, ms, rpmmds, containers, commits, true,
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

	testedRepoRegistry, err := testrepos.New()
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

	var configs *BuildConfigs
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
					panic(fmt.Sprintf("no configs defined for image type %q for %s", imgTypeName, distribution.Name()))
				}

				for _, itConfig := range imgTypeConfigs {
					if needsSkipping, reason := configs.needsSkipping(distribution.Name(), itConfig); needsSkipping {
						fmt.Printf("Skipping %s for %s/%s (reason: %v)\n", itConfig.Name, imgTypeName, distribution.Name(), reason)
						continue
					}

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
