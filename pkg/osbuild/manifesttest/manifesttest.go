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
