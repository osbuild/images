package depsolvednf

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/internal/mocks/rpmrepo"
	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/osbuild/images/pkg/sbom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var forceDNF = flag.Bool("force-dnf", false, "force dnf testing, making them fail instead of skip if dnf isn't installed")

func TestDepsolver(t *testing.T) {
	if !*forceDNF {
		// dnf tests aren't forced: skip them if the dnf sniff check fails
		if findDepsolveDnf() == "" {
			t.Skip("Test needs an installed osbuild-depsolve-dnf")
		}
	}

	s := rpmrepo.NewTestServer()
	defer s.Close()

	type testCase struct {
		packages [][]string
		repos    []rpmmd.RepoConfig
		rootDir  string
		sbomType sbom.StandardType
		err      bool
		expMsg   string
	}

	tmpdir := t.TempDir()
	solver := NewSolver("platform:el9", "9", "x86_64", "rhel9.0", tmpdir)

	rootDir := t.TempDir()
	reposDir := filepath.Join(rootDir, "etc", "yum.repos.d")
	require.NoError(t, os.MkdirAll(reposDir, 0777))
	s.WriteConfig(filepath.Join(reposDir, "test.repo"))

	testCases := map[string]testCase{
		"flat": {
			packages: [][]string{{"kernel", "vim-minimal", "tmux", "zsh"}},
			repos:    []rpmmd.RepoConfig{s.RepoConfig},
			err:      false,
		},
		"chain": {
			// chain depsolve of the same packages in order should produce the same result (at least in this case)
			packages: [][]string{{"kernel"}, {"vim-minimal", "tmux", "zsh"}},
			repos:    []rpmmd.RepoConfig{s.RepoConfig},
			err:      false,
		},
		"bad-flat": {
			packages: [][]string{{"this-package-does-not-exist"}},
			repos:    []rpmmd.RepoConfig{s.RepoConfig},
			err:      true,
			expMsg:   "this-package-does-not-exist",
		},
		"bad-chain": {
			packages: [][]string{{"kernel"}, {"this-package-does-not-exist"}},
			repos:    []rpmmd.RepoConfig{s.RepoConfig},
			err:      true,
			expMsg:   "this-package-does-not-exist",
		},
		"bad-chain-part-deux": {
			packages: [][]string{{"this-package-does-not-exist"}, {"vim-minimal", "tmux", "zsh"}},
			repos:    []rpmmd.RepoConfig{s.RepoConfig},
			err:      true,
			expMsg:   "this-package-does-not-exist",
		},
		"flat+dir": {
			packages: [][]string{{"kernel", "vim-minimal", "tmux", "zsh"}},
			rootDir:  rootDir,
			err:      false,
		},
		"chain+dir": {
			packages: [][]string{{"kernel"}, {"vim-minimal", "tmux", "zsh"}},
			rootDir:  rootDir,
			err:      false,
		},
		"bad-flat+dir": {
			packages: [][]string{{"this-package-does-not-exist"}},
			rootDir:  rootDir,
			err:      true,
			expMsg:   "this-package-does-not-exist",
		},
		"bad-chain+dir": {
			packages: [][]string{{"kernel"}, {"this-package-does-not-exist"}},
			rootDir:  rootDir,
			err:      true,
			expMsg:   "this-package-does-not-exist",
		},
		"bad-chain-part-deux+dir": {
			packages: [][]string{{"this-package-does-not-exist"}, {"vim-minimal", "tmux", "zsh"}},
			rootDir:  rootDir,
			err:      true,
			expMsg:   "this-package-does-not-exist",
		},
		"chain-with-sbom": {
			// chain depsolve of the same packages in order should produce the same result (at least in this case)
			packages: [][]string{{"kernel"}, {"vim-minimal", "tmux", "zsh"}},
			repos:    []rpmmd.RepoConfig{s.RepoConfig},
			sbomType: sbom.StandardTypeSpdx,
			err:      false,
		},
	}

	for tcName := range testCases {
		t.Run(tcName, func(t *testing.T) {
			assert := assert.New(t)
			tc := testCases[tcName]
			pkgsets := make([]rpmmd.PackageSet, len(tc.packages))
			for idx := range tc.packages {
				pkgsets[idx] = rpmmd.PackageSet{Include: tc.packages[idx], Repositories: tc.repos, InstallWeakDeps: true}
			}

			solver.SetRootDir(tc.rootDir)
			res, err := solver.Depsolve(pkgsets, tc.sbomType)
			if tc.err {
				assert.Error(err)
				assert.Contains(err.Error(), tc.expMsg)
				return
			} else {
				assert.Nil(err)
				require.NotNil(t, res)
			}

			assert.Equal(len(res.Repos), 1)
			assert.Equal(expectedResult(res.Repos[0]), res.Packages)

			if tc.sbomType != sbom.StandardTypeNone {
				require.NotNil(t, res.SBOM)
				assert.Equal(sbom.StandardTypeSpdx, res.SBOM.DocType)
			} else {
				assert.Nil(res.SBOM)
			}
		})
	}
}

func TestMakeDepsolveRequest(t *testing.T) {
	baseOS := rpmmd.RepoConfig{
		Name:     "baseos",
		BaseURLs: []string{"https://example.org/baseos"},
	}
	appstream := rpmmd.RepoConfig{
		Name:     "appstream",
		BaseURLs: []string{"https://example.org/appstream"},
	}
	userRepo := rpmmd.RepoConfig{
		Name:     "user-repo",
		BaseURLs: []string{"https://example.org/user-repo"},
	}
	userRepo2 := rpmmd.RepoConfig{
		Name:     "user-repo-2",
		BaseURLs: []string{"https://example.org/user-repo-2"},
	}
	moduleHotfixRepo := rpmmd.RepoConfig{
		Name:           "module-hotfixes",
		BaseURLs:       []string{"https://example.org/nginx"},
		ModuleHotfixes: common.ToPtr(true),
	}
	mtlsRepo := rpmmd.RepoConfig{
		Name:          "mtls",
		BaseURLs:      []string{"https://example.org/mtls"},
		SSLCACert:     "/cacert",
		SSLClientCert: "/cert",
		SSLClientKey:  "/key",
	}

	tests := []struct {
		packageSets []rpmmd.PackageSet
		args        []transactionArgs
		wantRepos   []repoConfig
		withSbom    bool
		err         bool
	}{
		// single transaction
		{
			packageSets: []rpmmd.PackageSet{
				{
					Include: []string{"pkg1"},
					Exclude: []string{"pkg2"},
					Repositories: []rpmmd.RepoConfig{
						baseOS,
						appstream,
					},
					InstallWeakDeps: true,
				},
			},
			args: []transactionArgs{
				{
					PackageSpecs:    []string{"pkg1"},
					ExcludeSpecs:    []string{"pkg2"},
					RepoIDs:         []string{baseOS.Hash(), appstream.Hash()},
					InstallWeakDeps: true,
				},
			},
			wantRepos: []repoConfig{
				{
					ID:       baseOS.Hash(),
					Name:     "baseos",
					BaseURLs: []string{"https://example.org/baseos"},
					repoHash: "f177f580cf201f52d1c62968d5b85cddae3e06cb9d5058987c07de1dbd769d4b",
				},
				{
					ID:       appstream.Hash(),
					Name:     "appstream",
					BaseURLs: []string{"https://example.org/appstream"},
					repoHash: "5c4a57bbb1b6a1886291819f2ceb25eb7c92e80065bc986a75c5837cf3d55a1f",
				},
			},
		},
		// 2 transactions + package set specific repo
		{
			packageSets: []rpmmd.PackageSet{
				{
					Include:         []string{"pkg1"},
					Exclude:         []string{"pkg2"},
					Repositories:    []rpmmd.RepoConfig{baseOS, appstream},
					InstallWeakDeps: true,
				},
				{
					Include:      []string{"pkg3"},
					Repositories: []rpmmd.RepoConfig{baseOS, appstream, userRepo},
				},
			},
			args: []transactionArgs{
				{
					PackageSpecs:    []string{"pkg1"},
					ExcludeSpecs:    []string{"pkg2"},
					RepoIDs:         []string{baseOS.Hash(), appstream.Hash()},
					InstallWeakDeps: true,
				},
				{
					PackageSpecs: []string{"pkg3"},
					RepoIDs:      []string{baseOS.Hash(), appstream.Hash(), userRepo.Hash()},
				},
			},
			wantRepos: []repoConfig{
				{
					ID:       baseOS.Hash(),
					Name:     "baseos",
					BaseURLs: []string{"https://example.org/baseos"},
					repoHash: "f177f580cf201f52d1c62968d5b85cddae3e06cb9d5058987c07de1dbd769d4b",
				},
				{
					ID:       appstream.Hash(),
					Name:     "appstream",
					BaseURLs: []string{"https://example.org/appstream"},
					repoHash: "5c4a57bbb1b6a1886291819f2ceb25eb7c92e80065bc986a75c5837cf3d55a1f",
				},
				{
					ID:       userRepo.Hash(),
					Name:     "user-repo",
					BaseURLs: []string{"https://example.org/user-repo"},
					repoHash: "1d3b23c311a5597ae217a0023eab3a401e7ba569066a0b91ffdcae04795af184",
				},
			},
		},
		// 2 transactions + no package set specific repos
		{
			packageSets: []rpmmd.PackageSet{
				{
					Include:         []string{"pkg1"},
					Exclude:         []string{"pkg2"},
					Repositories:    []rpmmd.RepoConfig{baseOS, appstream},
					InstallWeakDeps: true,
				},
				{
					Include:      []string{"pkg3"},
					Repositories: []rpmmd.RepoConfig{baseOS, appstream},
				},
			},
			args: []transactionArgs{
				{
					PackageSpecs:    []string{"pkg1"},
					ExcludeSpecs:    []string{"pkg2"},
					RepoIDs:         []string{baseOS.Hash(), appstream.Hash()},
					InstallWeakDeps: true,
				},
				{
					PackageSpecs: []string{"pkg3"},
					RepoIDs:      []string{baseOS.Hash(), appstream.Hash()},
				},
			},
			wantRepos: []repoConfig{
				{
					ID:       baseOS.Hash(),
					Name:     "baseos",
					BaseURLs: []string{"https://example.org/baseos"},
					repoHash: "f177f580cf201f52d1c62968d5b85cddae3e06cb9d5058987c07de1dbd769d4b",
				},
				{
					ID:       appstream.Hash(),
					Name:     "appstream",
					BaseURLs: []string{"https://example.org/appstream"},
					repoHash: "5c4a57bbb1b6a1886291819f2ceb25eb7c92e80065bc986a75c5837cf3d55a1f",
				},
			},
		},
		// 3 transactions + package set specific repo used by 2nd and 3rd transaction
		{
			packageSets: []rpmmd.PackageSet{
				{
					Include:         []string{"pkg1"},
					Exclude:         []string{"pkg2"},
					Repositories:    []rpmmd.RepoConfig{baseOS, appstream},
					InstallWeakDeps: true,
				},
				{
					Include:      []string{"pkg3"},
					Repositories: []rpmmd.RepoConfig{baseOS, appstream, userRepo},
				},
				{
					Include:      []string{"pkg4"},
					Repositories: []rpmmd.RepoConfig{baseOS, appstream, userRepo},
				},
			},
			args: []transactionArgs{
				{
					PackageSpecs:    []string{"pkg1"},
					ExcludeSpecs:    []string{"pkg2"},
					RepoIDs:         []string{baseOS.Hash(), appstream.Hash()},
					InstallWeakDeps: true,
				},
				{
					PackageSpecs: []string{"pkg3"},
					RepoIDs:      []string{baseOS.Hash(), appstream.Hash(), userRepo.Hash()},
				},
				{
					PackageSpecs: []string{"pkg4"},
					RepoIDs:      []string{baseOS.Hash(), appstream.Hash(), userRepo.Hash()},
				},
			},
			wantRepos: []repoConfig{
				{
					ID:       baseOS.Hash(),
					Name:     "baseos",
					BaseURLs: []string{"https://example.org/baseos"},
					repoHash: "f177f580cf201f52d1c62968d5b85cddae3e06cb9d5058987c07de1dbd769d4b",
				},
				{
					ID:       appstream.Hash(),
					Name:     "appstream",
					BaseURLs: []string{"https://example.org/appstream"},
					repoHash: "5c4a57bbb1b6a1886291819f2ceb25eb7c92e80065bc986a75c5837cf3d55a1f",
				},
				{
					ID:       userRepo.Hash(),
					Name:     "user-repo",
					BaseURLs: []string{"https://example.org/user-repo"},
					repoHash: "1d3b23c311a5597ae217a0023eab3a401e7ba569066a0b91ffdcae04795af184",
				},
			},
		},
		// 3 transactions + package set specific repo used by 2nd and 3rd transaction
		// + 3rd transaction using another repo
		{
			packageSets: []rpmmd.PackageSet{
				{
					Include:         []string{"pkg1"},
					Exclude:         []string{"pkg2"},
					Repositories:    []rpmmd.RepoConfig{baseOS, appstream},
					InstallWeakDeps: true,
				},
				{
					Include:      []string{"pkg3"},
					Repositories: []rpmmd.RepoConfig{baseOS, appstream, userRepo},
				},
				{
					Include:      []string{"pkg4"},
					Repositories: []rpmmd.RepoConfig{baseOS, appstream, userRepo, userRepo2},
				},
			},
			args: []transactionArgs{
				{
					PackageSpecs:    []string{"pkg1"},
					ExcludeSpecs:    []string{"pkg2"},
					RepoIDs:         []string{baseOS.Hash(), appstream.Hash()},
					InstallWeakDeps: true,
				},
				{
					PackageSpecs: []string{"pkg3"},
					RepoIDs:      []string{baseOS.Hash(), appstream.Hash(), userRepo.Hash()},
				},
				{
					PackageSpecs: []string{"pkg4"},
					RepoIDs:      []string{baseOS.Hash(), appstream.Hash(), userRepo.Hash(), userRepo2.Hash()},
				},
			},
			wantRepos: []repoConfig{
				{
					ID:       baseOS.Hash(),
					Name:     "baseos",
					BaseURLs: []string{"https://example.org/baseos"},
					repoHash: "f177f580cf201f52d1c62968d5b85cddae3e06cb9d5058987c07de1dbd769d4b",
				},
				{
					ID:       appstream.Hash(),
					Name:     "appstream",
					BaseURLs: []string{"https://example.org/appstream"},
					repoHash: "5c4a57bbb1b6a1886291819f2ceb25eb7c92e80065bc986a75c5837cf3d55a1f",
				},
				{
					ID:       userRepo.Hash(),
					Name:     "user-repo",
					BaseURLs: []string{"https://example.org/user-repo"},
					repoHash: "1d3b23c311a5597ae217a0023eab3a401e7ba569066a0b91ffdcae04795af184",
				},
				{
					ID:       userRepo2.Hash(),
					Name:     "user-repo-2",
					BaseURLs: []string{"https://example.org/user-repo-2"},
					repoHash: "9fca2ee4a26933d0b2f8e318b398d5e2bff53cb8c14d3c7a8c47f4429ccb4c41",
				},
			},
		},
		// Error: 3 transactions + 3rd one not using repo used by 2nd one
		{
			packageSets: []rpmmd.PackageSet{
				{
					Include:         []string{"pkg1"},
					Exclude:         []string{"pkg2"},
					Repositories:    []rpmmd.RepoConfig{baseOS, appstream},
					InstallWeakDeps: true,
				},
				{
					Include:      []string{"pkg3"},
					Repositories: []rpmmd.RepoConfig{baseOS, appstream, userRepo},
				},
				{
					Include:      []string{"pkg4"},
					Repositories: []rpmmd.RepoConfig{baseOS, appstream, userRepo2},
				},
			},
			err: true,
		},
		// Error: 3 transactions but last one doesn't specify user repos in 2nd
		{
			packageSets: []rpmmd.PackageSet{
				{
					Include:         []string{"pkg1"},
					Exclude:         []string{"pkg2"},
					Repositories:    []rpmmd.RepoConfig{baseOS, appstream},
					InstallWeakDeps: true,
				},
				{
					Include:      []string{"pkg3"},
					Repositories: []rpmmd.RepoConfig{baseOS, appstream, userRepo, userRepo2},
				},
				{
					Include:      []string{"pkg4"},
					Repositories: []rpmmd.RepoConfig{baseOS, appstream},
				},
			},
			err: true,
		},
		// module hotfixes flag passed
		{
			packageSets: []rpmmd.PackageSet{
				{
					Include:      []string{"pkg1"},
					Repositories: []rpmmd.RepoConfig{baseOS, appstream, moduleHotfixRepo},
				},
			},
			args: []transactionArgs{
				{
					PackageSpecs: []string{"pkg1"},
					RepoIDs:      []string{baseOS.Hash(), appstream.Hash(), moduleHotfixRepo.Hash()},
				},
			},
			wantRepos: []repoConfig{
				{
					ID:       baseOS.Hash(),
					Name:     "baseos",
					BaseURLs: []string{"https://example.org/baseos"},
					repoHash: "f177f580cf201f52d1c62968d5b85cddae3e06cb9d5058987c07de1dbd769d4b",
				},
				{
					ID:       appstream.Hash(),
					Name:     "appstream",
					BaseURLs: []string{"https://example.org/appstream"},
					repoHash: "5c4a57bbb1b6a1886291819f2ceb25eb7c92e80065bc986a75c5837cf3d55a1f",
				},
				{
					ID:             moduleHotfixRepo.Hash(),
					Name:           "module-hotfixes",
					BaseURLs:       []string{"https://example.org/nginx"},
					ModuleHotfixes: common.ToPtr(true),
					repoHash:       "b7d998ee8657964c17709e35ea7eaaffe4c84f9e41cc05250a1d16e8352d52e4",
				},
			},
		},
		// mtls certs passed
		{
			packageSets: []rpmmd.PackageSet{
				{
					Include:      []string{"pkg1"},
					Repositories: []rpmmd.RepoConfig{baseOS, appstream, mtlsRepo},
				},
			},
			args: []transactionArgs{
				{
					PackageSpecs: []string{"pkg1"},
					RepoIDs:      []string{baseOS.Hash(), appstream.Hash(), mtlsRepo.Hash()},
				},
			},
			wantRepos: []repoConfig{
				{
					ID:       baseOS.Hash(),
					Name:     "baseos",
					BaseURLs: []string{"https://example.org/baseos"},
					repoHash: "f177f580cf201f52d1c62968d5b85cddae3e06cb9d5058987c07de1dbd769d4b",
				},
				{
					ID:       appstream.Hash(),
					Name:     "appstream",
					BaseURLs: []string{"https://example.org/appstream"},
					repoHash: "5c4a57bbb1b6a1886291819f2ceb25eb7c92e80065bc986a75c5837cf3d55a1f",
				},
				{
					ID:            mtlsRepo.Hash(),
					Name:          "mtls",
					BaseURLs:      []string{"https://example.org/mtls"},
					SSLCACert:     "/cacert",
					SSLClientCert: "/cert",
					SSLClientKey:  "/key",
					repoHash:      "a1e83d633e76a8c6bcf5df21f010f4e97b864f2fa296c3dac214da10efad650a",
				},
			},
		},
		// 2 transactions + wantSbom flag
		{
			packageSets: []rpmmd.PackageSet{
				{
					Include:         []string{"pkg1"},
					Exclude:         []string{"pkg2"},
					Repositories:    []rpmmd.RepoConfig{baseOS, appstream},
					InstallWeakDeps: true,
				},
				{
					Include:      []string{"pkg3"},
					Repositories: []rpmmd.RepoConfig{baseOS, appstream},
				},
			},
			args: []transactionArgs{
				{
					PackageSpecs:    []string{"pkg1"},
					ExcludeSpecs:    []string{"pkg2"},
					RepoIDs:         []string{baseOS.Hash(), appstream.Hash()},
					InstallWeakDeps: true,
				},
				{
					PackageSpecs: []string{"pkg3"},
					RepoIDs:      []string{baseOS.Hash(), appstream.Hash()},
				},
			},
			wantRepos: []repoConfig{
				{
					ID:       baseOS.Hash(),
					Name:     "baseos",
					BaseURLs: []string{"https://example.org/baseos"},
					repoHash: "f177f580cf201f52d1c62968d5b85cddae3e06cb9d5058987c07de1dbd769d4b",
				},
				{
					ID:       appstream.Hash(),
					Name:     "appstream",
					BaseURLs: []string{"https://example.org/appstream"},
					repoHash: "5c4a57bbb1b6a1886291819f2ceb25eb7c92e80065bc986a75c5837cf3d55a1f",
				},
			},
			withSbom: true,
		},
	}
	solver := NewSolver("", "", "", "", "")
	for idx, tt := range tests {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			var sbomType sbom.StandardType
			if tt.withSbom {
				sbomType = sbom.StandardTypeSpdx
			}
			req, _, err := solver.makeDepsolveRequest(tt.packageSets, sbomType)
			if tt.err {
				assert.NotNilf(t, err, "expected an error, but got 'nil' instead")
				assert.Nilf(t, req, "got non-nill request, but expected an error")
			} else {
				assert.Nilf(t, err, "expected 'nil', but got error instead")
				assert.NotNilf(t, req, "expected non-nill request, but got 'nil' instead")

				assert.Equal(t, tt.args, req.Arguments.Transactions)
				assert.Equal(t, tt.wantRepos, req.Arguments.Repos)
				if tt.withSbom {
					assert.NotNil(t, req.Arguments.Sbom)
					assert.Equal(t, req.Arguments.Sbom.Type, sbom.StandardTypeSpdx.String())
				} else {
					assert.Nil(t, req.Arguments.Sbom)
				}
			}
		})
	}
}

//go:embed testdata/expected_packages.json
var expectedPackagesJSON []byte

func expectedResult(repo rpmmd.RepoConfig) rpmmd.PackageList {
	// need to change the url for the RemoteLocation and the repo ID since the port is different each time and we don't want to have a fixed one
	var templates []struct {
		Name            string   `json:"name"`
		Epoch           uint     `json:"epoch"`
		Version         string   `json:"version"`
		Release         string   `json:"release"`
		Arch            string   `json:"arch"`
		RemoteLocations []string `json:"remote_locations"`
		Checksum        struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"checksum"`
		Secrets   string `json:"secrets"`
		CheckGPG  bool   `json:"check_gpg"`
		IgnoreSSL bool   `json:"ignore_ssl"`
	}
	if err := json.Unmarshal(expectedPackagesJSON, &templates); err != nil {
		panic(fmt.Sprintf("failed to unmarshal expected packages JSON: %v", err))
	}

	exp := make(rpmmd.PackageList, len(templates))
	for idx, tmpl := range templates {
		exp[idx] = rpmmd.Package{
			Name:            tmpl.Name,
			Epoch:           tmpl.Epoch,
			Version:         tmpl.Version,
			Release:         tmpl.Release,
			Arch:            tmpl.Arch,
			RemoteLocations: tmpl.RemoteLocations,
			Checksum:        rpmmd.Checksum{Type: tmpl.Checksum.Type, Value: tmpl.Checksum.Value},
			Secrets:         tmpl.Secrets,
			CheckGPG:        tmpl.CheckGPG,
			IgnoreSSL:       tmpl.IgnoreSSL,
		}
		urlTemplate := exp[idx].RemoteLocations[0]
		exp[idx].RemoteLocations[0] = fmt.Sprintf(urlTemplate, strings.Join(repo.BaseURLs, ","))
		exp[idx].Location = strings.TrimPrefix(urlTemplate, "%s/")
		exp[idx].RepoID = repo.Id
	}
	return exp
}

func TestErrorRepoInfo(t *testing.T) {
	if !*forceDNF {
		// dnf tests aren't forced: skip them if the dnf sniff check fails
		if findDepsolveDnf() == "" {
			t.Skip("Test needs an installed osbuild-depsolve-dnf")
		}
	}

	assert := assert.New(t)

	type testCase struct {
		repo   rpmmd.RepoConfig
		expMsg string
	}

	testCases := []testCase{
		{
			repo: rpmmd.RepoConfig{
				Name:     "",
				BaseURLs: []string{"https://0.0.0.0/baseos/repo"},
				Metalink: "https://0.0.0.0/baseos/metalink",
			},
			expMsg: "https://0.0.0.0/baseos/repo",
		},
		{
			repo: rpmmd.RepoConfig{
				Name:     "baseos",
				BaseURLs: []string{"https://0.0.0.0/baseos/repo"},
				Metalink: "https://0.0.0.0/baseos/metalink",
			},
			expMsg: "https://0.0.0.0/baseos/repo",
		},
		{
			repo: rpmmd.RepoConfig{
				Name:     "fedora",
				Metalink: "https://0.0.0.0/f35/metalink",
			},
			expMsg: "https://0.0.0.0/f35/metalink",
		},
		{
			repo: rpmmd.RepoConfig{
				Name:       "",
				MirrorList: "https://0.0.0.0/baseos/mirrors",
			},
			expMsg: "https://0.0.0.0/baseos/mirrors",
		},
	}

	solver := NewSolver("platform:f38", "38", "x86_64", "fedora-38", "/tmp/cache")
	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			_, err := solver.Depsolve([]rpmmd.PackageSet{
				{
					Include:      []string{"osbuild"},
					Exclude:      nil,
					Repositories: []rpmmd.RepoConfig{tc.repo},
				},
			}, sbom.StandardTypeNone)
			assert.Error(err)
			assert.Contains(err.Error(), tc.expMsg)
		})
	}
}

func TestRepoConfigHash(t *testing.T) {
	repos := []rpmmd.RepoConfig{
		{
			Id:        "repoid-1",
			Name:      "A test repository",
			BaseURLs:  []string{"https://arepourl/"},
			IgnoreSSL: common.ToPtr(false),
		},
		{
			BaseURLs: []string{"https://adifferenturl/"},
		},
	}

	solver := NewSolver("platform:f38", "38", "x86_64", "fedora-38", "/tmp/cache")

	rcs, err := solver.reposFromRPMMD(repos)
	assert.Nil(t, err)

	hash := rcs[0].Hash()
	assert.Equal(t, 64, len(hash))

	assert.NotEqual(t, hash, rcs[1].Hash())
}

func TestRequestHash(t *testing.T) {
	solver := NewSolver("platform:f38", "38", "x86_64", "fedora-38", "/tmp/cache")
	repos := []rpmmd.RepoConfig{
		rpmmd.RepoConfig{
			Name:      "A test repository",
			BaseURLs:  []string{"https://arepourl/"},
			IgnoreSSL: common.ToPtr(false),
		},
	}

	req, err := solver.makeDumpRequest(repos)
	assert.Nil(t, err)
	hash := req.Hash()
	assert.Equal(t, 64, len(hash))

	req, err = solver.makeSearchRequest(repos, []string{"package0*"})
	assert.Nil(t, err)
	assert.Equal(t, 64, len(req.Hash()))
	assert.NotEqual(t, hash, req.Hash())
}

func TestRepoConfigMarshalAlsmostEmpty(t *testing.T) {
	repoCfg := &repoConfig{}
	js, _ := json.Marshal(repoCfg)
	// double check here that anything that uses pointers has "omitempty" set
	assert.Equal(t, string(js), `{"id":"","gpgcheck":false,"repo_gpgcheck":false}`)
}

func TestRunErrorEmptyOutput(t *testing.T) {
	fakeDepsolveDNFPath := filepath.Join(t.TempDir(), "osbuild-depsolve-dnf")
	fakeDepsolveDNFNoOutput := `#!/bin/sh -e
cat - > "$0".stdin
exit 1
`
	err := os.WriteFile(fakeDepsolveDNFPath, []byte(fakeDepsolveDNFNoOutput), 0o755)
	assert.NoError(t, err)

	_, err = run([]string{fakeDepsolveDNFPath}, &Request{}, nil)
	assert.EqualError(t, err, `DNF error occurred: InternalError: osbuild-depsolve-dnf output was empty`)
}

func TestSolverRunWithSolverNoError(t *testing.T) {
	tmpdir := t.TempDir()
	fakeSolver := `#!/bin/sh -e
cat - > "$0".stdin
echo '{"solver": "zypper"}'
>&2 echo "output-on-stderr" 
`
	fakeSolverPath := filepath.Join(tmpdir, "fake-solver")
	err := os.WriteFile(fakeSolverPath, []byte(fakeSolver), 0755) //nolint:gosec
	assert.NoError(t, err)

	var capturedStderr bytes.Buffer
	solver := NewSolver("platform:f38", "38", "x86_64", "fedora-38", "/tmp/cache")
	solver.Stderr = &capturedStderr
	solver.depsolveDNFCmd = []string{fakeSolverPath}
	res, err := solver.Depsolve(nil, sbom.StandardTypeNone)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, "output-on-stderr\n", capturedStderr.String())

	// prerequisite check, i.e. ensure our fake was called in the right way
	stdin, err := os.ReadFile(fakeSolverPath + ".stdin")
	assert.NoError(t, err)
	assert.Contains(t, string(stdin), `"command":"depsolve"`)

	// adding the "solver" did not cause any issues
	assert.NoError(t, err)
	assert.Equal(t, 0, len(res.Packages))
	assert.Equal(t, 0, len(res.Repos))
}

func TestDepsolveResultWithModulesKey(t *testing.T) {
	// quick test that verifies that `depsolveResult` understands JSON that contains
	// a `modules` key
	data := []byte(`{"modules": {}}`)

	var result depsolveResult
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	err := dec.Decode(&result)

	assert.NoError(t, err)
}

func TestDepsolverSubscriptionsError(t *testing.T) {
	if _, err := os.Stat("/etc/yum.repos.d/redhat.repo"); err == nil {
		t.Skip("Test must run on unsubscribed system")
	}

	tmpdir := t.TempDir()
	solver := NewSolver("platform:el9", "9", "x86_64", "rhel9.0", tmpdir)

	rootDir := t.TempDir()
	reposDir := filepath.Join(rootDir, "etc", "yum.repos.d")
	require.NoError(t, os.MkdirAll(reposDir, 0777))

	s := rpmrepo.NewTestServer()
	defer s.Close()
	s.WriteConfig(filepath.Join(reposDir, "test.repo"))
	s.RepoConfig.RHSM = true

	pkgsets := []rpmmd.PackageSet{
		{
			Include:      []string{"kernel"},
			Repositories: []rpmmd.RepoConfig{s.RepoConfig},
		},
	}
	solver.SetRootDir(rootDir)
	_, err := solver.Depsolve(pkgsets, 0)
	assert.EqualError(t, err, "makeDepsolveRequest failed: This system does not have any valid subscriptions. Subscribe it before specifying rhsm: true in sources (error details: no matching key and certificate pair)")
}
