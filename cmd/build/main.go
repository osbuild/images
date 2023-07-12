// Standalone executable for building a test image.
// Must be run from the root of the repository and reads
// tools/test-case-generators/repos.json for repositories
// tools/test-case-generators/format-request-map.json for customizations

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

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/internal/dnfjson"
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distroregistry"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/ostree"
	"github.com/osbuild/images/pkg/rhsm/facts"
	"github.com/osbuild/images/pkg/rpmmd"
)

type repository struct {
	Name           string   `json:"name"`
	Id             string   `json:"id,omitempty"`
	BaseURL        string   `json:"baseurl,omitempty"`
	Metalink       string   `json:"metalink,omitempty"`
	MirrorList     string   `json:"mirrorlist,omitempty"`
	GPGKey         string   `json:"gpgkey,omitempty"`
	CheckGPG       bool     `json:"check_gpg,omitempty"`
	CheckRepoGPG   bool     `json:"check_repo_gpg,omitempty"`
	IgnoreSSL      bool     `json:"ignore_ssl,omitempty"`
	RHSM           bool     `json:"rhsm,omitempty"`
	MetadataExpire string   `json:"metadata_expire,omitempty"`
	ImageTypeTags  []string `json:"image_type_tags,omitempty"`
	PackageSets    []string `json:"package-sets,omitempty"`
}

type ostreeOptions struct {
	Ref    string `json:"ref"`
	URL    string `json:"url"`
	Parent string `json:"parent"`
	RHSM   bool   `json:"rhsm"`
}

type crBlueprint struct {
	Name           string                    `json:"name,omitempty"`
	Description    string                    `json:"description,omitempty"`
	Version        string                    `json:"version,omitempty"`
	Packages       []blueprint.Package       `json:"packages,omitempty"`
	Modules        []blueprint.Package       `json:"modules,omitempty"`
	Groups         []blueprint.Group         `json:"groups,omitempty"`
	Containers     []blueprint.Container     `json:"containers,omitempty"`
	Customizations *blueprint.Customizations `json:"customizations,omitempty"`
	Distro         string                    `json:"distro,omitempty"`
}

type composeRequest struct {
	Distro       string         `json:"distro,omitempty"`
	Arch         string         `json:"arch,omitempty"`
	ImageType    string         `json:"image-type,omitempty"`
	Repositories []repository   `json:"repositories,omitempty"`
	Filename     string         `json:"filename,omitempty"`
	OSTree       *ostreeOptions `json:"ostree,omitempty"`
	Blueprint    *crBlueprint   `json:"blueprint,omitempty"`
}

type manifestRequest struct {
	ComposeRequest  composeRequest            `json:"compose-request"`
	Overrides       map[string]composeRequest `json:"overrides"`
	SupportedArches []string                  `json:"supported_arches"`
}

type formatRequestMap map[string]manifestRequest

func loadFormatRequestMap() formatRequestMap {
	requestMapPath := "./tools/test-case-generators/format-request-map.json"
	fp, err := os.Open(requestMapPath)
	if err != nil {
		panic(fmt.Sprintf("failed to open format request map %q: %s", requestMapPath, err.Error()))
	}
	defer fp.Close()
	data, err := io.ReadAll(fp)
	if err != nil {
		panic(fmt.Sprintf("failed to read format request map %q: %s", requestMapPath, err.Error()))
	}
	var frm formatRequestMap
	if err := json.Unmarshal(data, &frm); err != nil {
		panic(fmt.Sprintf("failed to unmarshal format request map %q: %s", requestMapPath, err.Error()))
	}

	return frm
}

type manifestJob func(chan string) error

func makeManifest(name string, imgType distro.ImageType, cr composeRequest, distribution distro.Distro, archName string, seedArg int64, cacheRoot string) (manifest.OSBuildManifest, error) {
	cacheDir := filepath.Join(cacheRoot, archName+distribution.Name())

	options := distro.ImageOptions{Size: 0}
	if cr.OSTree != nil {
		options.OSTree = &ostree.ImageOptions{
			URL:       cr.OSTree.URL,
			ImageRef:  cr.OSTree.Ref,
			ParentRef: cr.OSTree.Parent,
			RHSM:      cr.OSTree.RHSM,
		}
	}

	// add RHSM fact to detect changes
	options.Facts = &facts.ImageOptions{
		APIType: facts.TEST_APITYPE,
	}

	repos := convertRepos(cr.Repositories)
	var bp blueprint.Blueprint
	if cr.Blueprint != nil {
		bp = blueprint.Blueprint(*cr.Blueprint)
	}

	manifest, warnings, err := imgType.Manifest(&bp, options, repos, seedArg)
	if err != nil {
		return nil, fmt.Errorf("[ERROR] manifest generation failed: %s", err.Error())
	}
	if len(warnings) > 0 {
		fmt.Fprintf(os.Stderr, "[WARNING]\n%s", strings.Join(warnings, "\n"))
	}

	packageSpecs, err := depsolve(cacheDir, manifest.GetPackageSetChains(), distribution, archName)
	if err != nil {
		return nil, fmt.Errorf("[ERROR] depsolve failed: %s", err.Error())
	}
	if packageSpecs == nil {
		return nil, fmt.Errorf("[ERROR] depsolve did not return any packages")
	}

	if cr.Blueprint != nil {
		bp = blueprint.Blueprint(*cr.Blueprint)
	}

	containerSpecs, err := resolvePipelineContainers(manifest.GetContainerSourceSpecs(), archName)
	if err != nil {
		return nil, fmt.Errorf("[ERROR] container resolution failed: %s", err.Error())
	}

	commitSpecs := resolvePipelineCommits(manifest.GetOSTreeSourceSpecs())

	mf, err := manifest.Serialize(packageSpecs, containerSpecs, commitSpecs)
	if err != nil {
		return nil, fmt.Errorf("[ERROR] manifest serialization failed: %s", err.Error())
	}

	return mf, nil
}

type DistroArchRepoMap map[string]map[string][]repository

func convertRepo(r repository) rpmmd.RepoConfig {
	var urls []string
	if r.BaseURL != "" {
		urls = []string{r.BaseURL}
	}

	var keys []string
	if r.GPGKey != "" {
		keys = []string{r.GPGKey}
	}

	return rpmmd.RepoConfig{
		Id:             r.Id,
		Name:           r.Name,
		BaseURLs:       urls,
		Metalink:       r.Metalink,
		MirrorList:     r.MirrorList,
		GPGKeys:        keys,
		CheckGPG:       &r.CheckGPG,
		CheckRepoGPG:   &r.CheckRepoGPG,
		IgnoreSSL:      &r.IgnoreSSL,
		MetadataExpire: r.MetadataExpire,
		RHSM:           r.RHSM,
		ImageTypeTags:  r.ImageTypeTags,
		PackageSets:    r.PackageSets,
	}
}

func convertRepos(rr []repository) []rpmmd.RepoConfig {
	cr := make([]rpmmd.RepoConfig, len(rr))
	for idx, r := range rr {
		cr[idx] = convertRepo(r)
	}
	return cr
}

func readRepos() DistroArchRepoMap {
	file := "./tools/test-case-generators/repos.json"
	var darm DistroArchRepoMap
	fp, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer fp.Close()
	data, err := io.ReadAll(fp)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(data, &darm); err != nil {
		panic(err)
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

func resolveCommit(commitSource ostree.SourceSpec) ostree.CommitSpec {
	// "resolve" ostree commits by hashing the URL + ref to create a
	// realistic-looking commit ID in a deterministic way
	checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(commitSource.URL+commitSource.Ref)))
	spec := ostree.CommitSpec{
		Ref:      commitSource.Ref,
		URL:      commitSource.URL,
		Checksum: checksum,
	}
	if commitSource.RHSM {
		spec.Secrets = "org.osbuild.rhsm.consumer"
	}
	return spec
}

func resolvePipelineCommits(commitSources map[string][]ostree.SourceSpec) map[string][]ostree.CommitSpec {
	commits := make(map[string][]ostree.CommitSpec, len(commitSources))
	for name, commitSources := range commitSources {
		commitSpecs := make([]ostree.CommitSpec, len(commitSources))
		for idx, commitSource := range commitSources {
			commitSpecs[idx] = resolveCommit(commitSource)
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

func filterRepos(repos []repository, typeName string) []repository {
	filtered := make([]repository, 0)
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

// collects requests from a formatRequestMap based on image type
func requestsByImageType(requestMap formatRequestMap) map[string]map[string]manifestRequest {
	imgTypeRequestMap := make(map[string]map[string]manifestRequest)

	for name, req := range requestMap {
		it := req.ComposeRequest.ImageType
		reqs := imgTypeRequestMap[it]
		if reqs == nil {
			reqs = make(map[string]manifestRequest)
		}
		reqs[name] = req
		imgTypeRequestMap[it] = reqs
	}
	return imgTypeRequestMap
}

func archIsSupported(req manifestRequest, arch string) bool {
	if len(req.SupportedArches) == 0 {
		// none specified: all arches supported implicitly
		return true
	}
	for _, supportedArch := range req.SupportedArches {
		if supportedArch == arch {
			return true
		}
	}
	return false
}

func mergeOverrides(base, overrides composeRequest) composeRequest {
	// NOTE: in most cases overrides are only used for blueprints and probably
	// doesn't make sense to use them for most fields, but let's merge all
	// regardless
	merged := composeRequest(base)
	if overrides.Blueprint != nil {
		merged.Blueprint = overrides.Blueprint
	}

	if overrides.Filename != "" {
		merged.Filename = overrides.Filename
	}
	if overrides.ImageType != "" {
		merged.ImageType = overrides.ImageType
	}
	if overrides.OSTree != nil {
		merged.OSTree = overrides.OSTree
	}
	if overrides.Distro != "" {
		merged.Distro = overrides.Distro
	}
	if overrides.Arch != "" {
		merged.Arch = overrides.Arch
	}
	if len(overrides.Repositories) > 0 {
		merged.Repositories = overrides.Repositories
	}
	return merged
}

func main() {
	// common args
	var outputDir, osbuildStore, rpmCacheRoot string
	flag.StringVar(&outputDir, "output", ".", "artifact output directory")
	flag.StringVar(&osbuildStore, "store", ".osbuild", "osbuild store for intermediate pipeline trees")
	flag.StringVar(&rpmCacheRoot, "rpmmd", "/tmp/rpmmd", "rpm metadata cache directory")

	// image selection args
	var distroName, imgTypeName string
	flag.StringVar(&distroName, "distro", "", "distribution (required)")
	flag.StringVar(&imgTypeName, "image", "", "image type name (required)")

	flag.Parse()

	if distroName == "" || imgTypeName == "" {
		flag.Usage()
		os.Exit(1)
	}

	seedArg := int64(0)
	darm := readRepos()
	distroReg := distroregistry.NewDefault()

	requestMap := loadFormatRequestMap()
	itRequestMap := requestsByImageType(requestMap)

	if err := os.MkdirAll(outputDir, 0770); err != nil {
		panic(fmt.Sprintf("failed to create target directory: %s", err.Error()))
	}

	distribution := distroReg.GetDistro(distroName)
	if distribution == nil {
		panic(fmt.Sprintf("invalid or unsupported distribution: %q", distroName))
	}

	archName := common.CurrentArch()
	arch, err := distribution.GetArch(archName)
	if err != nil {
		panic(fmt.Sprintf("invalid arch name %q for distro %q: %s\n", archName, distroName, err.Error()))
	}
	imgType, err := arch.GetImageType(imgTypeName)
	if err != nil {
		panic(fmt.Sprintf("invalid image type %q for distro %q and arch %q: %s\n", imgTypeName, distroName, archName, err.Error()))
	}

	// get repositories
	repos := darm[distroName][archName]
	if len(repos) == 0 {
		panic(fmt.Sprintf("no repositories defined for %s/%s\n", distroName, archName))
	}

	// run through jobs from request map that match the image type
	for jobName, req := range itRequestMap[imgTypeName] {
		// skip if architecture is not supported
		if !archIsSupported(req, archName) {
			continue
		}

		// check for distro-specific overrides
		if or, exist := req.Overrides[distroName]; exist {
			req.ComposeRequest = mergeOverrides(req.ComposeRequest, or)
		}

		composeReq := req.ComposeRequest
		composeReq.Repositories = filterRepos(repos, imgTypeName)

		fmt.Printf("Generating manifest for %s: ", jobName)
		mf, err := makeManifest(jobName, imgType, composeReq, distribution, archName, seedArg, rpmCacheRoot)
		if err != nil {
			panic(err)
		}
		fmt.Print("DONE\n")
		distroName := distribution.Name()

		u := func(s string) string {
			return strings.Replace(s, "-", "_", -1)
		}
		fileBasename := fmt.Sprintf("%s-%s-%s", u(distroName), u(archName), u(jobName))
		filename := fileBasename + ".json"
		manifestPath := filepath.Join(outputDir, filename)
		if err := save(mf, manifestPath); err != nil {
			panic(err)
		}

		fmt.Printf("Building manifest: %s\n", manifestPath)

		jobOutput := filepath.Join(outputDir, fileBasename)
		if _, err := osbuild.RunOSBuild(mf, osbuildStore, jobOutput, imgType.Exports(), nil, nil, false, os.Stderr); err != nil {
			panic(err)
		}

	}
	fmt.Printf("Jobs done. Results saved in\n%s\n", outputDir)
}
