package rpmmd

import (
	"fmt"
	"sort"
	"time"

	"github.com/gobwas/glob"
)

type PackageList []Package

type Package struct {
	Name        string
	Summary     string
	Description string
	URL         string
	Epoch       uint
	Version     string
	Release     string
	Arch        string
	BuildTime   time.Time
	License     string
}

// The inputs to depsolve, a set of packages to include and a set of packages
// to exclude. The Repositories are used when depsolving this package set in
// addition to the base repositories.
type PackageSet struct {
	Include         []string
	Exclude         []string
	EnabledModules  []string
	Repositories    []RepoConfig
	InstallWeakDeps bool
}

// Append the Include and Exclude package list from another PackageSet and
// return the result.
func (ps PackageSet) Append(other PackageSet) PackageSet {
	ps.Include = append(ps.Include, other.Include...)
	ps.Exclude = append(ps.Exclude, other.Exclude...)
	ps.EnabledModules = append(ps.EnabledModules, other.EnabledModules...)
	return ps
}

// TODO: the public API of this package should not be reused for serialization.
type PackageSpec struct {
	Name           string `json:"name"`
	Epoch          uint   `json:"epoch"`
	Version        string `json:"version,omitempty"`
	Release        string `json:"release,omitempty"`
	Arch           string `json:"arch,omitempty"`
	RemoteLocation string `json:"remote_location,omitempty"`
	Checksum       string `json:"checksum,omitempty"`
	Secrets        string `json:"secrets,omitempty"`
	CheckGPG       bool   `json:"check_gpg,omitempty"`
	IgnoreSSL      bool   `json:"ignore_ssl,omitempty"`

	Path   string `json:"path,omitempty"`
	RepoID string `json:"repo_id,omitempty"`
}

// GetEVRA returns the package's Epoch:Version-Release.Arch string
func (ps *PackageSpec) GetEVRA() string {
	if ps.Epoch == 0 {
		return fmt.Sprintf("%s-%s.%s", ps.Version, ps.Release, ps.Arch)
	}
	return fmt.Sprintf("%d:%s-%s.%s", ps.Epoch, ps.Version, ps.Release, ps.Arch)
}

// GetNEVRA returns the package's Name-Epoch:Version-Release.Arch string
func (ps *PackageSpec) GetNEVRA() string {
	return fmt.Sprintf("%s-%s", ps.Name, ps.GetEVRA())
}

func GetPackage(pkgs []PackageSpec, packageName string) (PackageSpec, error) {
	for _, pkg := range pkgs {
		if pkg.Name == packageName {
			return pkg, nil
		}
	}

	return PackageSpec{}, fmt.Errorf("package %q not found in the PackageSpec list", packageName)
}

func (packages PackageList) Search(globPatterns ...string) (PackageList, error) {
	var globs []glob.Glob

	for _, globPattern := range globPatterns {
		g, err := glob.Compile(globPattern)
		if err != nil {
			return nil, err
		}

		globs = append(globs, g)
	}

	var foundPackages PackageList

	for _, pkg := range packages {
		for _, g := range globs {
			if g.Match(pkg.Name) {
				foundPackages = append(foundPackages, pkg)
				break
			}
		}
	}

	sort.Slice(packages, func(i, j int) bool {
		return packages[i].Name < packages[j].Name
	})

	return foundPackages, nil
}
