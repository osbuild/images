// Package osbuild provides primitives for representing and (un)marshalling
// OSBuild types.
package osbuild

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPipeline_AddStage(t *testing.T) {
	expectedPipeline := &Pipeline{
		Build: "name:build",
		Stages: []*Stage{
			{
				Type: "org.osbuild.rpm",
			},
		},
	}
	actualPipeline := &Pipeline{
		Build: "name:build",
	}
	actualPipeline.AddStage(&Stage{
		Type: "org.osbuild.rpm",
	})
	assert.Equal(t, expectedPipeline, actualPipeline)
	assert.Equal(t, 1, len(actualPipeline.Stages))
}

var fakeOsbuildManifestWithIdentifiers = []byte(`{
  "version": "2",
  "pipelines": [
    {
       "name": "build",
       "stages": [
         {
			"id": "1234",
            "type": "org.osbuild.rpm"
         },
         {
			"id": "5678",
            "type": "org.osbuild.mkdir"
         }
       ]
    }
  ]
}`)

func TestManifestFromBytes(t *testing.T) {
	manifest, err := NewManifestFromBytes(fakeOsbuildManifestWithIdentifiers)
	assert.NoError(t, err)

	assert.Equal(t, manifest.Pipelines[0].Stages[0].ID, "1234")
	assert.Equal(t, manifest.Pipelines[0].Stages[1].ID, "5678")

	pID, err := manifest.Pipelines[0].GetID()
	assert.NoError(t, err)

	assert.Equal(t, pID, "5678")
}
