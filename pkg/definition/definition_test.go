package definition

import (
	"embed"
	"io/fs"
	"path"
	"strings"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

//go:embed merge_test_data
var mergeTestData embed.FS

func TestMergeConfig(t *testing.T) {
	fsDataFs, err := fs.Sub(mergeTestData, "merge_test_data")
	require.NoError(t, err)
	fsData := fsDataFs.(fs.ReadFileFS)

	cases, err := fsData.(fs.ReadDirFS).ReadDir(".")
	require.NoError(t, err)

	for _, tc := range cases {
		if !tc.IsDir() {
			continue
		}
		t.Run(tc.Name(), func(t *testing.T) {
			actual, err := MergeConfig(fsData, path.Join(tc.Name(), "case.yaml"))

			// If there's an error file, we expect an error
			if expectedErr, ferr := fsData.ReadFile(path.Join(tc.Name(), "error")); ferr == nil {
				assert.EqualError(t, err, strings.TrimSpace(string(expectedErr)))
				return
			}

			// Otherwise, check no error and the actual output
			assert.NoError(t, err)

			expected, err := fsData.ReadFile(path.Join(tc.Name(), "expected.yaml"))
			assert.NoError(t, err)

			actualYaml, err := yaml.Marshal(actual)
			assert.NoError(t, err)

			assert.YAMLEq(t, string(expected), string(actualYaml))
		})
	}
}

func TestFindBestMatch(t *testing.T) {
	v := func(s string) version.Version {
		return *version.Must(version.NewVersion(s))
	}
	vp := func(s string) *version.Version {
		v := v(s)
		return &v
	}
	cases := []struct {
		name     string
		target   version.Version
		distros  []version.Version
		expected *version.Version
	}{
		{"no-versions", v("9.5"), nil, nil},
		{"no-older-versions", v("9.5"), []version.Version{v("9.6"), v("10.0")}, nil},
		{"exact-match", v("9.5"), []version.Version{v("9.4"), v("9.5"), v("9.6")}, vp("9.5")},
		{"older-match", v("9.5"), []version.Version{v("9.4"), v("9.6")}, vp("9.4")},
		{"older-match2", v("9.5"), []version.Version{v("9.4")}, vp("9.4")},
		{"older-major-match", v("10.0"), []version.Version{v("9.4"), v("9.6")}, vp("9.6")},
		{"multiple-majors", v("10.2"), []version.Version{v("9.6"), v("10.0")}, vp("10.0")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := findBestVersionMatch(tc.target, tc.distros)
			if tc.expected == nil {
				assert.Nil(t, actual)
			} else {
				assert.Equal(t, tc.expected.Original(), actual.Original())
			}
		})
	}
}

//go:embed find_best_def_file_test_data/*
var findBestDefFileData embed.FS

func TestFindBestDefinitionFile(t *testing.T) {
	cases := []struct {
		name          string
		distroId      string
		distroVersion string
		arch          string
		imageType     string
		expected      string
		err           string
	}{
		// happy
		{"exact-match", "hatos", "42", "x86_64", "livecd", "hatos/42/x86_64/livecd.yaml", ""},
		{"generic-fallback", "hatos", "42", "aarch64", "livecd", "hatos/42/generic/livecd.yaml", ""},
		{"older-version-fallback", "hatos", "43", "x86_64", "livecd", "hatos/42/x86_64/livecd.yaml", ""},
		{"exact-match2", "hatos", "43", "ppc64le", "livecd", "hatos/43/ppc64le/livecd.yaml", ""},

		// sad
		{"no-match", "hatos", "42", "ppc64le", "livedvd", "", "no match found for hatos, 42.0.0, ppc64le, livedvd"},
		{"no-distro", "nonexistent", "42", "x86_64", "livecd", "", "distro nonexistent doesn't have any definitions: open nonexistent: file does not exist"},
		{"invalid-version", "hatos", "abc", "x86_64", "livecd", "", "failed to parse requested distro version: Malformed version: abc"},
	}

	fsData, err := fs.Sub(findBestDefFileData, "find_best_def_file_test_data")
	require.NoError(t, err)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := FindBestDefinitionFile(fsData.(fs.ReadDirFS), tc.distroId, tc.distroVersion, tc.arch, tc.imageType)
			if tc.err == "" {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, actual)
			} else {
				assert.EqualError(t, err, tc.err)
			}
		})
	}

}
