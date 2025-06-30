package osbuild

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGzipStageOptions(t *testing.T) {
	filename := "image.raw.gz"

	expectedOptions := &GzipStageOptions{
		Filename: filename,
	}

	actualOptions := NewGzipStageOptions(filename)
	assert.Equal(t, expectedOptions, actualOptions)
}

func TestNewGzipStage(t *testing.T) {
	inputFilename := "image.raw"
	filename := "image.raw.gz"
	pipeline := "os"

	expectedStage := &Stage{
		Type:    "org.osbuild.gzip",
		Options: NewGzipStageOptions(filename),
		Inputs:  NewGzipStageInputs(NewFilesInputPipelineObjectRef(pipeline, inputFilename, nil)),
	}

	actualStage := NewGzipStage(NewGzipStageOptions(filename),
		NewGzipStageInputs(NewFilesInputPipelineObjectRef(pipeline, inputFilename, nil)))
	assert.Equal(t, expectedStage, actualStage)
}

func TestNewGzipStageNoInputs(t *testing.T) {
	filename := "image.raw.gz"

	expectedStage := &Stage{
		Type:    "org.osbuild.gzip",
		Options: &GzipStageOptions{Filename: filename},
		Inputs:  nil,
	}

	actualStage := NewGzipStage(&GzipStageOptions{Filename: filename}, nil)
	assert.Equal(t, expectedStage, actualStage)
}
