package manifestmock_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/depsolvednf"
	"github.com/osbuild/images/pkg/manifestgen/manifestmock"
	"github.com/osbuild/images/pkg/ostree"
	"github.com/osbuild/images/pkg/rpmmd"
)

func TestResolveContainers_EmptyInpu(t *testing.T) {
	result := manifestmock.ResolveContainers(nil)
	assert.Equal(t, map[string][]container.Spec{}, result)

	result = manifestmock.ResolveContainers(map[string][]container.SourceSpec{})
	assert.Equal(t, map[string][]container.Spec{}, result)
}

func TestResolveContainers_Smoke(t *testing.T) {
	input := map[string][]container.SourceSpec{
		"build": {
			{
				Name:      "Build container",
				Source:    "ghcr.io/ondrejbudai/booc:fedora",
				TLSVerify: common.ToPtr(true),
			},
		},
	}
	result := manifestmock.ResolveContainers(input)
	assert.Equal(t, map[string][]container.Spec{
		"build": []container.Spec{
			{
				Source:     "ghcr.io/ondrejbudai/booc:fedora",
				Digest:     "sha256:df023f283afc154c1374e2335ea4a54a210f1cf0f8fe2af812c239a576577efa",
				ListDigest: "sha256:26c7349a68c3e90dd897c8ca1fff7097b274efc33fed56d858331de3bd01c9d8",
				TLSVerify:  common.ToPtr(true),
				ImageID:    "sha256:2c380abcfa442874be885a28e4f909600c24e5457374cd6671a4dbd74e28ffe7",
				LocalName:  "Build container",
			},
		},
	}, result)
}

func TestResolveCommits_EmptyInput(t *testing.T) {
	result := manifestmock.ResolveCommits(nil)
	assert.Equal(t, map[string][]ostree.CommitSpec{}, result)

	result = manifestmock.ResolveCommits(map[string][]ostree.SourceSpec{})
	assert.Equal(t, map[string][]ostree.CommitSpec{}, result)
}

func TestResolveCommits_Smoke(t *testing.T) {
	input := map[string][]ostree.SourceSpec{
		"pipeline1": {
			{
				Ref: "test/ref",
				URL: "https://example.com/repo",
			},
		},
	}
	result := manifestmock.ResolveCommits(input)
	assert.Equal(t, map[string][]ostree.CommitSpec{
		"pipeline1": []ostree.CommitSpec{
			{
				Ref:      "test/ref",
				URL:      "https://example.com/repo",
				Checksum: "b9b3034a43bf9c404fce8c7713f7e115a10a429d67afc55076b911878ca92615",
			},
		},
	}, result)
}

func TestDepsolve_EmptyInput(t *testing.T) {
	result := manifestmock.Depsolve(nil, nil, "x86_64")
	assert.Equal(t, map[string]depsolvednf.DepsolveResult{}, result)

	result = manifestmock.Depsolve(map[string][]rpmmd.PackageSet{}, []rpmmd.RepoConfig{}, "x86_64")
	assert.Equal(t, map[string]depsolvednf.DepsolveResult{}, result)
}

func TestDepsolve_Smoke(t *testing.T) {
	packageSets := map[string][]rpmmd.PackageSet{
		"build": {
			{
				Include:         []string{"inc1"},
				Exclude:         []string{"exc1"},
				Repositories:    []rpmmd.RepoConfig{{Name: "repo1"}},
				InstallWeakDeps: true,
			},
		},
	}
	repos := []rpmmd.RepoConfig{
		{
			Name:     "repo1",
			BaseURLs: []string{"https://example.com/foo"},
		},
	}
	arch := "x86_64"
	result := manifestmock.Depsolve(packageSets, repos, arch)
	assert.Equal(t, map[string]depsolvednf.DepsolveResult{
		"build": depsolvednf.DepsolveResult{
			Packages: rpmmd.PackageList{
				{
					Name:            "inc1",
					Version:         "2",
					Release:         "3.pkgset~build^trans~0",
					Arch:            "x86_64",
					RemoteLocations: []string{"https://example.com/repo/packages/inc1-0:2-3.pkgset~build^trans~0.x86_64.rpm"},
					Checksum:        rpmmd.Checksum{Type: "sha256", Value: "4060c918d6fd91bffe885125b1aa1d6a691e871d6f3e924ccecdbacca899c111"},
				},
				{
					Name:            "exclude:exc1",
					Version:         "2",
					Release:         "2.pkgset~build^trans~0",
					Arch:            "x86_64",
					RemoteLocations: []string{"https://example.com/repo/packages/exclude:exc1-0:2-2.pkgset~build^trans~0.x86_64.rpm"},
					Checksum:        rpmmd.Checksum{Type: "sha256", Value: "ee11e9be701d637dc94d627456f0b61635210035d9f59ad7a0a6f089d496ca8e"},
				},
				{
					Name:            "pkgset:build_trans:0_repos:repo1+weak",
					Version:         "7",
					Release:         "4.fk1",
					Arch:            "x86_64",
					RemoteLocations: []string{"https://example.com/repo/packages/pkgset:build_trans:0_repos:repo1+weak-0:7-4.fk1.x86_64.rpm"},
					Checksum:        rpmmd.Checksum{Type: "sha256", Value: "a16f5db781c832938af5d8c0c7fc91cba80e8e269324d8382c22213b1795526f"},
				},
				{
					Name:            "https://example.com/passed-arch:x86_64/passed-repo:/foo",
					Version:         "0",
					Release:         "6.fk1",
					Arch:            "x86_64",
					RemoteLocations: []string{"https://example.com/passed-arch:x86_64/passed-repo:/foo"},
					Checksum:        rpmmd.Checksum{Type: "sha256", Value: "63c9a60a4f279e4825c170c3cd893560716635f84a9a71fb262a6206d59ca74d"},
				},
			},
			Repos: []rpmmd.RepoConfig{
				{
					Name:     "repo1",
					BaseURLs: []string{"https://example.com/foo"},
				},
			},
		},
	}, result)
}
