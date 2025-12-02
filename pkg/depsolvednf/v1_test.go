package depsolvednf

import (
	"fmt"
	"testing"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/osbuild/images/pkg/sbom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestV1HandlerMakeDepsolveRequest(t *testing.T) {
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

	testCases := []struct {
		name        string
		packageSets []rpmmd.PackageSet
		withSbom    bool
		wantJSON    string
	}{
		{
			name: "single transaction",
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
			wantJSON: fmt.Sprintf(`{
				"command": "depsolve",
				"module_platform_id": "platform:el8",
				"releasever": "8",
				"arch": "x86_64",
				"cachedir": "/cache",
				"proxy": "",
				"arguments": {
					"repos": [
						{"id": %[1]q, "name": "baseos", "baseurl": ["https://example.org/baseos"], "gpgcheck": false, "repo_gpgcheck": false},
						{"id": %[2]q, "name": "appstream", "baseurl": ["https://example.org/appstream"], "gpgcheck": false, "repo_gpgcheck": false}
					],
					"search": {"latest": false, "packages": null},
					"transactions": [
						{"package-specs": ["pkg1"], "exclude-specs": ["pkg2"], "repo-ids": [%[1]q, %[2]q], "install_weak_deps": true}
					],
					"root_dir": "/root",
					"optional-metadata": ["filelists"]
				}
			}`, baseOS.Hash(), appstream.Hash()),
		},
		{
			name: "2 transactions + package set specific repo",
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
			wantJSON: fmt.Sprintf(`{
				"command": "depsolve",
				"module_platform_id": "platform:el8",
				"releasever": "8",
				"arch": "x86_64",
				"cachedir": "/cache",
				"proxy": "",
				"arguments": {
					"repos": [
						{"id": %[1]q, "name": "baseos", "baseurl": ["https://example.org/baseos"], "gpgcheck": false, "repo_gpgcheck": false},
						{"id": %[2]q, "name": "appstream", "baseurl": ["https://example.org/appstream"], "gpgcheck": false, "repo_gpgcheck": false},
						{"id": %[3]q, "name": "user-repo", "baseurl": ["https://example.org/user-repo"], "gpgcheck": false, "repo_gpgcheck": false}
					],
					"search": {"latest": false, "packages": null},
					"transactions": [
						{"package-specs": ["pkg1"], "exclude-specs": ["pkg2"], "repo-ids": [%[1]q, %[2]q], "install_weak_deps": true},
						{"package-specs": ["pkg3"], "exclude-specs": null, "repo-ids": [%[1]q, %[2]q, %[3]q], "install_weak_deps": false}
					],
					"root_dir": "/root",
					"optional-metadata": ["filelists"]
				}
			}`, baseOS.Hash(), appstream.Hash(), userRepo.Hash()),
		},
		{
			name: "2 transactions + no package set specific repos",
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
			wantJSON: fmt.Sprintf(`{
				"command": "depsolve",
				"module_platform_id": "platform:el8",
				"releasever": "8",
				"arch": "x86_64",
				"cachedir": "/cache",
				"proxy": "",
				"arguments": {
					"repos": [
						{"id": %[1]q, "name": "baseos", "baseurl": ["https://example.org/baseos"], "gpgcheck": false, "repo_gpgcheck": false},
						{"id": %[2]q, "name": "appstream", "baseurl": ["https://example.org/appstream"], "gpgcheck": false, "repo_gpgcheck": false}
					],
					"search": {"latest": false, "packages": null},
					"transactions": [
						{"package-specs": ["pkg1"], "exclude-specs": ["pkg2"], "repo-ids": [%[1]q, %[2]q], "install_weak_deps": true},
						{"package-specs": ["pkg3"], "exclude-specs": null, "repo-ids": [%[1]q, %[2]q], "install_weak_deps": false}
					],
					"root_dir": "/root",
					"optional-metadata": ["filelists"]
				}
			}`, baseOS.Hash(), appstream.Hash()),
		},
		{
			name: "3 transactions + package set specific repo used by 2nd and 3rd transaction",
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
			wantJSON: fmt.Sprintf(`{
				"command": "depsolve",
				"module_platform_id": "platform:el8",
				"releasever": "8",
				"arch": "x86_64",
				"cachedir": "/cache",
				"proxy": "",
				"arguments": {
					"repos": [
						{"id": %[1]q, "name": "baseos", "baseurl": ["https://example.org/baseos"], "gpgcheck": false, "repo_gpgcheck": false},
						{"id": %[2]q, "name": "appstream", "baseurl": ["https://example.org/appstream"], "gpgcheck": false, "repo_gpgcheck": false},
						{"id": %[3]q, "name": "user-repo", "baseurl": ["https://example.org/user-repo"], "gpgcheck": false, "repo_gpgcheck": false}
					],
					"search": {"latest": false, "packages": null},
					"transactions": [
						{"package-specs": ["pkg1"], "exclude-specs": ["pkg2"], "repo-ids": [%[1]q, %[2]q], "install_weak_deps": true},
						{"package-specs": ["pkg3"], "exclude-specs": null, "repo-ids": [%[1]q, %[2]q, %[3]q], "install_weak_deps": false},
						{"package-specs": ["pkg4"], "exclude-specs": null, "repo-ids": [%[1]q, %[2]q, %[3]q], "install_weak_deps": false}
					],
					"root_dir": "/root",
					"optional-metadata": ["filelists"]
				}
			}`, baseOS.Hash(), appstream.Hash(), userRepo.Hash()),
		},
		{
			name: "3 transactions + 3rd transaction using another repo",
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
			wantJSON: fmt.Sprintf(`{
				"command": "depsolve",
				"module_platform_id": "platform:el8",
				"releasever": "8",
				"arch": "x86_64",
				"cachedir": "/cache",
				"proxy": "",
				"arguments": {
					"repos": [
						{"id": %[1]q, "name": "baseos", "baseurl": ["https://example.org/baseos"], "gpgcheck": false, "repo_gpgcheck": false},
						{"id": %[2]q, "name": "appstream", "baseurl": ["https://example.org/appstream"], "gpgcheck": false, "repo_gpgcheck": false},
						{"id": %[3]q, "name": "user-repo", "baseurl": ["https://example.org/user-repo"], "gpgcheck": false, "repo_gpgcheck": false},
						{"id": %[4]q, "name": "user-repo-2", "baseurl": ["https://example.org/user-repo-2"], "gpgcheck": false, "repo_gpgcheck": false}
					],
					"search": {"latest": false, "packages": null},
					"transactions": [
						{"package-specs": ["pkg1"], "exclude-specs": ["pkg2"], "repo-ids": [%[1]q, %[2]q], "install_weak_deps": true},
						{"package-specs": ["pkg3"], "exclude-specs": null, "repo-ids": [%[1]q, %[2]q, %[3]q], "install_weak_deps": false},
						{"package-specs": ["pkg4"], "exclude-specs": null, "repo-ids": [%[1]q, %[2]q, %[3]q, %[4]q], "install_weak_deps": false}
					],
					"root_dir": "/root",
					"optional-metadata": ["filelists"]
				}
			}`, baseOS.Hash(), appstream.Hash(), userRepo.Hash(), userRepo2.Hash()),
		},
		{
			name: "module hotfixes flag passed",
			packageSets: []rpmmd.PackageSet{
				{
					Include:      []string{"pkg1"},
					Repositories: []rpmmd.RepoConfig{baseOS, appstream, moduleHotfixRepo},
				},
			},
			wantJSON: fmt.Sprintf(`{
				"command": "depsolve",
				"module_platform_id": "platform:el8",
				"releasever": "8",
				"arch": "x86_64",
				"cachedir": "/cache",
				"proxy": "",
				"arguments": {
					"repos": [
						{"id": %[1]q, "name": "baseos", "baseurl": ["https://example.org/baseos"], "gpgcheck": false, "repo_gpgcheck": false},
						{"id": %[2]q, "name": "appstream", "baseurl": ["https://example.org/appstream"], "gpgcheck": false, "repo_gpgcheck": false},
						{"id": %[3]q, "name": "module-hotfixes", "baseurl": ["https://example.org/nginx"], "gpgcheck": false, "repo_gpgcheck": false, "module_hotfixes": true}
					],
					"search": {"latest": false, "packages": null},
					"transactions": [
						{"package-specs": ["pkg1"], "exclude-specs": null, "repo-ids": [%[1]q, %[2]q, %[3]q], "install_weak_deps": false}
					],
					"root_dir": "/root",
					"optional-metadata": ["filelists"]
				}
			}`, baseOS.Hash(), appstream.Hash(), moduleHotfixRepo.Hash()),
		},
		{
			name: "mtls certs passed",
			packageSets: []rpmmd.PackageSet{
				{
					Include:      []string{"pkg1"},
					Repositories: []rpmmd.RepoConfig{baseOS, appstream, mtlsRepo},
				},
			},
			wantJSON: fmt.Sprintf(`{
				"command": "depsolve",
				"module_platform_id": "platform:el8",
				"releasever": "8",
				"arch": "x86_64",
				"cachedir": "/cache",
				"proxy": "",
				"arguments": {
					"repos": [
						{"id": %[1]q, "name": "baseos", "baseurl": ["https://example.org/baseos"], "gpgcheck": false, "repo_gpgcheck": false},
						{"id": %[2]q, "name": "appstream", "baseurl": ["https://example.org/appstream"], "gpgcheck": false, "repo_gpgcheck": false},
						{"id": %[3]q, "name": "mtls", "baseurl": ["https://example.org/mtls"], "gpgcheck": false, "repo_gpgcheck": false, "sslcacert": "/cacert", "sslclientkey": "/key", "sslclientcert": "/cert"}
					],
					"search": {"latest": false, "packages": null},
					"transactions": [
						{"package-specs": ["pkg1"], "exclude-specs": null, "repo-ids": [%[1]q, %[2]q, %[3]q], "install_weak_deps": false}
					],
					"root_dir": "/root",
					"optional-metadata": ["filelists"]
				}
			}`, baseOS.Hash(), appstream.Hash(), mtlsRepo.Hash()),
		},
		{
			name: "2 transactions + withSbom flag",
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
			withSbom: true,
			wantJSON: fmt.Sprintf(`{
				"command": "depsolve",
				"module_platform_id": "platform:el8",
				"releasever": "8",
				"arch": "x86_64",
				"cachedir": "/cache",
				"proxy": "",
				"arguments": {
					"repos": [
						{"id": %[1]q, "name": "baseos", "baseurl": ["https://example.org/baseos"], "gpgcheck": false, "repo_gpgcheck": false},
						{"id": %[2]q, "name": "appstream", "baseurl": ["https://example.org/appstream"], "gpgcheck": false, "repo_gpgcheck": false}
					],
					"search": {"latest": false, "packages": null},
					"transactions": [
						{"package-specs": ["pkg1"], "exclude-specs": ["pkg2"], "repo-ids": [%[1]q, %[2]q], "install_weak_deps": true},
						{"package-specs": ["pkg3"], "exclude-specs": null, "repo-ids": [%[1]q, %[2]q], "install_weak_deps": false}
					],
					"root_dir": "/root",
					"optional-metadata": ["filelists"],
					"sbom": {"type": "spdx"}
				}
			}`, baseOS.Hash(), appstream.Hash()),
		},
	}

	cfg := &solverConfig{
		modulePlatformID: "platform:el8",
		arch:             "x86_64",
		releaseVer:       "8",
		cacheDir:         "/cache",
		rootDir:          "/root",
	}
	v1Handler := newV1Handler()

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			var sbomType sbom.StandardType
			if tt.withSbom {
				sbomType = sbom.StandardTypeSpdx
			}

			rawReq, err := v1Handler.makeDepsolveRequest(cfg, tt.packageSets, sbomType)
			require.NoError(t, err)
			require.NotEmpty(t, rawReq)
			assert.JSONEq(t, tt.wantJSON, string(rawReq))
		})
	}
}

func TestV1HandlerMakeDumpRequest(t *testing.T) {
	baseOS := rpmmd.RepoConfig{
		Name:     "baseos",
		BaseURLs: []string{"https://example.org/baseos"},
	}
	appstream := rpmmd.RepoConfig{
		Name:     "appstream",
		BaseURLs: []string{"https://example.org/appstream"},
	}
	mtlsRepo := rpmmd.RepoConfig{
		Name:          "mtls",
		BaseURLs:      []string{"https://example.org/mtls"},
		SSLCACert:     "/cacert",
		SSLClientCert: "/cert",
		SSLClientKey:  "/key",
	}

	testCases := []struct {
		name     string
		repos    []rpmmd.RepoConfig
		wantJSON string
	}{
		{
			name:  "single repo",
			repos: []rpmmd.RepoConfig{baseOS},
			wantJSON: fmt.Sprintf(`{
				"command": "dump",
				"module_platform_id": "platform:el8",
				"releasever": "8",
				"arch": "x86_64",
				"cachedir": "/cache",
				"proxy": "",
				"arguments": {
					"repos": [
						{"id": %q, "name": "baseos", "baseurl": ["https://example.org/baseos"], "gpgcheck": false, "repo_gpgcheck": false}
					],
					"search": {"latest": false, "packages": null},
					"transactions": null,
					"root_dir": ""
				}
			}`, baseOS.Hash()),
		},
		{
			name:  "multiple repos",
			repos: []rpmmd.RepoConfig{baseOS, appstream},
			wantJSON: fmt.Sprintf(`{
				"command": "dump",
				"module_platform_id": "platform:el8",
				"releasever": "8",
				"arch": "x86_64",
				"cachedir": "/cache",
				"proxy": "",
				"arguments": {
					"repos": [
						{"id": %q, "name": "baseos", "baseurl": ["https://example.org/baseos"], "gpgcheck": false, "repo_gpgcheck": false},
						{"id": %q, "name": "appstream", "baseurl": ["https://example.org/appstream"], "gpgcheck": false, "repo_gpgcheck": false}
					],
					"search": {"latest": false, "packages": null},
					"transactions": null,
					"root_dir": ""
				}
			}`, baseOS.Hash(), appstream.Hash()),
		},
		{
			name:  "mtls certs passed",
			repos: []rpmmd.RepoConfig{baseOS, mtlsRepo},
			wantJSON: fmt.Sprintf(`{
				"command": "dump",
				"module_platform_id": "platform:el8",
				"releasever": "8",
				"arch": "x86_64",
				"cachedir": "/cache",
				"proxy": "",
				"arguments": {
					"repos": [
						{"id": %q, "name": "baseos", "baseurl": ["https://example.org/baseos"], "gpgcheck": false, "repo_gpgcheck": false},
						{"id": %q, "name": "mtls", "baseurl": ["https://example.org/mtls"], "gpgcheck": false, "repo_gpgcheck": false, "sslcacert": "/cacert", "sslclientkey": "/key", "sslclientcert": "/cert"}
					],
					"search": {"latest": false, "packages": null},
					"transactions": null,
					"root_dir": ""
				}
			}`, baseOS.Hash(), mtlsRepo.Hash()),
		},
	}

	cfg := &solverConfig{
		modulePlatformID: "platform:el8",
		arch:             "x86_64",
		releaseVer:       "8",
		cacheDir:         "/cache",
	}
	v1Handler := newV1Handler()

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			rawReq, err := v1Handler.makeDumpRequest(cfg, tt.repos)
			require.NoError(t, err)
			require.NotEmpty(t, rawReq)
			assert.JSONEq(t, tt.wantJSON, string(rawReq))
		})
	}
}

func TestV1HandlerMakeSearchRequest(t *testing.T) {
	baseOS := rpmmd.RepoConfig{
		Name:     "baseos",
		BaseURLs: []string{"https://example.org/baseos"},
	}
	appstream := rpmmd.RepoConfig{
		Name:     "appstream",
		BaseURLs: []string{"https://example.org/appstream"},
	}

	testCases := []struct {
		name     string
		repos    []rpmmd.RepoConfig
		packages []string
		wantJSON string
	}{
		{
			name:     "single package search",
			repos:    []rpmmd.RepoConfig{baseOS, appstream},
			packages: []string{"vim"},
			wantJSON: fmt.Sprintf(`{
				"command": "search",
				"module_platform_id": "platform:el8",
				"releasever": "8",
				"arch": "x86_64",
				"cachedir": "/cache",
				"proxy": "",
				"arguments": {
					"repos": [
						{"id": %q, "name": "baseos", "baseurl": ["https://example.org/baseos"], "gpgcheck": false, "repo_gpgcheck": false},
						{"id": %q, "name": "appstream", "baseurl": ["https://example.org/appstream"], "gpgcheck": false, "repo_gpgcheck": false}
					],
					"search": {"latest": false, "packages": ["vim"]},
					"transactions": null,
					"root_dir": ""
				}
			}`, baseOS.Hash(), appstream.Hash()),
		},
		{
			name:     "glob pattern search",
			repos:    []rpmmd.RepoConfig{baseOS},
			packages: []string{"python3*", "kernel-*"},
			wantJSON: fmt.Sprintf(`{
				"command": "search",
				"module_platform_id": "platform:el8",
				"releasever": "8",
				"arch": "x86_64",
				"cachedir": "/cache",
				"proxy": "",
				"arguments": {
					"repos": [
						{"id": %q, "name": "baseos", "baseurl": ["https://example.org/baseos"], "gpgcheck": false, "repo_gpgcheck": false}
					],
					"search": {"latest": false, "packages": ["python3*", "kernel-*"]},
					"transactions": null,
					"root_dir": ""
				}
			}`, baseOS.Hash()),
		},
		{
			name:     "empty packages list",
			repos:    []rpmmd.RepoConfig{baseOS},
			packages: []string{},
			wantJSON: fmt.Sprintf(`{
				"command": "search",
				"module_platform_id": "platform:el8",
				"releasever": "8",
				"arch": "x86_64",
				"cachedir": "/cache",
				"proxy": "",
				"arguments": {
					"repos": [
						{"id": %q, "name": "baseos", "baseurl": ["https://example.org/baseos"], "gpgcheck": false, "repo_gpgcheck": false}
					],
					"search": {"latest": false, "packages": []},
					"transactions": null,
					"root_dir": ""
				}
			}`, baseOS.Hash()),
		},
	}

	cfg := &solverConfig{
		modulePlatformID: "platform:el8",
		arch:             "x86_64",
		releaseVer:       "8",
		cacheDir:         "/cache",
	}
	v1Handler := newV1Handler()

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			rawReq, err := v1Handler.makeSearchRequest(cfg, tt.repos, tt.packages)
			require.NoError(t, err)
			require.NotEmpty(t, rawReq)
			assert.JSONEq(t, tt.wantJSON, string(rawReq))
		})
	}
}
