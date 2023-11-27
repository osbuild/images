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
	"os"
	"path/filepath"
	"strings"

	"github.com/gobwas/glob"

	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distroregistry"
	"github.com/osbuild/images/pkg/dnfjson"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/ostree"
	"github.com/osbuild/images/pkg/rhsm/facts"
	"github.com/osbuild/images/pkg/rpmmd"
)

type multiValue []string

func (mv *multiValue) String() string {
	return strings.Join(*mv, ", ")
}

func (mv *multiValue) Set(v string) error {
	split := strings.Split(v, ",")
	*mv = split
	return nil
}

type buildRequest struct {
	Distro       string             `json:"distro,omitempty"`
	Arch         string             `json:"arch,omitempty"`
	ImageType    string             `json:"image-type,omitempty"`
	Repositories []rpmmd.RepoConfig `json:"repositories,omitempty"`
	Config       *BuildConfig       `json:"config"`
}

type BuildConfig struct {
	Name      string               `json:"name"`
	OSTree    *ostree.ImageOptions `json:"ostree,omitempty"`
	Blueprint *blueprint.Blueprint `json:"blueprint,omitempty"`
	Depends   BuildDependency      `json:"depends,omitempty"`
}

type BuildDependency struct {
	Config    string `json:"config"`
	ImageType string `json:"image-type"`
}

// BuildConfigs is a nested map representing the configs to use for each
// distro/arch/image-type. If any component is empty, it maps to all values.
type BuildConfigs map[string]map[string]map[string][]BuildConfig

func (bc BuildConfigs) Insert(distro, arch, imageType string, cfg BuildConfig) {
	distroCfgs := bc[distro]
	if distroCfgs == nil {
		distroCfgs = make(map[string]map[string][]BuildConfig)
	}

	distroArchCfgs := distroCfgs[arch]
	if distroArchCfgs == nil {
		distroArchCfgs = make(map[string][]BuildConfig)
	}

	distroArchItCfgs := distroArchCfgs[imageType]
	if distroArchItCfgs == nil {
		distroArchItCfgs = make([]BuildConfig, 0)
	}

	distroArchItCfgs = append(distroArchItCfgs, cfg)
	distroArchCfgs[imageType] = distroArchItCfgs
	distroCfgs[arch] = distroArchCfgs
	bc[distro] = distroCfgs
}

func (bc BuildConfigs) Get(distro, arch, imageType string) []BuildConfig {
	configs := make([]BuildConfig, 0)
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

func loadConfig(path string) BuildConfig {
	fp, err := os.Open(path)
	if err != nil {
		panic(fmt.Sprintf("failed to open config %q: %s", path, err.Error()))
	}
	defer fp.Close()

	dec := json.NewDecoder(fp)
	dec.DisallowUnknownFields()
	var conf BuildConfig

	if err := dec.Decode(&conf); err != nil {
		panic(fmt.Sprintf("failed to unmarshal config %q: %s", path, err.Error()))
	}
	if dec.More() {
		panic(fmt.Sprintf("multiple configuration objects or extra data found in %q", path))
	}
	return conf
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
		config := loadConfig(path)
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
	config := loadConfig(configPath)
	cm.Insert("*", "*", "*", config)
	return cm
}

type manifestJob func(chan string) error

func makeManifestJob(
	name string,
	imgType distro.ImageType,
	bc BuildConfig,
	distribution distro.Distro,
	repos []rpmmd.RepoConfig,
	archName string,
	seedArg int64,
	path string,
	cacheRoot string,
	content map[string]bool,
	metadata bool,
) manifestJob {
	distroName := distribution.Name()
	filename := fmt.Sprintf("%s-%s-%s-%s.json", u(distroName), u(archName), u(imgType.Name()), u(name))
	cacheDir := filepath.Join(cacheRoot, archName+distribution.Name())

	options := distro.ImageOptions{Size: 0}
	options.OSTree = bc.OSTree

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
		if content["packages"] {
			packageSpecs, err = depsolve(cacheDir, manifest.GetPackageSetChains(), distribution, archName)
			if err != nil {
				err = fmt.Errorf("[%s] depsolve failed: %s", filename, err.Error())
				return
			}
			if packageSpecs == nil {
				err = fmt.Errorf("[%s] nil package specs", filename)
				return
			}
		} else {
			packageSpecs = mockDepsolve(manifest.GetPackageSetChains())
		}

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

		mf, err := manifest.Serialize(packageSpecs, containerSpecs, commitSpecs)
		if err != nil {
			return fmt.Errorf("[%s] manifest serialization failed: %s", filename, err.Error())
		}

		request := buildRequest{
			Distro:       distribution.Name(),
			Arch:         archName,
			ImageType:    imgType.Name(),
			Repositories: repos,
			Config:       &bc,
		}
		err = save(mf, packageSpecs, containerSpecs, commitSpecs, request, path, filename, metadata)
		return
	}
	return job
}

type DistroArchRepoMap map[string]map[string][]rpmmd.RepoConfig

func readRepos() DistroArchRepoMap {
	reposDir := "./test/data/repositories/"
	darm := make(DistroArchRepoMap)
	filelist, err := os.ReadDir(reposDir)
	if err != nil {
		panic(err)
	}
	for _, file := range filelist {
		filename := file.Name()
		if !strings.HasSuffix(filename, ".json") {
			continue
		}
		reposFilepath := filepath.Join(reposDir, filename)
		fp, err := os.Open(reposFilepath)
		if err != nil {
			panic(err)
		}
		defer fp.Close()
		data, err := io.ReadAll(fp)
		if err != nil {
			panic(err)
		}
		repos := make(map[string][]rpmmd.RepoConfig)
		if err := json.Unmarshal(data, &repos); err != nil {
			panic(err)
		}
		distro := strings.TrimSuffix(filename, filepath.Ext(filename))
		darm[distro] = repos
	}
	return darm
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
			checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(commitSource.URL+commitSource.Ref)))
			spec := ostree.CommitSpec{
				Ref:      commitSource.Ref,
				URL:      commitSource.URL,
				Checksum: checksum,
			}
			if commitSource.RHSM {
				spec.Secrets = "org.osbuild.rhsm.consumer"
			}
			commitSpecs[idx] = spec
		}
		commits[name] = commitSpecs
	}
	return commits
}

func depsolve(cacheDir string, packageSets map[string][]rpmmd.PackageSet, d distro.Distro, arch string) (map[string][]rpmmd.PackageSpec, error) {
	solver := dnfjson.NewSolver(d.ModulePlatformID(), d.Releasever(), arch, d.Name(), cacheDir)
	solver.SetDNFJSONPath("./dnf-json")
	depsolvedSets := make(map[string][]rpmmd.PackageSpec)
	for name, pkgSet := range packageSets {
		res, err := solver.Depsolve(pkgSet)
		if err != nil {
			return nil, err
		}
		depsolvedSets[name] = res
	}
	return depsolvedSets, nil
}

func mockDepsolve(packageSets map[string][]rpmmd.PackageSet) map[string][]rpmmd.PackageSpec {
	depsolvedSets := make(map[string][]rpmmd.PackageSpec)
	for name, pkgSetChain := range packageSets {
		specSet := make([]rpmmd.PackageSpec, 0)
		for _, pkgSet := range pkgSetChain {
			for _, pkgName := range pkgSet.Include {
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
		depsolvedSets[name] = specSet
	}
	return depsolvedSets
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

// resolveArgValues returns a list of valid values from the list of values on the
// command line. Invalid values are returned separately. Globs are expanded.
// If the args are empty, the valueList is returned as is.
func resolveArgValues(args multiValue, valueList []string) ([]string, []string) {
	if len(args) == 0 {
		return valueList, nil
	}
	selection := make([]string, 0, len(args))
	invalid := make([]string, 0, len(args))
	for _, arg := range args {
		g := glob.MustCompile(arg)
		match := false
		for _, v := range valueList {
			if g.Match(v) {
				selection = append(selection, v)
				match = true
			}
		}
		if !match {
			invalid = append(invalid, arg)
		}
	}
	return selection, invalid
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
	var arches, distros, imgTypes multiValue
	flag.Var(&arches, "arches", "comma-separated list of architectures (globs supported)")
	flag.Var(&distros, "distros", "comma-separated list of distributions (globs supported)")
	flag.Var(&imgTypes, "images", "comma-separated list of image types (globs supported)")

	flag.Parse()

	seedArg := int64(0)
	darm := readRepos()
	distroReg := distroregistry.NewDefault()
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
	distros, invalidDistros := resolveArgValues(distros, distroReg.List())
	if len(invalidDistros) > 0 {
		fmt.Fprintf(os.Stderr, "WARNING: invalid distro names: [%s]\n", strings.Join(invalidDistros, ","))
	}
	for _, distroName := range distros {
		distribution := distroReg.GetDistro(distroName)

		distroArches, invalidArches := resolveArgValues(arches, distribution.ListArches())
		if len(invalidArches) > 0 {
			fmt.Fprintf(os.Stderr, "WARNING: invalid arch names [%s] for distro %q\n", strings.Join(invalidArches, ","), distroName)
		}
		for _, archName := range distroArches {
			arch, err := distribution.GetArch(archName)
			if err != nil {
				// resolveArgValues should prevent this
				panic(fmt.Sprintf("invalid arch name %q for distro %q: %s\n", archName, distroName, err.Error()))
			}

			daImgTypes, invalidImageTypes := resolveArgValues(imgTypes, arch.ListImageTypes())
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
				repos := darm[distroName][archName]
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
					job := makeManifestJob(itConfig.Name, imgType, itConfig, distribution, repos, archName, seedArg, outputDir, cacheRoot, contentResolve, metadata)
					jobs = append(jobs, job)
				}
			}
		}
	}

	nJobs := len(jobs)
	fmt.Printf("Collected %d jobs\n", nJobs)
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
