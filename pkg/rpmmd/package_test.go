package rpmmd_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/stretchr/testify/assert"
)

func TestChecksumString(t *testing.T) {
	assert.Equal(t, "sha256:1234567890", (&rpmmd.Checksum{Type: "sha256", Value: "1234567890"}).String())
}

var packageList = rpmmd.PackageList{
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

func TestGetPackagePackageList(t *testing.T) {
	testCases := []struct {
		packages        rpmmd.PackageList
		packageName     string
		expectedPackage *rpmmd.Package
		expectedError   error
	}{
		{
			packages:        packageList,
			packageName:     "grub2",
			expectedPackage: &packageList[1],
		},
		{
			packages:        packageList,
			packageName:     "not-a-package",
			expectedPackage: nil,
			expectedError:   fmt.Errorf("package \"not-a-package\" not found in the Package list"),
		},
		{
			packages:        rpmmd.PackageList{},
			packageName:     "tmux",
			expectedPackage: nil,
			expectedError:   fmt.Errorf("package \"tmux\" not found in the Package list"),
		},
		{
			packages:        nil,
			packageName:     "tmux",
			expectedPackage: nil,
			expectedError:   fmt.Errorf("package \"tmux\" not found in the Package list"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.packageName, func(t *testing.T) {
			pkg, err := tc.packages.Package(tc.packageName)
			assert.Equal(t, tc.expectedError, err)
			assert.Equal(t, tc.expectedPackage, pkg)
		})
	}
}

var specs = []rpmmd.PackageSpec{
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

func TestPackageSpecGetEVRA(t *testing.T) {
	assert.Equal(t, "3.3a-3.fc38.x86_64", specs[0].GetEVRA())
	assert.Equal(t, "1:2.06-94.fc38.noarch", specs[1].GetEVRA())
}

func TestPackageSpecGetNEVRA(t *testing.T) {
	assert.Equal(t, "tmux-3.3a-3.fc38.x86_64", specs[0].GetNEVRA())
	assert.Equal(t, "grub2-1:2.06-94.fc38.noarch", specs[1].GetNEVRA())
}

func TestPackageSpecEmptyJson(t *testing.T) {
	pkg := &rpmmd.PackageSpec{Name: "pkg1"}
	js, err := json.Marshal(pkg)
	assert.NoError(t, err)
	assert.Equal(t, string(js), `{"name":"pkg1","epoch":0}`)
}

func TestPackageSpecFull(t *testing.T) {
	pkg := &rpmmd.PackageSpec{
		Name:           "acl",
		Epoch:          0,
		Version:        "2.3.1",
		Release:        "3.el9",
		Arch:           "x86_64",
		RemoteLocation: "http://example.com/repo/Packages/acl-2.3.1-3.el9.x86_64.rpm",
		Checksum:       "sha256:986044c3837eddbc9231d7be5e5fc517e245296978b988a803bc9f9172fe84ea",
		Secrets:        "",
		CheckGPG:       false,
		IgnoreSSL:      true,
		Path:           "Packages/acl-2.3.1-3.el9.x86_64.rpm",
		RepoID:         "813859d10fe28ff54dbde44655a18b071c8adbaa849a551ec23cc415f0f7f1b0",
	}

	js, err := json.MarshalIndent(pkg, "", " ")
	assert.NoError(t, err)
	assert.Equal(t, string(js), `{
 "name": "acl",
 "epoch": 0,
 "version": "2.3.1",
 "release": "3.el9",
 "arch": "x86_64",
 "remote_location": "http://example.com/repo/Packages/acl-2.3.1-3.el9.x86_64.rpm",
 "checksum": "sha256:986044c3837eddbc9231d7be5e5fc517e245296978b988a803bc9f9172fe84ea",
 "ignore_ssl": true,
 "path": "Packages/acl-2.3.1-3.el9.x86_64.rpm",
 "repo_id": "813859d10fe28ff54dbde44655a18b071c8adbaa849a551ec23cc415f0f7f1b0"
}`)
}
