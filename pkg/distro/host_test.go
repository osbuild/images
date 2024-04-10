package distro

import (
	"os"
	"path"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOSRelease(t *testing.T) {
	var cases = []struct {
		Input     string
		OSRelease map[string]string
	}{
		{
			``,
			map[string]string{},
		},
		{
			`NAME=Fedora
VERSION="30 (Workstation Edition)"
ID=fedora
VERSION_ID=30
VERSION_CODENAME=""
PLATFORM_ID="platform:f30"
PRETTY_NAME="Fedora 30 (Workstation Edition)"
VARIANT="Workstation Edition"
VARIANT_ID=workstation`,
			map[string]string{
				"NAME":             "Fedora",
				"VERSION":          "30 (Workstation Edition)",
				"ID":               "fedora",
				"VERSION_ID":       "30",
				"VERSION_CODENAME": "",
				"PLATFORM_ID":      "platform:f30",
				"PRETTY_NAME":      "Fedora 30 (Workstation Edition)",
				"VARIANT":          "Workstation Edition",
				"VARIANT_ID":       "workstation",
			},
		},
	}

	for i, c := range cases {
		r := strings.NewReader(c.Input)

		osrelease, err := readOSRelease(r)
		if err != nil {
			t.Fatalf("%d: readOSRelease: %v", i, err)
		}

		if !reflect.DeepEqual(osrelease, c.OSRelease) {
			t.Fatalf("%d: readOSRelease returned unexpected result: %#v", i, osrelease)
		}
	}
}

func TestReadOSReleaseFromTree(t *testing.T) {
	tree := t.TempDir()

	// initialize dirs
	require.NoError(t, os.MkdirAll(path.Join(tree, "usr/lib"), 0755))
	require.NoError(t, os.MkdirAll(path.Join(tree, "etc"), 0755))

	// firstly, let's write a simple /usr/lib/os-release
	require.NoError(t,
		os.WriteFile(path.Join(tree, "usr/lib/os-release"), []byte("ID=toucan\n"), 0600),
	)

	osRelease, err := ReadOSReleaseFromTree(tree)
	require.NoError(t, err)
	require.Equal(t, "toucan", osRelease["ID"])

	// secondly, let's override it with /etc/os-release
	require.NoError(t,
		os.WriteFile(path.Join(tree, "etc/os-release"), []byte("ID=kingfisher\n"), 0600),
	)

	osRelease, err = ReadOSReleaseFromTree(tree)
	require.NoError(t, err)
	require.Equal(t, "kingfisher", osRelease["ID"])
}

func TestReadOSReleaseFromTreeUnhappy(t *testing.T) {
	tree := t.TempDir()

	_, err := ReadOSReleaseFromTree(tree)
	require.ErrorContains(t, err, "failed to read os-release")
}
