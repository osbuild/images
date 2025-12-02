package rpmmd_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/pkg/rpmmd"
)

func TestRepoConfigMarshalEmpty(t *testing.T) {
	repoCfg := &rpmmd.RepoConfig{}
	js, err := json.Marshal(repoCfg)
	assert.NoError(t, err)
	assert.Equal(t, string(js), `{}`)
}

func TestRepoConfigUnmarshalHappy(t *testing.T) {
	testCases := []struct {
		name string
		json string
		repo rpmmd.Repository
	}{
		{
			name: "single-baseurl",
			json: `{"baseurl":"http://example.com/repo"}`,
			repo: rpmmd.Repository{BaseURL: []string{"http://example.com/repo"}},
		},
		{
			name: "multiple-baseurls",
			json: `{"baseurl":["http://example.com/repo1", "http://example.com/repo2"]}`,
			repo: rpmmd.Repository{BaseURL: []string{"http://example.com/repo1", "http://example.com/repo2"}},
		},
		{
			name: "empty",
			json: `{}`,
			repo: rpmmd.Repository{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var repos rpmmd.Repository
			err := json.Unmarshal([]byte(tc.json), &repos)
			assert.NoError(t, err)
			assert.Equal(t, tc.repo, repos)
		})
	}
}

func TestRepoConfigUnmarshalSad(t *testing.T) {
	testCases := []struct {
		name        string
		json        string
		expectedErr string
	}{
		{
			name:        "wrong type",
			json:        `{"baseurl":true}`,
			expectedErr: `unexpected type for baseurl: bool`,
		},
		{
			name:        "wrong baseurl list content",
			json:        `{"baseurl": ["url1", 2.71]}`,
			expectedErr: `unexpected non-string value 2.71 in baseurl list`,
		},
		{
			name:        "wrong json",
			json:        `all-wrong`,
			expectedErr: `invalid character 'a' looking for beginning of value`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var repos rpmmd.Repository
			err := json.Unmarshal([]byte(tc.json), &repos)
			assert.EqualError(t, err, tc.expectedErr)
		})
	}
}
