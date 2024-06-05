package otk

import (
	"github.com/osbuild/images/pkg/disk"
)

type PartitionInternal struct {
	PartitionTable *disk.PartitionTable `json:"partition-table"`
}
