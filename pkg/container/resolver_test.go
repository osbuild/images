package container_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/container"
)

func TestResolver(t *testing.T) {

	registry := NewTestRegistry()
	defer registry.Close()
	repo := registry.AddRepo("library/osbuild")
	ref := registry.GetRef("library/osbuild")

	refs := make([]string, 10)
	for i := 0; i < len(refs); i++ {
		checksum := repo.AddImage(
			[]Blob{NewDataBlobFromBase64(rootLayer)},
			[]string{"amd64", "ppc64le"},
			fmt.Sprintf("image %d", i),
			time.Time{})

		tag := fmt.Sprintf("%d", i)
		repo.AddTag(checksum, tag)
		refs[i] = fmt.Sprintf("%s:%s", ref, tag)
	}

	resolver := container.NewResolver("amd64")

	for _, r := range refs {
		resolver.Add(container.SourceSpec{
			r,
			"",
			common.ToPtr(""),
			common.ToPtr(false),
			false,
			nil,
		})
	}

	have, err := resolver.Finish()
	assert.NoError(t, err)
	assert.NotNil(t, have)

	assert.Len(t, have, len(refs))

	want := make([]container.Spec, len(refs))
	for i, r := range refs {
		spec, err := registry.Resolve(r, arch.ARCH_X86_64)
		assert.NoError(t, err)
		want[i] = spec
	}

	assert.ElementsMatch(t, have, want)
}

func TestResolverFail(t *testing.T) {
	resolver := container.NewResolver("amd64")

	resolver.Add(container.SourceSpec{
		"invalid-reference@${IMAGE_DIGEST}",
		"",
		common.ToPtr(""),
		common.ToPtr(false),
		false,
		nil,
	})
	specs, err := resolver.Finish()
	assert.Error(t, err)
	assert.Len(t, specs, 0)

	registry := NewTestRegistry()
	defer registry.Close()

	resolver.Add(container.SourceSpec{
		registry.GetRef("repo"),
		"",
		common.ToPtr(""),
		common.ToPtr(false),
		false,
		nil,
	})
	specs, err = resolver.Finish()
	assert.Error(t, err)
	assert.Len(t, specs, 0)

	resolver.Add(container.SourceSpec{
		registry.GetRef("repo"),
		"",
		common.ToPtr(""),
		common.ToPtr(false),
		false,
		nil,
	})
	specs, err = resolver.Finish()
	assert.Error(t, err)
	assert.Len(t, specs, 0)

	resolver.Add(container.SourceSpec{registry.GetRef("repo"),
		"",
		common.ToPtr(""),
		common.ToPtr(false),
		false,
		nil,
	})
	specs, err = resolver.Finish()
	assert.Error(t, err)
	assert.Len(t, specs, 0)
}
