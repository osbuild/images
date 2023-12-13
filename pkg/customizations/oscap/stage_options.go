package oscap

import (
	"fmt"

	"github.com/osbuild/images/pkg/customizations/fsnode"
)

func CreateRequiredDirectories(createTailoring bool) ([]*fsnode.Directory, error) {
	var directories []*fsnode.Directory

	// although the osbuild stage will create this directory,
	// it's probably better to ensure that it is created here
	dataDirNode, err := fsnode.NewDirectory(dataDirPath, nil, nil, nil, true)
	if err != nil {
		return nil, fmt.Errorf("unexpected error creating OpenSCAP data directory: %s", err)
	}

	directories = append(directories, dataDirNode)

	if createTailoring {
		tailoringDirNode, err := fsnode.NewDirectory(tailoringDirPath, nil, nil, nil, true)
		if err != nil {
			return nil, fmt.Errorf("unexpected error creating OpenSCAP tailoring directory: %s", err)
		}

		directories = append(directories, tailoringDirNode)
	}

	return directories, nil
}
