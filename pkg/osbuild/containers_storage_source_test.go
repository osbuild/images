package osbuild

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewContainersStorageSource(t *testing.T) {
	imageID := "sha256:c2ecf25cf190e76b12b07436ad5140d4ba53d8a136d498705e57a006837a720f"

	source := NewContainersStorageSource()

	source.AddItem(imageID)
	assert.Len(t, source.Items, 1)

	_, ok := source.Items[imageID]
	assert.True(t, ok)

	imageID = "sha256:d2ab8fea7f08a22f03b30c13c6ea443121f25e87202a7496e93736efa6fe345a"

	source.AddItem(imageID)
	assert.Len(t, source.Items, 2)
	_, ok = source.Items[imageID]
	assert.True(t, ok)

	// empty image id
	assert.PanicsWithError(t, `item "" has invalid image id`, func() {
		source.AddItem("")
	})

	// invalid image id
	assert.PanicsWithError(t, `item "sha256:foo" has invalid image id`, func() {
		source.AddItem("sha256:foo")
	})
}
