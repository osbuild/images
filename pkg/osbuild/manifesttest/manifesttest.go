package manifesttest

import (
	"encoding/json"
	"fmt"
)

func pipelinesFrom(osbuildManifest []byte) ([]interface{}, error) {
	var manifest map[string]interface{}

	if err := json.Unmarshal(osbuildManifest, &manifest); err != nil {
		return nil, fmt.Errorf("cannot unmarshal manifest: %w", err)
	}
	if manifest["pipelines"] == nil {
		return nil, fmt.Errorf("cannot find any pipelines in %v", manifest)
	}
	pipelines, ok := manifest["pipelines"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("pipelines must be a list, got %T", pipelines)
	}
	return pipelines, nil
}

// PipelineNamesFrom will return all pipeline names from an osbuild
// json manifest. It will error on missing pipelines.
func PipelineNamesFrom(osbuildManifest []byte) ([]string, error) {
	pipelines, err := pipelinesFrom(osbuildManifest)
	if err != nil {
		return nil, err
	}

	pipelineNames := make([]string, len(pipelines))
	for idx, pi := range pipelines {
		pipelineNames[idx] = pi.(map[string]interface{})["name"].(string)
	}
	return pipelineNames, nil
}

// StagesForPipeline return the stages for the given a pipeline name. Only v2
// manifests are supported.
func StagesForPipeline(osbuildManifest []byte, searchedPipeline string) ([]string, error) {
	pipelines, err := pipelinesFrom(osbuildManifest)
	if err != nil {
		return nil, err
	}

	for _, pi := range pipelines {
		pipeline, ok := pi.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("pipeline must be an object, got %T", pi)
		}
		if pipeline["name"] == searchedPipeline {
			stageNames := make([]string, 0, len(pipeline))
			stages, ok := pipeline["stages"].([]interface{})
			if !ok {
				return nil, fmt.Errorf("stages must be a list, got %T", pipeline["stages"])
			}
			for _, stageIf := range stages {
				stage, ok := stageIf.(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("stage must be an object, got %T", stageIf)
				}
				stageNames = append(stageNames, stage["type"].(string))
			}
			return stageNames, nil
		}
	}

	return nil, fmt.Errorf("cannot find pipeline %q in %v", searchedPipeline, pipelines)
}

// Manifest is a unmarshalable version of osbuild.Manifest with extra
// debug helpers
type Manifest struct {
	Version   string     `json:"version"`
	Pipelines []Pipeline `json:"pipelines"`
	Sources   Sources    `json:"sources"`
}

func (m *Manifest) PipelineNames() []string {
	names := make([]string, len(m.Pipelines))
	for idx, pipeline := range m.Pipelines {
		names[idx] = pipeline.Name
	}
	return names
}

// Pipeline is a unmarshalable version of osbuild.Pipeline with extra
// debug helpers
type Pipeline struct {
	Name string `json:"name,omitempty"`
	// The build environment which can run this pipeline
	Build string `json:"build,omitempty"`

	Runner string `json:"runner,omitempty"`

	// Sequence of stages that produce the filesystem tree, which is the
	// payload of the produced image.
	Stages []*Stage `json:"stages,omitempty"`
}

func (p *Pipeline) Stage(typ string) *Stage {
	for _, stage := range p.Stages {
		if stage.Type == typ {
			return stage
		}
	}
	return nil
}

// Stage is a unmarshalable version of osbuild.Stage with extra
// debug helpers
type Stage struct {
	Type string `json:"type"`

	Inputs  json.RawMessage `json:"inputs,omitempty"`
	Options json.RawMessage `json:"options,omitempty"`
}

type Sources map[string]Source

type Source map[string]any

// NewManifestFromBytes uses the manifesttest data structures to
// unmarshal
func NewManifestFromBytes(osbuildManifest []byte) (*Manifest, error) {
	var mani Manifest
	if err := json.Unmarshal(osbuildManifest, &mani); err != nil {
		return nil, err
	}
	return &mani, nil
}
