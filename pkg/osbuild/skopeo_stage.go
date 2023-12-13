package osbuild

type SkopeoDestinationContainersStorage struct {
	Type          string `json:"type"`
	StoragePath   string `json:"storage-path,omitempty"`
	StorageDriver string `json:"storage-driver,omitempty"`
}

type SkopeoStageOptions struct {
	DestinationContainersStorage SkopeoDestinationContainersStorage `json:"destination"`
}

func (o SkopeoStageOptions) isStageOptions() {}

type SkopeoStageInputs struct {
	Images        ContainersInput `json:"images"`
	ManifestLists *FilesInput     `json:"manifest-lists,omitempty"`
}

func (SkopeoStageInputs) isStageInputs() {}

func NewSkopeoStageWithContainersStorage(path string, images ContainersInput, manifests *FilesInput) *Stage {

	inputs := SkopeoStageInputs{
		Images:        images,
		ManifestLists: manifests,
	}

	return &Stage{
		Type: "org.osbuild.skopeo",
		Options: &SkopeoStageOptions{
			DestinationContainersStorage: SkopeoDestinationContainersStorage{
				Type:        "containers-storage",
				StoragePath: path,
			},
		},
		Inputs: inputs,
	}
}
