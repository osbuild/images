package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/internal/testregistry"
	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	resolver "github.com/osbuild/images/cmd/otk-resolve-containers"
)

const (
	owner    = "osbuild"
	reponame = "testcontainer"
)

func createTestRegistry() (*testregistry.Registry, []string) {
	registry := testregistry.New()
	repo := registry.AddRepo(fmt.Sprintf("%s/%s", owner, reponame))
	ref := registry.GetRef(fmt.Sprintf("%s/%s", owner, reponame))

	// add 10 images, all in the same repository with the same content
	// (rootLayer), but each with a different tag and comment
	refs := make([]string, 10)
	for idx := 0; idx < len(refs); idx++ {
		checksum := repo.AddImage(
			[]testregistry.Blob{testregistry.NewDataBlobFromBase64(testregistry.RootLayer)},
			[]string{"amd64", "ppc64le"},
			fmt.Sprintf("image %d", idx),
			time.Time{})

		tag := fmt.Sprintf("tag-%d", idx)
		repo.AddTag(checksum, tag)
		refs[idx] = fmt.Sprintf("%s:%s", ref, tag)
	}
	return registry, refs
}

func TestResolver(t *testing.T) {
	registry, refs := createTestRegistry()
	defer registry.Close()

	inpContainers := make([]blueprint.Container, len(refs))
	for idx, ref := range refs {
		inpContainers[idx] = blueprint.Container{
			Source:       ref,
			Name:         fmt.Sprintf("test/localhost/%s", ref), // add a prefix for the local name to override the source
			TLSVerify:    common.ToPtr(false),
			LocalStorage: false,
		}
	}

	for _, containerArch := range []string{"amd64", "ppc64le"} {
		t.Run(containerArch, func(t *testing.T) {

			require := require.New(t)
			assert := assert.New(t)

			input := map[string]interface{}{
				"arch":       containerArch,
				"containers": inpContainers,
			}
			inputReq, err := json.Marshal(map[string]map[string]interface{}{
				"tree": input,
			})
			require.NoError(err)

			inpBuf := bytes.NewBuffer(inputReq)
			outBuf := &bytes.Buffer{}

			assert.NoError(resolver.Run(inpBuf, outBuf))

			var output map[string]resolver.Output
			require.NoError(json.Unmarshal(outBuf.Bytes(), &output))

			outputContainers := output["tree"].Const.Containers

			assert.Len(outputContainers, len(refs))

			expectedOutput := make([]resolver.ContainerInfo, len(refs))
			for idx, ref := range refs {
				// resolve directly with the registry and convert to ContainerInfo to
				// compare with output.
				spec, err := registry.Resolve(ref, arch.FromString(containerArch))
				assert.NoError(err)
				expectedOutput[idx] = resolver.ContainerInfo{
					Source:  spec.Source,
					Digest:  spec.Digest,
					ImageID: spec.ImageID,
					// registry.Resolve() copies the ref to the local name but the
					// resolver will add the user-defined local name instead
					LocalName:  fmt.Sprintf("test/localhost/%s", ref),
					ListDigest: spec.ListDigest,
					Arch:       spec.Arch.String(),
					TLSVerify:  spec.TLSVerify,
				}
			}

			// NOTE: the order of containers in the resolver's output is stable but is
			// not the same as the order of the inputs.
			assert.ElementsMatch(outputContainers, expectedOutput)
		})
	}
}

func TestResolverUnhappy(t *testing.T) {
	registry, refs := createTestRegistry()
	defer registry.Close()

	type testCase struct {
		source    string
		arch      string
		tlsverify bool
		errSubstr string
	}

	testCases := map[string]testCase{
		"bad-registry": {
			source:    "127.0.0.2:1990/org/repo:tag",
			arch:      "amd64",
			errSubstr: "127.0.0.2:1990: connect: connection refused",
		},
		"bad-repo": {
			// modify the container path of an existing ref
			source:    strings.Replace(refs[0], owner, "notosbuild", 1),
			arch:      "amd64",
			errSubstr: fmt.Sprintf("notosbuild/%s: StatusCode: 404", reponame),
		},
		"bad-repo-containername": {
			// modify the container path of an existing ref
			source:    strings.Replace(refs[0], reponame, "container-does-not-exist", 1),
			arch:      "amd64",
			errSubstr: fmt.Sprintf("%s/container-does-not-exist: StatusCode: 404", owner),
		},
		"bad-tag": {
			// modify the tag of an existing ref
			source:    strings.Replace(refs[0], "tag", "not-a-tag", 1),
			arch:      "amd64",
			errSubstr: "error getting manifest: reading manifest not-a-tag-0",
		},
		"bad-arch": {
			source:    refs[0],
			arch:      "s390x",
			errSubstr: "no image found in manifest list for architecture \"s390x\"",
		},
		"tls-fail": {
			source:    refs[0],
			arch:      "amd64",
			tlsverify: true,
			errSubstr: "failed to verify certificate: x509: certificate signed by unknown authority",
		},
	}

	for name := range testCases {
		tc := testCases[name]
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			input := map[string]interface{}{
				"arch": tc.arch,
				"containers": []blueprint.Container{
					{
						Source:    tc.source,
						TLSVerify: &tc.tlsverify,
					},
				},
			}
			inputReq, err := json.Marshal(map[string]map[string]interface{}{
				"tree": input,
			})
			require.NoError(err)
			inpBuf := bytes.NewBuffer(inputReq)
			outBuf := &bytes.Buffer{}
			assert.ErrorContains(resolver.Run(inpBuf, outBuf), tc.errSubstr)
		})
	}
}
