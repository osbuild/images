package osbuild_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/pkg/container"
	"github.com/osbuild/images/pkg/osbuild"
)

func TestNewContainersInputForSource(t *testing.T) {
	expectedJson := `{
  "type": "org.osbuild.containers",
  "origin": "org.osbuild.source",
  "references": {
    "id1": {
      "name": "local-name1"
    },
    "id2": {
      "name": "local-name2"
    }
  }
}`
	containerInputs := osbuild.NewContainersInputForSources([]container.Spec{
		{
			ImageID:   "id1",
			LocalName: "local-name1",
		},
		{
			ImageID:   "id2",
			LocalName: "local-name2",
		},
	})
	json, err := json.MarshalIndent(containerInputs, "", "  ")
	require.Nil(t, err)
	assert.Equal(t, string(json), expectedJson)
}

func TestNewLocalContainersInputForSource(t *testing.T) {
	expectedJson := `{
  "type": "org.osbuild.containers-storage",
  "origin": "org.osbuild.source",
  "references": {
    "id1": {
      "name": "local-name1"
    },
    "id2": {
      "name": "local-name2"
    }
  }
}`
	containerInputs := osbuild.NewLocalContainersInputForSources([]container.Spec{
		{
			ImageID:      "id1",
			LocalName:    "local-name1",
			LocalStorage: true,
		},
		{
			ImageID:      "id2",
			LocalName:    "local-name2",
			LocalStorage: true,
		},
	})
	json, err := json.MarshalIndent(containerInputs, "", "  ")
	require.Nil(t, err)
	assert.Equal(t, string(json), expectedJson)
}
