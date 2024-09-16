package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
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

const rootLayer = `H4sIAAAJbogA/+SWUYqDMBCG53lP4V5g9x8dzRX2Bvtc0VIhEIhKe/wSKxgU6ktjC/O9hMzAQDL8
/8yltdb9DLeB0gEGKhHCg/UJsBAL54zKFBAC54ZzyrCUSMfYDydPgHfu6R/s5VePilOfzF/of/bv
vG2+lqhyFNGPddP53yjyegCBKcuNROZ77AmBoP+CmbIyqpEM5fqf+3/ubJtsCuz7P1b+L1Du/4f5
v+vrsVPu/Vq9P3ANk//d+x/MZv8TKNf/Qfqf9v9v5fLXK3/lKEc5ypm4AwAA//8DAE6E6nIAEgAA
`

func createTestRegistry() (*testregistry.Registry, []string) {
	registry := testregistry.New()
	repo := registry.AddRepo("library/osbuild")
	ref := registry.GetRef("library/osbuild")

	// add 10 images, all in the same repository with the same content
	// (rootLayer), but each with a different tag and comment
	refs := make([]string, 10)
	for idx := 0; idx < len(refs); idx++ {
		checksum := repo.AddImage(
			[]testregistry.Blob{testregistry.NewDataBlobFromBase64(rootLayer)},
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

	require := require.New(t)
	assert := assert.New(t)

	inpContainers := make([]blueprint.Container, len(refs))
	for idx, ref := range refs {
		inpContainers[idx] = blueprint.Container{
			Source:       ref,
			Name:         fmt.Sprintf("test/localhost/%s", ref), // add a prefix for the local name to override the source
			TLSVerify:    common.ToPtr(false),
			LocalStorage: false,
		}
	}

	amd64input := map[string]interface{}{
		"arch":       "amd64",
		"containers": inpContainers,
	}
	inputReq, err := json.Marshal(map[string]map[string]interface{}{
		"tree": amd64input,
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
		spec, err := registry.Resolve(ref, arch.ARCH_X86_64)
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
}
