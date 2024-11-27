package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/dnfjson"
	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/osbuild/images/pkg/sbom"
)

// XXX: duplicated from cmd/build/main.go:depsolve (and probably more places)
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

type genManifestOptions struct {
	OutputFilename string
}

func outputManifest(out io.Writer, distroName, imgTypeStr, archStr string, opts *genManifestOptions) error {
	if opts == nil {
		opts = &genManifestOptions{}
	}

	filterResult, err := getOneImage(distroName, imgTypeStr, archStr)
	if err != nil {
		return err
	}
	dist := filterResult.Distro
	imgType := filterResult.ImgType

	reporeg, err := newRepoRegistry()
	if err != nil {
		return err
	}
	repos, err := reporeg.ReposByImageTypeName(distroName, archStr, imgTypeStr)
	if err != nil {
		return err
	}

	var bp blueprint.Blueprint
	imgOpts := distro.ImageOptions{
		OutputFilename: opts.OutputFilename,
	}
	preManifest, warnings, err := imgType.Manifest(&bp, imgOpts, repos, 0)
	if err != nil {
		return err
	}
	if len(warnings) > 0 {
		// XXX: what can we do here? for things like json output?
		// what are these warnings?
		return fmt.Errorf("warnings during manifest creation: %v", strings.Join(warnings, "\n"))
	}
	// XXX: cleanup, use shared dir,etc
	cacheDir, err := os.MkdirTemp("", "depsolve")
	if err != nil {
		return err
	}
	packageSpecs, _, err := depsolve(cacheDir, preManifest.GetPackageSetChains(), dist, archStr)
	if err != nil {
		return err
	}
	if packageSpecs == nil {
		return fmt.Errorf("depsolve did not return any packages")
	}
	// XXX: support commit/container specs
	mf, err := preManifest.Serialize(packageSpecs, nil, nil, nil)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "%s\n", mf)

	return nil
}
