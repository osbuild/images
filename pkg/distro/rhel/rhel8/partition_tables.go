package rhel8

import (
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/distro/defs"
	"github.com/osbuild/images/pkg/distro/rhel"
)

func defaultBasePartitionTables(t *rhel.ImageType) (disk.PartitionTable, bool) {
	partitionTable, err := defs.PartitionTable(t)
	if err != nil {
		// XXX: have a check to differenciate ErrNoEnt and else
		return disk.PartitionTable{}, false
	}
	if partitionTable == nil {
		return disk.PartitionTable{}, false
	}

	return *partitionTable, true
}

func edgeBasePartitionTables(t *rhel.ImageType) (disk.PartitionTable, bool) {
	return defaultBasePartitionTables(t)
}

func ec2PartitionTables(t *rhel.ImageType) (disk.PartitionTable, bool) {
	return defaultBasePartitionTables(t)
}
