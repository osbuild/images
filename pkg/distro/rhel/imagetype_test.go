package rhel_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/distrofactory"
	"github.com/osbuild/images/pkg/rpmmd"
)

// XXX: keep in sync with fedora/imagetype_test.go
func TestManifestRepositoryCustomization(t *testing.T) {
	var options distro.ImageOptions
	var repos []rpmmd.RepoConfig

	// this is the only difference compared to the fedora test /o\
	distroFactory := distrofactory.NewDefault()
	distro := distroFactory.GetDistro("rhel-9.4")
	arch, err := distro.GetArch("x86_64")
	assert.NoError(t, err)
	imgType, err := arch.GetImageType("qcow2")
	assert.NoError(t, err)

	for _, enabledDuringBuild := range []bool{false, true} {
		t.Run(fmt.Sprintf("repo enabled %v", enabledDuringBuild), func(t *testing.T) {
			bp := &blueprint.Blueprint{
				Packages: []blueprint.Package{{Name: "hello"}},
				Customizations: &blueprint.Customizations{
					Repositories: []blueprint.RepositoryCustomization{
						{Id: "repo1", BaseURLs: []string{"example.com/repo1"}},
						{Id: "repo2", BaseURLs: []string{"example.com/repo2"}, EnabledDuringBuild: enabledDuringBuild},
					},
				},
			}
			mani, _, err := imgType.Manifest(bp, options, repos, nil)
			assert.NoError(t, err)
			chains := mani.GetPackageSetChains()
			osChains := chains["os"]
			baseChain := osChains[0]
			assert.Contains(t, baseChain.Include, "kernel")
			payloadChain := osChains[1]
			assert.Equal(t, []string{"hello"}, payloadChain.Include)
			if enabledDuringBuild {
				// the bp repo got added for this payload resolving
				assert.Equal(t, 1, len(payloadChain.Repositories))
				expected := []rpmmd.RepoConfig{
					{Id: "repo2", BaseURLs: []string{"example.com/repo2"}, GPGKeys: []string{}},
				}
				assert.Equal(t, expected, payloadChain.Repositories)
			} else {
				// we configured no base repos and the bp repo is not included
				assert.Equal(t, 0, len(payloadChain.Repositories))
			}
		})
	}
}
