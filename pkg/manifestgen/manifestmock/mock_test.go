package manifestmock_test

import (
	"slices"
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
	result := manifestmock.Depsolve(nil, "x86_64")
	assert.Equal(t, map[string]depsolvednf.DepsolveResult{}, result)

	result = manifestmock.Depsolve(map[string][]rpmmd.PackageSet{}, "x86_64")
	assert.Equal(t, map[string]depsolvednf.DepsolveResult{}, result)
}

func TestDepsolve_Smoke(t *testing.T) {
	baseRepos := []rpmmd.RepoConfig{
		{
			Name:     "baseos",
			BaseURLs: []string{"https://example.com/baseos"},
		},
		{
			Name:     "appstream",
			BaseURLs: []string{"https://example.com/appstream"},
		},
	}
	userRepos := []rpmmd.RepoConfig{
		{
			Name:     "user",
			BaseURLs: []string{"https://example.com/user"},
		},
	}
	allRepos := slices.Concat(baseRepos, userRepos)

	packageSets := map[string][]rpmmd.PackageSet{
		"build": {
			{
				Include:         []string{"build-inc1", "dnf"},
				Exclude:         []string{"build-exc1"},
				Repositories:    baseRepos,
				InstallWeakDeps: true,
			},
		},
		"os": {
			{
				Include:         []string{"os-inc1", "dnf"},
				Exclude:         []string{"os-exc1"},
				Repositories:    baseRepos,
				InstallWeakDeps: true,
			},
			{
				Include:         []string{"os-inc2", "dnf"},
				Exclude:         []string{"os-exc2"},
				Repositories:    allRepos,
				InstallWeakDeps: false,
			},
		},
	}

	arch := "x86_64"
	result := manifestmock.Depsolve(packageSets, arch)
	assert.Equal(t, map[string]depsolvednf.DepsolveResult{
		"build": depsolvednf.DepsolveResult{
			Packages: rpmmd.PackageList{
				{
					Name:            "build-inc1",
					Version:         "1",
					Release:         "3.pkgset~build^trans~0",
					Arch:            "x86_64",
					RemoteLocations: []string{"https://example.com/repo/packages/build-inc1-0:1-3.pkgset~build^trans~0.x86_64.rpm"},
					Checksum:        rpmmd.Checksum{Type: "sha256", Value: "efd4f3331ab734995ccbd2fa9d16ee989b17b8160e7f79baf23118b19849c027"},
				},
				{
					Name:            "dnf",
					Version:         "3",
					Release:         "7.pkgset~build^trans~0",
					Arch:            "x86_64",
					RemoteLocations: []string{"https://example.com/repo/packages/dnf-0:3-7.pkgset~build^trans~0.x86_64.rpm"},
					Checksum:        rpmmd.Checksum{Type: "sha256", Value: "ba61ee16bedef613c9779ecc5d40be8304ece49473923c828082ab5c48dcf8ee"},
				},
				{
					Name:            "exclude:build-exc1",
					Version:         "1",
					Release:         "8.pkgset~build^trans~0",
					Arch:            "x86_64",
					RemoteLocations: []string{"https://example.com/repo/packages/exclude:build-exc1-0:1-8.pkgset~build^trans~0.x86_64.rpm"},
					Checksum:        rpmmd.Checksum{Type: "sha256", Value: "75dacb0c543a006c4ba5f339dbbd2e53006a7f918079b4f5e5fdb683aa395ab7"},
				},
				{
					Name:            "https://example.com/passed-arch:x86_64/passed-repo:/appstream",
					Version:         "2",
					Release:         "1.fk1",
					Arch:            "x86_64",
					RemoteLocations: []string{"https://example.com/passed-arch:x86_64/passed-repo:/appstream"},
					Checksum:        rpmmd.Checksum{Type: "sha256", Value: "e795634c003b026414b3299574c7b847d9168280e8d2bbdbe3e881cbed194c9d"},
				},
				{
					Name:            "https://example.com/passed-arch:x86_64/passed-repo:/baseos",
					Version:         "6",
					Release:         "0.fk1",
					Arch:            "x86_64",
					RemoteLocations: []string{"https://example.com/passed-arch:x86_64/passed-repo:/baseos"},
					Checksum:        rpmmd.Checksum{Type: "sha256", Value: "3cdb35e9d0c01e60bc3362af2544e8b46429c3b1d5177579a0bec588ac65e707"},
				},
				{
					Name:            "pkgset:build_trans:0_repos:baseos+appstream+weak",
					Version:         "7",
					Release:         "7.fk1",
					Arch:            "x86_64",
					RemoteLocations: []string{"https://example.com/repo/packages/pkgset:build_trans:0_repos:baseos+appstream+weak-0:7-7.fk1.x86_64.rpm"},
					Checksum:        rpmmd.Checksum{Type: "sha256", Value: "aa52cccdc067ec6796580b2cf1a0a6c34b7e996fbd8740c9cff8f0e82c6f4646"},
				},
			},
			Transactions: depsolvednf.TransactionList{
				{
					{
						Name:            "build-inc1",
						Version:         "1",
						Release:         "3.pkgset~build^trans~0",
						Arch:            "x86_64",
						RemoteLocations: []string{"https://example.com/repo/packages/build-inc1-0:1-3.pkgset~build^trans~0.x86_64.rpm"},
						Checksum:        rpmmd.Checksum{Type: "sha256", Value: "efd4f3331ab734995ccbd2fa9d16ee989b17b8160e7f79baf23118b19849c027"},
					},
					{
						Name:            "dnf",
						Version:         "3",
						Release:         "7.pkgset~build^trans~0",
						Arch:            "x86_64",
						RemoteLocations: []string{"https://example.com/repo/packages/dnf-0:3-7.pkgset~build^trans~0.x86_64.rpm"},
						Checksum:        rpmmd.Checksum{Type: "sha256", Value: "ba61ee16bedef613c9779ecc5d40be8304ece49473923c828082ab5c48dcf8ee"},
					},
					{
						Name:            "exclude:build-exc1",
						Version:         "1",
						Release:         "8.pkgset~build^trans~0",
						Arch:            "x86_64",
						RemoteLocations: []string{"https://example.com/repo/packages/exclude:build-exc1-0:1-8.pkgset~build^trans~0.x86_64.rpm"},
						Checksum:        rpmmd.Checksum{Type: "sha256", Value: "75dacb0c543a006c4ba5f339dbbd2e53006a7f918079b4f5e5fdb683aa395ab7"},
					},
					{
						Name:            "https://example.com/passed-arch:x86_64/passed-repo:/appstream",
						Version:         "2",
						Release:         "1.fk1",
						Arch:            "x86_64",
						RemoteLocations: []string{"https://example.com/passed-arch:x86_64/passed-repo:/appstream"},
						Checksum:        rpmmd.Checksum{Type: "sha256", Value: "e795634c003b026414b3299574c7b847d9168280e8d2bbdbe3e881cbed194c9d"},
					},
					{
						Name:            "https://example.com/passed-arch:x86_64/passed-repo:/baseos",
						Version:         "6",
						Release:         "0.fk1",
						Arch:            "x86_64",
						RemoteLocations: []string{"https://example.com/passed-arch:x86_64/passed-repo:/baseos"},
						Checksum:        rpmmd.Checksum{Type: "sha256", Value: "3cdb35e9d0c01e60bc3362af2544e8b46429c3b1d5177579a0bec588ac65e707"},
					},
					{
						Name:            "pkgset:build_trans:0_repos:baseos+appstream+weak",
						Version:         "7",
						Release:         "7.fk1",
						Arch:            "x86_64",
						RemoteLocations: []string{"https://example.com/repo/packages/pkgset:build_trans:0_repos:baseos+appstream+weak-0:7-7.fk1.x86_64.rpm"},
						Checksum:        rpmmd.Checksum{Type: "sha256", Value: "aa52cccdc067ec6796580b2cf1a0a6c34b7e996fbd8740c9cff8f0e82c6f4646"},
					},
				},
			},
			Repos: baseRepos,
		},
		"os": depsolvednf.DepsolveResult{
			Packages: rpmmd.PackageList{
				{
					Name:            "dnf",
					Version:         "0",
					Release:         "5.pkgset~os^trans~0",
					Arch:            "x86_64",
					RemoteLocations: []string{"https://example.com/repo/packages/dnf-0:0-5.pkgset~os^trans~0.x86_64.rpm"},
					Checksum:        rpmmd.Checksum{Type: "sha256", Value: "22c603fbf9285e567b76ea505f365da7d013cc48174b494e40a5032e153291ac"},
				},
				{
					Name:            "dnf",
					Version:         "3",
					Release:         "1.pkgset~os^trans~1",
					Arch:            "x86_64",
					RemoteLocations: []string{"https://example.com/repo/packages/dnf-0:3-1.pkgset~os^trans~1.x86_64.rpm"},
					Checksum:        rpmmd.Checksum{Type: "sha256", Value: "5d182a7a8683bbc8e3f64a2d9d9547de758e7ff9d98cdebdcac97c62d4912602"},
				},
				{
					Name:            "exclude:os-exc1",
					Version:         "1",
					Release:         "5.pkgset~os^trans~0",
					Arch:            "x86_64",
					RemoteLocations: []string{"https://example.com/repo/packages/exclude:os-exc1-0:1-5.pkgset~os^trans~0.x86_64.rpm"},
					Checksum:        rpmmd.Checksum{Type: "sha256", Value: "d292792edfe3ffd70bd96aae58368cf64607fd8ca5d989df23138bacba271a8d"},
				},
				{
					Name:            "exclude:os-exc2",
					Version:         "1",
					Release:         "2.pkgset~os^trans~1",
					Arch:            "x86_64",
					RemoteLocations: []string{"https://example.com/repo/packages/exclude:os-exc2-0:1-2.pkgset~os^trans~1.x86_64.rpm"},
					Checksum:        rpmmd.Checksum{Type: "sha256", Value: "780cae4f0a0ccba5ba6ecb647e03d7dd1d38cd1c5843e67b00432d5c478d6018"},
				},
				{
					Name:            "https://example.com/passed-arch:x86_64/passed-repo:/appstream",
					Version:         "2",
					Release:         "1.fk1",
					Arch:            "x86_64",
					RemoteLocations: []string{"https://example.com/passed-arch:x86_64/passed-repo:/appstream"},
					Checksum:        rpmmd.Checksum{Type: "sha256", Value: "e795634c003b026414b3299574c7b847d9168280e8d2bbdbe3e881cbed194c9d"},
				},
				{
					Name:            "https://example.com/passed-arch:x86_64/passed-repo:/baseos",
					Version:         "6",
					Release:         "0.fk1",
					Arch:            "x86_64",
					RemoteLocations: []string{"https://example.com/passed-arch:x86_64/passed-repo:/baseos"},
					Checksum:        rpmmd.Checksum{Type: "sha256", Value: "3cdb35e9d0c01e60bc3362af2544e8b46429c3b1d5177579a0bec588ac65e707"},
				},
				{
					Name:            "https://example.com/passed-arch:x86_64/passed-repo:/user",
					Version:         "3",
					Release:         "0.fk1",
					Arch:            "x86_64",
					RemoteLocations: []string{"https://example.com/passed-arch:x86_64/passed-repo:/user"},
					Checksum:        rpmmd.Checksum{Type: "sha256", Value: "0c676fb4e895762dc80cd8abadec73e49970317f281ad5b94b93df9afdcad4e9"},
				},
				{
					Name:            "os-inc1",
					Version:         "3",
					Release:         "7.pkgset~os^trans~0",
					Arch:            "x86_64",
					RemoteLocations: []string{"https://example.com/repo/packages/os-inc1-0:3-7.pkgset~os^trans~0.x86_64.rpm"},
					Checksum:        rpmmd.Checksum{Type: "sha256", Value: "540bb70a104b25dce75c51363eadbdb9c0ed0387467fa2ee28fec5c3103b7f0a"},
				},
				{
					Name:            "os-inc2",
					Version:         "2",
					Release:         "0.pkgset~os^trans~1",
					Arch:            "x86_64",
					RemoteLocations: []string{"https://example.com/repo/packages/os-inc2-0:2-0.pkgset~os^trans~1.x86_64.rpm"},
					Checksum:        rpmmd.Checksum{Type: "sha256", Value: "fc429287a10941ecc73ad362fa2e0cf97226613de5206025f92f6fd5da24fd73"},
				},
				{
					Name:            "pkgset:os_trans:0_repos:baseos+appstream+weak",
					Version:         "4",
					Release:         "7.fk1",
					Arch:            "x86_64",
					RemoteLocations: []string{"https://example.com/repo/packages/pkgset:os_trans:0_repos:baseos+appstream+weak-0:4-7.fk1.x86_64.rpm"},
					Checksum:        rpmmd.Checksum{Type: "sha256", Value: "14f5cd42c2f0cf10809965631f90375e9a4a8593b144cfe0e3c27fe83d1ee7f9"},
				},
				{
					Name:            "pkgset:os_trans:1_repos:baseos+appstream+user",
					Version:         "0",
					Release:         "3.fk1",
					Arch:            "x86_64",
					RemoteLocations: []string{"https://example.com/repo/packages/pkgset:os_trans:1_repos:baseos+appstream+user-0:0-3.fk1.x86_64.rpm"},
					Checksum:        rpmmd.Checksum{Type: "sha256", Value: "c9d715c8576f7f5037c9a1f139aacc7a7208631eff8eba40c24fc18dd5fe44a2"},
				},
			},
			Transactions: depsolvednf.TransactionList{
				{
					{
						Name:            "dnf",
						Version:         "0",
						Release:         "5.pkgset~os^trans~0",
						Arch:            "x86_64",
						RemoteLocations: []string{"https://example.com/repo/packages/dnf-0:0-5.pkgset~os^trans~0.x86_64.rpm"},
						Checksum:        rpmmd.Checksum{Type: "sha256", Value: "22c603fbf9285e567b76ea505f365da7d013cc48174b494e40a5032e153291ac"},
					},
					{
						Name:            "exclude:os-exc1",
						Version:         "1",
						Release:         "5.pkgset~os^trans~0",
						Arch:            "x86_64",
						RemoteLocations: []string{"https://example.com/repo/packages/exclude:os-exc1-0:1-5.pkgset~os^trans~0.x86_64.rpm"},
						Checksum:        rpmmd.Checksum{Type: "sha256", Value: "d292792edfe3ffd70bd96aae58368cf64607fd8ca5d989df23138bacba271a8d"},
					},
					{
						Name:            "https://example.com/passed-arch:x86_64/passed-repo:/appstream",
						Version:         "2",
						Release:         "1.fk1",
						Arch:            "x86_64",
						RemoteLocations: []string{"https://example.com/passed-arch:x86_64/passed-repo:/appstream"},
						Checksum:        rpmmd.Checksum{Type: "sha256", Value: "e795634c003b026414b3299574c7b847d9168280e8d2bbdbe3e881cbed194c9d"},
					},
					{
						Name:            "https://example.com/passed-arch:x86_64/passed-repo:/baseos",
						Version:         "6",
						Release:         "0.fk1",
						Arch:            "x86_64",
						RemoteLocations: []string{"https://example.com/passed-arch:x86_64/passed-repo:/baseos"},
						Checksum:        rpmmd.Checksum{Type: "sha256", Value: "3cdb35e9d0c01e60bc3362af2544e8b46429c3b1d5177579a0bec588ac65e707"},
					},
					{
						Name:            "os-inc1",
						Version:         "3",
						Release:         "7.pkgset~os^trans~0",
						Arch:            "x86_64",
						RemoteLocations: []string{"https://example.com/repo/packages/os-inc1-0:3-7.pkgset~os^trans~0.x86_64.rpm"},
						Checksum:        rpmmd.Checksum{Type: "sha256", Value: "540bb70a104b25dce75c51363eadbdb9c0ed0387467fa2ee28fec5c3103b7f0a"},
					},
					{
						Name:            "pkgset:os_trans:0_repos:baseos+appstream+weak",
						Version:         "4",
						Release:         "7.fk1",
						Arch:            "x86_64",
						RemoteLocations: []string{"https://example.com/repo/packages/pkgset:os_trans:0_repos:baseos+appstream+weak-0:4-7.fk1.x86_64.rpm"},
						Checksum:        rpmmd.Checksum{Type: "sha256", Value: "14f5cd42c2f0cf10809965631f90375e9a4a8593b144cfe0e3c27fe83d1ee7f9"},
					},
				},
				{
					{
						Name:            "dnf",
						Version:         "3",
						Release:         "1.pkgset~os^trans~1",
						Arch:            "x86_64",
						RemoteLocations: []string{"https://example.com/repo/packages/dnf-0:3-1.pkgset~os^trans~1.x86_64.rpm"},
						Checksum:        rpmmd.Checksum{Type: "sha256", Value: "5d182a7a8683bbc8e3f64a2d9d9547de758e7ff9d98cdebdcac97c62d4912602"},
					},
					{
						Name:            "exclude:os-exc2",
						Version:         "1",
						Release:         "2.pkgset~os^trans~1",
						Arch:            "x86_64",
						RemoteLocations: []string{"https://example.com/repo/packages/exclude:os-exc2-0:1-2.pkgset~os^trans~1.x86_64.rpm"},
						Checksum:        rpmmd.Checksum{Type: "sha256", Value: "780cae4f0a0ccba5ba6ecb647e03d7dd1d38cd1c5843e67b00432d5c478d6018"},
					},
					{
						Name:            "https://example.com/passed-arch:x86_64/passed-repo:/user",
						Version:         "3",
						Release:         "0.fk1",
						Arch:            "x86_64",
						RemoteLocations: []string{"https://example.com/passed-arch:x86_64/passed-repo:/user"},
						Checksum:        rpmmd.Checksum{Type: "sha256", Value: "0c676fb4e895762dc80cd8abadec73e49970317f281ad5b94b93df9afdcad4e9"},
					},
					{
						Name:            "os-inc2",
						Version:         "2",
						Release:         "0.pkgset~os^trans~1",
						Arch:            "x86_64",
						RemoteLocations: []string{"https://example.com/repo/packages/os-inc2-0:2-0.pkgset~os^trans~1.x86_64.rpm"},
						Checksum:        rpmmd.Checksum{Type: "sha256", Value: "fc429287a10941ecc73ad362fa2e0cf97226613de5206025f92f6fd5da24fd73"},
					},
					{
						Name:            "pkgset:os_trans:1_repos:baseos+appstream+user",
						Version:         "0",
						Release:         "3.fk1",
						Arch:            "x86_64",
						RemoteLocations: []string{"https://example.com/repo/packages/pkgset:os_trans:1_repos:baseos+appstream+user-0:0-3.fk1.x86_64.rpm"},
						Checksum:        rpmmd.Checksum{Type: "sha256", Value: "c9d715c8576f7f5037c9a1f139aacc7a7208631eff8eba40c24fc18dd5fe44a2"},
					},
				},
			},
			Repos: allRepos,
		},
	}, result)
}
