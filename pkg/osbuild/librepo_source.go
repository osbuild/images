package osbuild

import (
	"fmt"
	"regexp"

	"github.com/osbuild/images/pkg/rpmmd"
)

var librepoDigestPattern = regexp.MustCompile(`(sha256|sha384|sha512):[0-9a-f]{32,128}`)

type LibrepoSource struct {
	Items   map[string]*LibrepoSourceItem `json:"items"`
	Options *LibrepoSourceOptions         `json:"options"`
}

func (LibrepoSource) isSource() {}

func NewLibrepoSource() *LibrepoSource {
	return &LibrepoSource{
		Items: make(map[string]*LibrepoSourceItem),
		Options: &LibrepoSourceOptions{
			Mirrors: make(map[string]*LibrepoSourceMirror),
		},
	}
}

type LibrepoSourceItem struct {
	Path     string `json:"path"`
	MirrorID string `json:"mirror"`
}

func findRepoById(repos map[string][]rpmmd.RepoConfig, repoID string) *rpmmd.RepoConfig {
	for _, repos := range repos {
		for _, repo := range repos {
			if repo.Id == repoID {
				return &repo
			}
		}
	}
	return nil
}

func mirrorFromRepo(repo *rpmmd.RepoConfig) (*LibrepoSourceMirror, error) {
	// XXX: add support for secrets
	switch {
	case repo.Metalink != "":
		return &LibrepoSourceMirror{
			URL:  repo.Metalink,
			Type: "metalink",
		}, nil
	case repo.MirrorList != "":
		return &LibrepoSourceMirror{
			URL:  repo.MirrorList,
			Type: "mirrorlist",
		}, nil
	case len(repo.BaseURLs) > 0:
		return &LibrepoSourceMirror{
			// XXX: should we pick a random one instead?
			URL:  repo.BaseURLs[0],
			Type: "baseurl",
		}, nil
	}

	return nil, fmt.Errorf("cannot find metalink, mirrorlist or baseurl for %+v", repo)
}

func (source *LibrepoSource) AddPackage(pkg rpmmd.PackageSpec, repos map[string][]rpmmd.RepoConfig) error {
	pkgRepo := findRepoById(repos, pkg.RepoID)
	if pkgRepo == nil {
		return fmt.Errorf("cannot find repo-id %v for %v in %+v", pkg.RepoID, pkg.Name, repos)
	}
	if _, ok := source.Options.Mirrors[pkgRepo.Id]; !ok {
		mirror, err := mirrorFromRepo(pkgRepo)
		if err != nil {
			return err
		}
		source.Options.Mirrors[pkgRepo.Id] = mirror
	}
	mirror := source.Options.Mirrors[pkgRepo.Id]
	// XXX: should we error here if one package requests IgnoreSSL
	// and one does not for the same mirror?
	if pkg.IgnoreSSL {
		mirror.Insecure = true
	}
	if pkg.Secrets == "org.osbuild.rhsm" {
		mirror.Secrets = &URLSecrets{
			Name: "org.osbuild.rhsm",
		}
	} else if pkg.Secrets == "org.osbuild.mtls" {
		mirror.Secrets = &URLSecrets{
			Name: "org.osbuild.mtls",
		}
	}

	item := &LibrepoSourceItem{
		Path:     pkg.Path,
		MirrorID: pkgRepo.Id,
	}
	source.Items[pkg.Checksum] = item
	return nil
}

type LibrepoSourceOptions struct {
	Mirrors map[string]*LibrepoSourceMirror `json:"mirrors"`
}

type LibrepoSourceMirror struct {
	URL  string `json:"url"`
	Type string `json:"type"`

	Insecure bool        `json:"insecure,omitempty"`
	Secrets  *URLSecrets `json:"secrets,omitempty"`

	// XXX: should we expose those? if so we need a way to set them,
	// current this is done in manifest.GenSources which cannot take
	// options.
	// MaxParallels  *int `json:"max-parallels,omitempty"`
	// FastestMirror bool `json:"fastest-mirror,omitempty"`
}
