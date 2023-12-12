package rpmmd

import (
	"encoding/json"
	"testing"

	"github.com/osbuild/images/internal/common"
	"github.com/stretchr/testify/assert"
)

func TestPackageSpecGetEVRA(t *testing.T) {
	specs := []PackageSpec{
		{
			Name:    "tmux",
			Epoch:   0,
			Version: "3.3a",
			Release: "3.fc38",
			Arch:    "x86_64",
		},
		{
			Name:    "grub2",
			Epoch:   1,
			Version: "2.06",
			Release: "94.fc38",
			Arch:    "noarch",
		},
	}

	assert.Equal(t, "3.3a-3.fc38.x86_64", specs[0].GetEVRA())
	assert.Equal(t, "1:2.06-94.fc38.noarch", specs[1].GetEVRA())
}

func TestPackageSpecGetNEVRA(t *testing.T) {
	specs := []PackageSpec{
		{
			Name:    "tmux",
			Epoch:   0,
			Version: "3.3a",
			Release: "3.fc38",
			Arch:    "x86_64",
		},
		{
			Name:    "grub2",
			Epoch:   1,
			Version: "2.06",
			Release: "94.fc38",
			Arch:    "noarch",
		},
	}

	assert.Equal(t, "tmux-3.3a-3.fc38.x86_64", specs[0].GetNEVRA())
	assert.Equal(t, "grub2-1:2.06-94.fc38.noarch", specs[1].GetNEVRA())
}

func TestRepoConfigMarshalEmpty(t *testing.T) {
	repoCfg := &RepoConfig{}
	js, _ := json.Marshal(repoCfg)
	assert.Equal(t, string(js), `{}`)
}

func TestOldWorkerRepositoryCompatUnmarshal(t *testing.T) {
	testCases := []struct {
		repoJSON []byte
		repo     RepoConfig
	}{
		{
			repoJSON: []byte(`{"name":"fedora","baseurl":"http://example.com/fedora"}`),
			repo: RepoConfig{
				Name:     "fedora",
				BaseURLs: []string{"http://example.com/fedora"},
			},
		},
		{
			repoJSON: []byte(`{"name":"multiple","baseurl":"http://example.com/one,http://example.com/two"}`),
			repo: RepoConfig{
				Name:     "multiple",
				BaseURLs: []string{"http://example.com/one", "http://example.com/two"},
			},
		},
		{
			repoJSON: []byte(`{"id":"all","name":"all","baseurls":["http://example.com/all"],"metalink":"http://example.com/metalink","mirrorlist":"http://example.com/mirrorlist","gpgkeys":["key1","key2"],"check_gpg":true,"check_repo_gpg":true,"ignore_ssl":true,"priority":10,"metadata_expire":"test","rhsm":true,"enabled":true,"image_type_tags":["one","two"],"package_sets":["1","2"],"baseurl":"http://example.com/all"}`),
			repo: RepoConfig{
				Id:             "all",
				Name:           "all",
				BaseURLs:       []string{"http://example.com/all"},
				Metalink:       "http://example.com/metalink",
				MirrorList:     "http://example.com/mirrorlist",
				GPGKeys:        []string{"key1", "key2"},
				CheckGPG:       common.ToPtr(true),
				CheckRepoGPG:   common.ToPtr(true),
				IgnoreSSL:      common.ToPtr(true),
				Priority:       common.ToPtr(10),
				MetadataExpire: "test",
				RHSM:           true,
				Enabled:        common.ToPtr(true),
				ImageTypeTags:  []string{"one", "two"},
				PackageSets:    []string{"1", "2"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.repo.Name, func(t *testing.T) {
			var repo RepoConfig
			err := json.Unmarshal(tc.repoJSON, &repo)
			assert.Nil(t, err)
			assert.Equal(t, tc.repo, repo)
		})
	}
}

func TestOldWorkerRepositoryCompatMarshal(t *testing.T) {
	testCases := []struct {
		repoJSON []byte
		repo     RepoConfig
	}{
		{
			repoJSON: []byte(`{"id":"fedora","name":"fedora","baseurls":["http://example.com/fedora"],"baseurl":"http://example.com/fedora"}`),
			repo: RepoConfig{
				Id:       "fedora",
				Name:     "fedora",
				BaseURLs: []string{"http://example.com/fedora"},
			},
		},
		{
			repoJSON: []byte(`{"id":"multiple","name":"multiple","baseurls":["http://example.com/one","http://example.com/two"],"baseurl":"http://example.com/one,http://example.com/two"}`),
			repo: RepoConfig{
				Id:       "multiple",
				Name:     "multiple",
				BaseURLs: []string{"http://example.com/one", "http://example.com/two"},
			},
		},
		{
			repoJSON: []byte(`{"id":"all","name":"all","baseurls":["http://example.com/all"],"metalink":"http://example.com/metalink","mirrorlist":"http://example.com/mirrorlist","gpgkeys":["key1","key2"],"check_gpg":true,"check_repo_gpg":true,"priority":10,"ignore_ssl":true,"metadata_expire":"test","rhsm":true,"enabled":true,"image_type_tags":["one","two"],"package_sets":["1","2"],"baseurl":"http://example.com/all"}`),
			repo: RepoConfig{
				Id:             "all",
				Name:           "all",
				BaseURLs:       []string{"http://example.com/all"},
				Metalink:       "http://example.com/metalink",
				MirrorList:     "http://example.com/mirrorlist",
				GPGKeys:        []string{"key1", "key2"},
				CheckGPG:       common.ToPtr(true),
				CheckRepoGPG:   common.ToPtr(true),
				Priority:       common.ToPtr(10),
				IgnoreSSL:      common.ToPtr(true),
				MetadataExpire: "test",
				RHSM:           true,
				Enabled:        common.ToPtr(true),
				ImageTypeTags:  []string{"one", "two"},
				PackageSets:    []string{"1", "2"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.repo.Name, func(t *testing.T) {
			gotJson, err := json.Marshal(tc.repo)
			assert.Nil(t, err)
			assert.Equal(t, tc.repoJSON, gotJson)
		})
	}
}
