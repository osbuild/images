package rhel8

import (
	"errors"

	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/distro/defs"
	"github.com/osbuild/images/pkg/distro/rhel"
)

func partitionTables(t *rhel.ImageType) (disk.PartitionTable, bool) {
	partitionTable, err := defs.PartitionTable(t, nil)
	if errors.Is(err, defs.ErrNoPartitionTableForImgType) {
		return disk.PartitionTable{}, false
	}
	if err != nil {
		panic(err)
	}
	if partitionTable == nil {
		return disk.PartitionTable{}, false
	}
	return *partitionTable, true
}

func defaultBasePartitionTables(t *rhel.ImageType) (disk.PartitionTable, bool) {
	return partitionTables(t)
}

func edgeBasePartitionTables(t *rhel.ImageType) (disk.PartitionTable, bool) {
	return partitionTables(t)
}

func ec2PartitionTables(t *rhel.ImageType) (disk.PartitionTable, bool) {
	return partitionTables(t)
}
