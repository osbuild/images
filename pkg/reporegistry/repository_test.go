package reporegistry

import (
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/sirupsen/logrus"
	logrusTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/distro/test_distro"
	"github.com/osbuild/images/pkg/rpmmd"
)

func getConfPaths(t *testing.T) []string {
	confPaths := []string{
		"./test/confpaths/priority1",
		"./test/confpaths/priority2",
	}
	var absConfPaths []string

	for _, path := range confPaths {
		absPath, err := filepath.Abs(path)
		assert.Nil(t, err)
		absConfPaths = append(absConfPaths, absPath)
	}

	return absConfPaths
}

func TestLoadRepositoriesExisting(t *testing.T) {
	confPaths := getConfPaths(t)
	type args struct {
		distro string
	}
	tests := []struct {
		name string
		args args
		want map[string][]string
	}{
		{
			name: "duplicate distro definition, load first encounter",
			args: args{
				distro: "fedora-33",
			},
			want: map[string][]string{
				test_distro.TestArchName:  {"fedora-33-p1", "updates-33-p1"},
				test_distro.TestArch2Name: {"fedora-33-p1", "updates-33-p1"},
			},
		},
		{
			name: "single distro definition",
			args: args{
				distro: "fedora-34",
			},
			want: map[string][]string{
				test_distro.TestArchName:  {"fedora-34-p2", "updates-34-p2"},
				test_distro.TestArch2Name: {"fedora-34-p2", "updates-34-p2"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadRepositories(confPaths, tt.args.distro)
			assert.Nil(t, err)

			for wantArch, wantRepos := range tt.want {
				gotArchRepos, exists := got[wantArch]
				assert.True(t, exists, "Expected '%s' arch in repos definition for '%s', but it does not exist", wantArch, tt.args.distro)

				var gotNames []string
				for _, r := range gotArchRepos {
					gotNames = append(gotNames, r.Name)
				}

				if !reflect.DeepEqual(gotNames, wantRepos) {
					t.Errorf("LoadRepositories() for %s/%s =\n got: %#v\n want: %#v", tt.args.distro, wantArch, gotNames, wantRepos)
				}
			}

		})
	}
}

func TestLoadRepositoriesNonExisting(t *testing.T) {
	confPaths := getConfPaths(t)
	repos, err := LoadRepositories(confPaths, "my-imaginary-distro")
	assert.Nil(t, repos)
	assert.NotNil(t, err)
}

func Test_LoadAllRepositories(t *testing.T) {
	expectedReposMap := rpmmd.DistrosRepoConfigs{
		"fedora-33": {
			test_distro.TestArchName: {
				{
					Name:     "fedora-33-p1",
					BaseURLs: []string{"https://example.com/fedora-33-p1/test_arch"},
					GPGKeys:  []string{"FAKE-GPG-KEY"},
					CheckGPG: common.ToPtr(true),
				},
				{
					Name:     "updates-33-p1",
					BaseURLs: []string{"https://example.com/updates-33-p1/test_arch"},
					GPGKeys:  []string{"FAKE-GPG-KEY"},
					CheckGPG: common.ToPtr(true),
				},
			},
			test_distro.TestArch2Name: {
				{
					Name:     "fedora-33-p1",
					BaseURLs: []string{"https://example.com/fedora-33-p1/test_arch2"},
					GPGKeys:  []string{"FAKE-GPG-KEY"},
					CheckGPG: common.ToPtr(true),
				},
				{
					Name:     "updates-33-p1",
					BaseURLs: []string{"https://example.com/updates-33-p1/test_arch2"},
					GPGKeys:  []string{"FAKE-GPG-KEY"},
					CheckGPG: common.ToPtr(true),
				},
			},
		},
		"fedora-34": {
			test_distro.TestArchName: {
				{
					Name:     "fedora-34-p2",
					BaseURLs: []string{"https://example.com/fedora-34-p2/test_arch"},
					GPGKeys:  []string{"FAKE-GPG-KEY"},
					CheckGPG: common.ToPtr(true),
				},
				{
					Name:     "updates-34-p2",
					BaseURLs: []string{"https://example.com/updates-34-p2/test_arch"},
					GPGKeys:  []string{"FAKE-GPG-KEY"},
					CheckGPG: common.ToPtr(true),
				},
			},
			test_distro.TestArch2Name: {
				{
					Name:     "fedora-34-p2",
					BaseURLs: []string{"https://example.com/fedora-34-p2/test_arch2"},
					GPGKeys:  []string{"FAKE-GPG-KEY"},
					CheckGPG: common.ToPtr(true),
				},
				{
					Name:     "updates-34-p2",
					BaseURLs: []string{"https://example.com/updates-34-p2/test_arch2"},
					GPGKeys:  []string{"FAKE-GPG-KEY"},
					CheckGPG: common.ToPtr(true),
				},
			},
		},
		"rhel-8.7": {
			test_distro.TestArchName: {
				{
					Name:     "rhel-8.7-baseos-p1",
					BaseURLs: []string{"https://example.com/rhel-8.7-baseos-p1/test_arch"},
					GPGKeys:  []string{"FAKE-GPG-KEY"},
					CheckGPG: common.ToPtr(true),
				},
				{
					Name:     "rhel-8.7-appstream-p1",
					BaseURLs: []string{"https://example.com/rhel-8.7-appstream-p1/test_arch"},
					GPGKeys:  []string{"FAKE-GPG-KEY"},
					CheckGPG: common.ToPtr(true),
				},
			},
			test_distro.TestArch2Name: {
				{
					Name:     "rhel-8.7-baseos-p1",
					BaseURLs: []string{"https://example.com/rhel-8.7-baseos-p1/test_arch2"},
					GPGKeys:  []string{"FAKE-GPG-KEY"},
					CheckGPG: common.ToPtr(true),
				},
				{
					Name:     "rhel-8.7-appstream-p1",
					BaseURLs: []string{"https://example.com/rhel-8.7-appstream-p1/test_arch2"},
					GPGKeys:  []string{"FAKE-GPG-KEY"},
					CheckGPG: common.ToPtr(true),
				},
			},
		},
		"rhel-8.8": {
			test_distro.TestArchName: {
				{
					Name:     "rhel-8.8-baseos-p1",
					BaseURLs: []string{"https://example.com/rhel-8.8-baseos-p1/test_arch"},
					GPGKeys:  []string{"FAKE-GPG-KEY"},
					CheckGPG: common.ToPtr(true),
				},
				{
					Name:     "rhel-8.8-appstream-p1",
					BaseURLs: []string{"https://example.com/rhel-8.8-appstream-p1/test_arch"},
					GPGKeys:  []string{"FAKE-GPG-KEY"},
					CheckGPG: common.ToPtr(true),
				},
			},
			test_distro.TestArch2Name: {
				{
					Name:     "rhel-8.8-baseos-p1",
					BaseURLs: []string{"https://example.com/rhel-8.8-baseos-p1/test_arch2"},
					GPGKeys:  []string{"FAKE-GPG-KEY"},
					CheckGPG: common.ToPtr(true),
				},
				{
					Name:     "rhel-8.8-appstream-p1",
					BaseURLs: []string{"https://example.com/rhel-8.8-appstream-p1/test_arch2"},
					GPGKeys:  []string{"FAKE-GPG-KEY"},
					CheckGPG: common.ToPtr(true),
				},
			},
		},
		"rhel-8.9": {
			test_distro.TestArchName: {
				{
					Name:     "rhel-8.9-baseos-p2",
					BaseURLs: []string{"https://example.com/rhel-8.9-baseos-p2/test_arch"},
					GPGKeys:  []string{"FAKE-GPG-KEY"},
					CheckGPG: common.ToPtr(true),
				},
				{
					Name:     "rhel-8.9-appstream-p2",
					BaseURLs: []string{"https://example.com/rhel-8.9-appstream-p2/test_arch"},
					GPGKeys:  []string{"FAKE-GPG-KEY"},
					CheckGPG: common.ToPtr(true),
				},
			},
			test_distro.TestArch2Name: {
				{
					Name:     "rhel-8.9-baseos-p2",
					BaseURLs: []string{"https://example.com/rhel-8.9-baseos-p2/test_arch2"},
					GPGKeys:  []string{"FAKE-GPG-KEY"},
					CheckGPG: common.ToPtr(true),
				},
				{
					Name:     "rhel-8.9-appstream-p2",
					BaseURLs: []string{"https://example.com/rhel-8.9-appstream-p2/test_arch2"},
					GPGKeys:  []string{"FAKE-GPG-KEY"},
					CheckGPG: common.ToPtr(true),
				},
			},
		},
		"rhel-8.10": {
			test_distro.TestArchName: {
				{
					Name:     "rhel-8.10-baseos-p1",
					BaseURLs: []string{"https://example.com/rhel-8.10-baseos-p1/test_arch"},
					GPGKeys:  []string{"FAKE-GPG-KEY"},
					CheckGPG: common.ToPtr(true),
				},
				{
					Name:     "rhel-8.10-appstream-p1",
					BaseURLs: []string{"https://example.com/rhel-8.10-appstream-p1/test_arch"},
					GPGKeys:  []string{"FAKE-GPG-KEY"},
					CheckGPG: common.ToPtr(true),
				},
			},
			test_distro.TestArch2Name: {
				{
					Name:     "rhel-8.10-baseos-p1",
					BaseURLs: []string{"https://example.com/rhel-8.10-baseos-p1/test_arch2"},
					GPGKeys:  []string{"FAKE-GPG-KEY"},
					CheckGPG: common.ToPtr(true),
				},
				{
					Name:     "rhel-8.10-appstream-p1",
					BaseURLs: []string{"https://example.com/rhel-8.10-appstream-p1/test_arch2"},
					GPGKeys:  []string{"FAKE-GPG-KEY"},
					CheckGPG: common.ToPtr(true),
				},
			},
		},
	}

	confPaths := getConfPaths(t)

	distroReposMap, err := LoadAllRepositories(confPaths)
	assert.NotNil(t, distroReposMap)
	assert.Nil(t, err)
	assert.Equal(t, len(distroReposMap), len(expectedReposMap))

	for expectedDistroName, expectedDistroArchRepos := range expectedReposMap {
		t.Run(expectedDistroName, func(t *testing.T) {
			distroArchRepos, exists := distroReposMap[expectedDistroName]
			assert.True(t, exists)
			assert.Equal(t, len(distroArchRepos), len(expectedDistroArchRepos))

			for expectedArch, expectedRepos := range expectedDistroArchRepos {
				repos, exists := distroArchRepos[expectedArch]
				assert.True(t, exists)
				if !reflect.DeepEqual(repos, expectedRepos) {
					t.Errorf("LoadAllRepositories() for %s/%s =\n got: %#v\n want: %#v", expectedDistroName, expectedArch, repos, expectedRepos)
				}
			}
		})
	}
}

func TestLoadRepositoriesLogging(t *testing.T) {
	_, logHook := logrusTest.NewNullLogger()
	logrus.AddHook(logHook)

	confPaths := getConfPaths(t)
	_, err := LoadAllRepositories(confPaths)
	require.NoError(t, err)
	needle := "Loaded repository configuration file: rhel-8.10.json"
	assert.True(t, slices.ContainsFunc(logHook.AllEntries(), func(entry *logrus.Entry) bool {
		return needle == entry.Message
	}), fmt.Sprintf("%q not found in look entries %+v (last: %q)", needle, logHook.AllEntries(), logHook.LastEntry().Message))
}
