package otkdisk

import (
	"fmt"

	"golang.org/x/exp/slices"

	"github.com/osbuild/images/pkg/disk"
)

// Data contains the full description of the partition table as well as extra
// options and a PartitionMap for easier access. The data under Const should
// not be modified by a consumer of this data structure.
type Data struct {
	Const Const `json:"const"`
}

// Const contains partition table data that is considered "constant",
// i.e.  that should not be modified by the consumer as there may be
// inter-dependencies between the values
type Const struct {
	KernelOptsList []string `json:"kernel_opts_list"`

	// PartitionMap is generated for convenient indexing of certain partitions
	// with predictable names in otk, such as
	// "filesystem.partition_map.boot.uuid"
	PartitionMap map[string]Partition `json:"partition_map"`

	// Internal representation of the full partition table. The representation
	// is internal to the partition tools and should not be used by otk
	// directly. It makes no external API guarantees about the content or
	// structure.
	Internal Internal `json:"internal"`

	// Filename for the disk image.
	Filename string `json:"filename"`
}

// Partition represents an exported view of a partition. This is an API so only
// add things here that are necessary for convenient external access and
// unlikely to change.
type Partition struct {
	// NOTE: Not a UUID type because fat UUIDs are not compliant
	UUID string `json:"uuid"`
}

// Interal contains partition table data that is stricly internal and
// may change in non-backward compatible ways. No "otk" manifest
// should ever use this directly, it's strictly meant for the
// "otk-{gen,make}-*" tools for their data exchange.
type Internal struct {
	PartitionTable *disk.PartitionTable `json:"partition-table"`
}

// PartType represents a partition type
type PartType string

const (
	PartTypeUnset PartType = ""
	PartTypeGPT   PartType = "gpt"
	PartTypeDOS   PartType = "dos"
)

// Validate validates that the given PartType is valid
func (p PartType) Validate() error {
	if !slices.Contains([]PartType{PartTypeGPT, PartTypeDOS}, p) {
		return fmt.Errorf("unsupported partition type %q", p)
	}
	return nil
}
