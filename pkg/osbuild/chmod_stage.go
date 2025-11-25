package osbuild

type ChmodStageOptions struct {
	Items map[string]ChmodStagePathOptions `json:"items"`
}

type ChmodStagePathOptions struct {
	Mode      string `json:"mode"`
	Recursive bool   `json:"recursive,omitempty"`
}

func (ChmodStageOptions) isStageOptions() {}

var _ = PathChanger(ChmodStageOptions{})

// NewChmodStage creates a new org.osbuild.chmod stage
func NewChmodStage(options *ChmodStageOptions) *Stage {
	return &Stage{
		Type:    "org.osbuild.chmod",
		Options: options,
	}
}

func (c ChmodStageOptions) PathsChanged() []string {
	var paths []string
	for path := range c.Items {
		paths = append(paths, path)
	}
	return paths
}
