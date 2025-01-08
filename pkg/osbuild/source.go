package osbuild

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/ostree"
	"github.com/osbuild/images/pkg/rpmmd"
)

// A Sources map contains all the sources made available to an osbuild run
type Sources map[string]Source

// Source specifies the operations of a given source-type.
type Source interface {
	isSource()
}

type SourceOptions interface {
	isSourceOptions()
}

type rawSources map[string]json.RawMessage

// UnmarshalJSON unmarshals JSON into a Source object. Each type of source has
// a custom unmarshaller for its options, selected based on the source name.
func (sources *Sources) UnmarshalJSON(data []byte) error {
	var rawSources rawSources
	err := json.Unmarshal(data, &rawSources)
	if err != nil {
		return err
	}
	*sources = make(map[string]Source)
	for name, rawSource := range rawSources {
		var source Source
		switch name {
		case "org.osbuild.curl":
			source = new(CurlSource)
		case "org.osbuild.inline":
			source = new(InlineSource)
		case "org.osbuild.ostree":
			source = new(OSTreeSource)
		default:
			return errors.New("unexpected source name: " + name)
		}
		err = json.Unmarshal(rawSource, source)
		if err != nil {
			return err
		}
		(*sources)[name] = source
	}

	return nil
}

func addPackagesCurl(sources Sources, packages []rpmmd.PackageSpec) error {
	curl := NewCurlSource()
	for _, pkg := range packages {
		err := curl.AddPackage(pkg)
		if err != nil {
			return err
		}
	}
	sources["org.osbuild.curl"] = curl
	return nil
}

func addPackagesLibrepo(sources Sources, packages []rpmmd.PackageSpec, rpmRepos map[string][]rpmmd.RepoConfig) error {
	librepo := NewLibrepoSource()
	for _, pkg := range packages {
		err := librepo.AddPackage(pkg, rpmRepos)
		if err != nil {
			return err
		}
	}
	sources["org.osbuild.librepo"] = librepo
	return nil
}

// RpmDownloader specifies what backend to use for rpm downloads
// Note that the librepo backend requires a newer osbuild.
type RpmDownloader uint64

const (
	RpmDownloaderCurl    = iota
	RpmDownloaderLibrepo = iota
)

func GenSources(packages []rpmmd.PackageSpec, ostreeCommits []ostree.CommitSpec, inlineData []string, containers []container.Spec, rpmRepos map[string][]rpmmd.RepoConfig, rpmDownloader RpmDownloader) (Sources, error) {
	// The signature of this functionis already relatively long,
	// if we need to add more options, refactor into "struct
	// Inputs" (rpm,ostree,etc) and "struct Options"
	// (rpmDownloader)
	sources := Sources{}

	// collect rpm package sources
	if len(packages) > 0 {
		var err error
		switch rpmDownloader {
		case RpmDownloaderCurl:
			err = addPackagesCurl(sources, packages)
		case RpmDownloaderLibrepo:
			err = addPackagesLibrepo(sources, packages, rpmRepos)
		default:
			err = fmt.Errorf("unknown rpm downloader %v", rpmDownloader)
		}
		if err != nil {
			return nil, err
		}
	}

	// collect ostree commit sources
	if len(ostreeCommits) > 0 {
		ostree := NewOSTreeSource()
		for _, commit := range ostreeCommits {
			ostree.AddItem(commit)
		}
		if len(ostree.Items) > 0 {
			sources["org.osbuild.ostree"] = ostree
		}
	}

	// collect inline data sources
	if len(inlineData) > 0 {
		ils := NewInlineSource()
		for _, data := range inlineData {
			ils.AddItem(data)
		}

		sources["org.osbuild.inline"] = ils
	}

	// collect skopeo and local container sources
	if len(containers) > 0 {
		skopeo := NewSkopeoSource()
		skopeoIndex := NewSkopeoIndexSource()
		localContainers := NewContainersStorageSource()
		for _, c := range containers {
			if c.LocalStorage {
				localContainers.AddItem(c.ImageID)
			} else {
				skopeo.AddItem(c.Source, c.Digest, c.ImageID, c.TLSVerify)
				// if we have a list digest, add a skopeo-index source as well
				if c.ListDigest != "" {
					skopeoIndex.AddItem(c.Source, c.ListDigest, c.TLSVerify)
				}
			}
		}
		if len(skopeo.Items) > 0 {
			sources["org.osbuild.skopeo"] = skopeo
		}
		if len(skopeoIndex.Items) > 0 {
			sources["org.osbuild.skopeo-index"] = skopeoIndex
		}
		if len(localContainers.Items) > 0 {
			sources["org.osbuild.containers-storage"] = localContainers
		}
	}

	return sources, nil
}
