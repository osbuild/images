package manifestgen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/dnfjson"
	"github.com/osbuild/images/pkg/manifest"
	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/ostree"
	"github.com/osbuild/images/pkg/reporegistry"
	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/osbuild/images/pkg/sbom"
)

const (
	defaultDepsolverSBOMType = sbom.StandardTypeSpdx
	defaultSBOMExt           = "spdx.json"

	defaultDepsolveCacheDir = "osbuild-depsolve-dnf"
)

// Options contains the optional settings for the manifest generation.
// For unset values defaults will be used.
type Options struct {
	Cachedir string
	// Output is the writer that the generated osbuild manifest will
	// written to.
	Output io.Writer

	RpmDownloader osbuild.RpmDownloader

	// SBOMWriter will be called for each generated SBOM the
	// filename contains the suggest filename string and the
	// content can be read
	SBOMWriter SBOMWriterFunc

	// CustomSeed overrides the default rng seed, this is mostly
	// useful for testing
	CustomSeed *int64

	// Custom "solver" functions, if unset the defaults will be
	// used. Only needed for specialized use-cases.
	Depsolver         DepsolveFunc
	ContainerResolver ContainerResolverFunc
	CommitResolver    CommitResolverFunc
}

// Generator can generate an osbuild manifest from a given repository
// and options.
type Generator struct {
	cacheDir string
	out      io.Writer

	depsolver         DepsolveFunc
	containerResolver ContainerResolverFunc
	commitResolver    CommitResolverFunc
	sbomWriter        SBOMWriterFunc

	reporegistry *reporegistry.RepoRegistry

	rpmDownloader osbuild.RpmDownloader

	customSeed *int64
}

// New will create a new manifest generator
func New(reporegistry *reporegistry.RepoRegistry, opts *Options) (*Generator, error) {
	if opts == nil {
		opts = &Options{}
	}
	mg := &Generator{
		reporegistry: reporegistry,

		cacheDir:          opts.Cachedir,
		out:               opts.Output,
		depsolver:         opts.Depsolver,
		containerResolver: opts.ContainerResolver,
		commitResolver:    opts.CommitResolver,
		rpmDownloader:     opts.RpmDownloader,
		sbomWriter:        opts.SBOMWriter,
		customSeed:        opts.CustomSeed,
	}
	if mg.out == nil {
		mg.out = os.Stdout
	}
	if mg.depsolver == nil {
		mg.depsolver = DefaultDepsolver
	}
	if mg.containerResolver == nil {
		mg.containerResolver = DefaultContainerResolver
	}
	if mg.commitResolver == nil {
		mg.commitResolver = DefaultCommitResolver
	}

	return mg, nil
}

// Generate will generate a new manifest for the given distro/imageType/arch
// combination.
func (mg *Generator) Generate(bp *blueprint.Blueprint, dist distro.Distro, imgType distro.ImageType, a distro.Arch, imgOpts *distro.ImageOptions) error {
	if imgOpts == nil {
		imgOpts = &distro.ImageOptions{}
	}

	repos, err := mg.reporegistry.ReposByImageTypeName(dist.Name(), a.Name(), imgType.Name())
	if err != nil {
		return err
	}
	// To support "user" a.k.a. "3rd party" repositories, these
	// will have to be added to the repos with
	// <repo_item>.PackageSets set to the "payload" pipeline names
	// for the given image type, see e.g. distro/rhel/imagetype.go:Manifest()
	preManifest, warnings, err := imgType.Manifest(bp, *imgOpts, repos, mg.customSeed)
	if err != nil {
		return err
	}
	if len(warnings) > 0 {
		// XXX: what can we do here? for things like json output?
		// what are these warnings?
		return fmt.Errorf("warnings during manifest creation: %v", strings.Join(warnings, "\n"))
	}
	depsolved, err := mg.depsolver(mg.cacheDir, preManifest.GetPackageSetChains(), dist, a.Name())
	if err != nil {
		return err
	}
	containerSpecs, err := mg.containerResolver(preManifest.GetContainerSourceSpecs(), a.Name())
	if err != nil {
		return err
	}
	commitSpecs, err := mg.commitResolver(preManifest.GetOSTreeSourceSpecs())
	if err != nil {
		return err
	}
	opts := &manifest.SerializeOptions{
		RpmDownloader: mg.rpmDownloader,
	}
	mf, err := preManifest.Serialize(depsolved, containerSpecs, commitSpecs, opts)
	if err != nil {
		return err
	}
	fmt.Fprintf(mg.out, "%s\n", mf)

	if mg.sbomWriter != nil {
		// XXX: this is very similar to
		// osbuild-composer:jobimpl-osbuild.go, see if code
		// can be shared
		for plName, depsolvedPipeline := range depsolved {
			pipelinePurpose := "unknown"
			switch {
			case slices.Contains(imgType.PayloadPipelines(), plName):
				pipelinePurpose = "image"
			case slices.Contains(imgType.BuildPipelines(), plName):
				pipelinePurpose = "buildroot"
			}
			// XXX: sync with image-builder-cli:build.go name generation - can we have a shared helper?
			imageName := fmt.Sprintf("%s-%s-%s", dist.Name(), imgType.Name(), a.Name())
			sbomDocOutputFilename := fmt.Sprintf("%s.%s-%s.%s", imageName, pipelinePurpose, plName, defaultSBOMExt)

			var buf bytes.Buffer
			enc := json.NewEncoder(&buf)
			if err := enc.Encode(depsolvedPipeline.SBOM.Document); err != nil {
				return err
			}
			if err := mg.sbomWriter(sbomDocOutputFilename, &buf, depsolvedPipeline.SBOM.DocType); err != nil {
				return err
			}
		}
	}

	return nil
}

func xdgCacheHome() (string, error) {
	xdgCacheHome := os.Getenv("XDG_CACHE_HOME")
	if xdgCacheHome != "" {
		return xdgCacheHome, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache"), nil
}

// DefaultDepsolver provides a default implementation for depsolving.
// It should rarely be necessary to use it directly and will be used
// by default by manifestgen (unless overriden)
func DefaultDepsolver(cacheDir string, packageSets map[string][]rpmmd.PackageSet, d distro.Distro, arch string) (map[string]dnfjson.DepsolveResult, error) {
	if cacheDir == "" {
		xdgCacheHomeDir, err := xdgCacheHome()
		if err != nil {
			return nil, err
		}
		cacheDir = filepath.Join(xdgCacheHomeDir, defaultDepsolveCacheDir)
	}

	solver := dnfjson.NewSolver(d.ModulePlatformID(), d.Releasever(), arch, d.Name(), cacheDir)
	depsolvedSets := make(map[string]dnfjson.DepsolveResult)
	for name, pkgSet := range packageSets {
		// Always generate Spdx SBOMs for now, this makes the
		// default depsolve slightly slower but it means we
		// need no extra argument here to select the SBOM
		// type. Once we have more types than Spdx of course
		// we need to add a option to select the type.
		res, err := solver.Depsolve(pkgSet, sbom.StandardTypeSpdx)
		if err != nil {
			return nil, fmt.Errorf("error depsolving: %w", err)
		}
		depsolvedSets[name] = *res
	}
	return depsolvedSets, nil
}

func resolveContainers(containers []container.SourceSpec, archName string) ([]container.Spec, error) {
	resolver := container.NewResolver(archName)

	for _, c := range containers {
		resolver.Add(c)
	}

	return resolver.Finish()
}

// DefaultContainersResolve provides a default implementation for
// container resolving.
// It should rarely be necessary to use it directly and will be used
// by default by manifestgen (unless overriden)
func DefaultContainerResolver(containerSources map[string][]container.SourceSpec, archName string) (map[string][]container.Spec, error) {
	containerSpecs := make(map[string][]container.Spec, len(containerSources))
	for plName, sourceSpecs := range containerSources {
		specs, err := resolveContainers(sourceSpecs, archName)
		if err != nil {
			return nil, fmt.Errorf("error container resolving: %w", err)
		}
		containerSpecs[plName] = specs
	}
	return containerSpecs, nil
}

// DefaultCommitResolver provides a default implementation for
// ostree commit resolving.
// It should rarely be necessary to use it directly and will be used
// by default by manifestgen (unless overriden)
func DefaultCommitResolver(commitSources map[string][]ostree.SourceSpec) (map[string][]ostree.CommitSpec, error) {
	commits := make(map[string][]ostree.CommitSpec, len(commitSources))
	for name, commitSources := range commitSources {
		commitSpecs := make([]ostree.CommitSpec, len(commitSources))
		for idx, commitSource := range commitSources {
			var err error
			commitSpecs[idx], err = ostree.Resolve(commitSource)
			if err != nil {
				return nil, fmt.Errorf("error ostree commit resolving: %w", err)
			}
		}
		commits[name] = commitSpecs
	}
	return commits, nil
}

type (
	DepsolveFunc func(cacheDir string, packageSets map[string][]rpmmd.PackageSet, d distro.Distro, arch string) (map[string]dnfjson.DepsolveResult, error)

	ContainerResolverFunc func(containerSources map[string][]container.SourceSpec, archName string) (map[string][]container.Spec, error)

	CommitResolverFunc func(commitSources map[string][]ostree.SourceSpec) (map[string][]ostree.CommitSpec, error)

	SBOMWriterFunc func(filename string, content io.Reader, docType sbom.StandardType) error
)
