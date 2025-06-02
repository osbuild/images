package manifest

import (
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/osbuild"
)

// filesystemConfigStages generates either an org.osbuild.fstab stage or a
// collection of org.osbuild.systemd.unit.create stages for .mount and .swap
// units (and an org.osbuild.systemd stage to enable them) depending on the
// pipeline configuration.
func filesystemConfigStages(pt *disk.PartitionTable, generate blueprint.GenerateMounts) ([]*osbuild.Stage, error) {
	var stages []*osbuild.Stage
	var err error
	switch generate {
	case blueprint.GenerateUnits:
		stages, err = osbuild.GenSystemdMountStages(pt)
		if err != nil {
			return nil, err
		}
	case blueprint.GenerateFstab:
		opts, err := osbuild.NewFSTabStageOptions(pt)
		if err != nil {
			return nil, err
		}
		stages = []*osbuild.Stage{osbuild.NewFSTabStage(opts)}
	case blueprint.GenerateNone:
		stages = []*osbuild.Stage{}
	}
	return stages, nil
}
