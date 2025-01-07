package osbuild_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/pkg/osbuild"
	"github.com/osbuild/images/pkg/rpmmd"
)

var opensslPkg = rpmmd.PackageSpec{
	Name:           "openssl-libs",
	Epoch:          1,
	Version:        "3.0.1",
	Release:        "5.el9",
	Arch:           "x86_64",
	RemoteLocation: "https://example.com/repo/Packages/openssl-libs-3.0.1-5.el9.x86_64.rpm",
	Checksum:       "sha256:fcf2515ec9115551c99d552da721803ecbca23b7ae5a974309975000e8bef666",
	Path:           "Packages/openssl-libs-3.0.1-5.el9.x86_64.rpm",
	RepoID:         "repo_id",
}

var fakeRepos = map[string][]rpmmd.RepoConfig{
	"build": []rpmmd.RepoConfig{
		{
			Id:       "repo_id",
			Metalink: "http://example.com/metalink",
		},
	},
}

func TestLibrepoSimple(t *testing.T) {
	pkg := opensslPkg

	sources := osbuild.NewLibrepoSource()
	err := sources.AddPackage(pkg, fakeRepos)
	assert.NoError(t, err)

	expectedJSON := `{
  "items": {
    "sha256:fcf2515ec9115551c99d552da721803ecbca23b7ae5a974309975000e8bef666": {
      "path": "Packages/openssl-libs-3.0.1-5.el9.x86_64.rpm",
      "mirror": "repo_id"
    }
  },
  "options": {
    "mirrors": {
      "repo_id": {
        "url": "http://example.com/metalink",
        "type": "metalink"
      }
    }
  }
}`
	b, err := json.MarshalIndent(sources, "", "  ")
	assert.NoError(t, err)
	assert.Equal(t, expectedJSON, string(b))
}

func TestLibrepoInsecure(t *testing.T) {
	pkg := opensslPkg
	pkg.IgnoreSSL = true

	sources := osbuild.NewLibrepoSource()
	err := sources.AddPackage(pkg, fakeRepos)
	assert.NoError(t, err)

	expectedJSON := `{
  "items": {
    "sha256:fcf2515ec9115551c99d552da721803ecbca23b7ae5a974309975000e8bef666": {
      "path": "Packages/openssl-libs-3.0.1-5.el9.x86_64.rpm",
      "mirror": "repo_id"
    }
  },
  "options": {
    "mirrors": {
      "repo_id": {
        "url": "http://example.com/metalink",
        "type": "metalink",
        "insecure": true
      }
    }
  }
}`
	b, err := json.MarshalIndent(sources, "", "  ")
	assert.NoError(t, err)
	assert.Equal(t, expectedJSON, string(b))
}

func TestLibrepoSecrets(t *testing.T) {
	for _, secret := range []string{"org.osbuild.rhsm", "org.osbuild.mtls"} {
		pkg := opensslPkg
		pkg.Secrets = secret

		sources := osbuild.NewLibrepoSource()
		err := sources.AddPackage(pkg, fakeRepos)
		assert.NoError(t, err)

		expectedJSON := fmt.Sprintf(`{
  "items": {
    "sha256:fcf2515ec9115551c99d552da721803ecbca23b7ae5a974309975000e8bef666": {
      "path": "Packages/openssl-libs-3.0.1-5.el9.x86_64.rpm",
      "mirror": "repo_id"
    }
  },
  "options": {
    "mirrors": {
      "repo_id": {
        "url": "http://example.com/metalink",
        "type": "metalink",
        "secrets": {
          "name": "%s"
        }
      }
    }
  }
}`, secret)
		b, err := json.MarshalIndent(sources, "", "  ")
		assert.NoError(t, err)
		assert.Equal(t, expectedJSON, string(b))
	}
}
