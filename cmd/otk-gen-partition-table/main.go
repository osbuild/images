package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"

	"github.com/osbuild/images/internal/buildconfig"
	"github.com/osbuild/images/internal/cmdutil"
	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/internal/otk"
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/osbuild"
)

type Input struct {
	Properties InputProperties   `json:"properties"`
	Partitions []*InputPartition `json:"partitions"`

	Modifications InputModifications `json:"modifications"`
}

type InputProperties struct {
	UEFI        InputUEFI `json:"uefi"`
	BIOS        bool      `json:"bios"`
	Type        string    `json:"type"`
	DefaultSize string    `json:"default_size"`
	UUID        string    `json:"uuid"`

	SectorSize uint64 `json:"sector_size"`
}

type InputUEFI struct {
	Size string `json:"size"`
}

type InputPartition struct {
	Name       string `json:"name"`
	Mountpoint string `json:"mountpoint"`
	Label      string `json:"label"`
	Size       string `json:"size"`
	Type       string `json:"type"`
	UUID       string `json:"uuid"`
	FSMntOps   string `json:"fs_mntops"`
	FSFreq     uint64 `json:"fs_freq"`
	FSPassNo   uint64 `json:"fs_passno"`

	// TODO: add sectorlvm,luks, see https://github.com/achilleas-k/images/pull/2#issuecomment-2136025471
}

type InputModifications struct {
	PartitionMode disk.PartitioningMode               `json:"partition_mode"`
	Filesystems   []blueprint.FilesystemCustomization `json:"filesystems"`
	MinDiskSize   string                              `json:"min_disk_size"`
}

type Output struct {
	Const OutputConst `json:"const"`
}

type OutputConst struct {
	KernelOptsList []string `json:"kernel_opts_list"`
	// we generate this for convenience for otk users, so that they
	// can write, e.g. "filesystem.partition_map.boot.uuid"
	PartitionMap map[string]OutputPartition `json:"partition_map"`
	Internal     OutputInternal             `json:"internal"`
}

// "exported" view of partitions, this is an API so only add things here
// that are really needed and unlikely to change
type OutputPartition struct {
	// not a UUID type because fat UUIDs are not compliant
	UUID string `json:"uuid"`
}

type OutputInternal = otk.PartitionInternal

func makePartMap(pt *disk.PartitionTable) map[string]OutputPartition {
	pm := make(map[string]OutputPartition, len(pt.Partitions))
	// TODO: think about exposing more partitions, if we do, what labels
	// would we use? ition.Name? what about clashes with
	// "{r,b}oot" then?
	for _, part := range pt.Partitions {
		switch pl := part.Payload.(type) {
		case *disk.Filesystem:
			switch pl.Mountpoint {
			case "/":
				pm["root"] = OutputPartition{
					UUID: pl.UUID,
				}
			case "/boot":
				pm["boot"] = OutputPartition{
					UUID: pl.UUID,
				}
			}
		}
	}

	return pm
}

func makePartitionTableFromOtkInput(input *Input) (*disk.PartitionTable, error) {
	pt := &disk.PartitionTable{
		UUID:       input.Properties.UUID,
		Type:       input.Properties.Type,
		SectorSize: input.Properties.SectorSize,
	}
	if input.Properties.BIOS {
		if len(pt.Partitions) > 0 {
			panic("internal error: bios partition *must* go first")
		}
		pt.Partitions = append(pt.Partitions, disk.Partition{
			Size:     1 * common.MebiByte,
			Bootable: true,
			Type:     disk.BIOSBootPartitionGUID,
			UUID:     disk.BIOSBootPartitionUUID,
		})
	}
	if input.Properties.UEFI.Size != "" {
		uintSize, err := common.DataSizeToUint64(input.Properties.UEFI.Size)
		if err != nil {
			return nil, err
		}
		if uintSize > 0 {
			pt.Partitions = append(pt.Partitions, disk.Partition{
				Size: uintSize,
				Type: disk.EFISystemPartitionGUID,
				UUID: disk.EFISystemPartitionUUID,
				Payload: &disk.Filesystem{
					Type:         "vfat",
					UUID:         disk.EFIFilesystemUUID,
					Mountpoint:   "/boot/efi",
					Label:        "EFI-SYSTEM",
					FSTabOptions: "defaults,uid=0,gid=0,umask=077,shortname=winnt",
					FSTabFreq:    0,
					FSTabPassNo:  2,
				},
			})
		}
	}

	for _, part := range input.Partitions {
		uintSize, err := common.DataSizeToUint64(part.Size)
		if err != nil {
			return nil, err
		}
		pt.Partitions = append(pt.Partitions, disk.Partition{
			Size: uintSize,
			// XXX: support lvm,luks here
			Payload: &disk.Filesystem{
				Label:        part.Label,
				Type:         part.Type,
				UUID:         part.UUID,
				Mountpoint:   part.Mountpoint,
				FSTabOptions: part.FSMntOps,
				FSTabFreq:    part.FSFreq,
				FSTabPassNo:  part.FSPassNo,
			},
		})
	}

	return pt, nil
}

func getDiskSizeFrom(input *Input) (diskSize uint64, err error) {
	var defaultSize, modMinSize uint64

	if input.Properties.DefaultSize != "" {
		defaultSize, err = common.DataSizeToUint64(input.Properties.DefaultSize)
		if err != nil {
			return 0, err
		}
	}
	if input.Modifications.MinDiskSize != "" {
		modMinSize, err = common.DataSizeToUint64(input.Modifications.MinDiskSize)
		if err != nil {
			return 0, err
		}
	}
	// TODO: use max() once we move to go1.21
	if defaultSize > modMinSize {
		return defaultSize, nil
	}
	return modMinSize, nil
}

func genPartitionTable(genPartInput *Input, rng *rand.Rand) (*Output, error) {
	basePt, err := makePartitionTableFromOtkInput(genPartInput)
	if err != nil {
		return nil, err
	}
	diskSize, err := getDiskSizeFrom(genPartInput)
	if err != nil {
		return nil, fmt.Errorf("cannot get the disk size: %w", err)
	}
	pt, err := disk.NewPartitionTable(basePt, genPartInput.Modifications.Filesystems, diskSize, genPartInput.Modifications.PartitionMode, nil, rng)
	if err != nil {
		return nil, err
	}

	kernelOptions := osbuild.GenImageKernelOptions(pt)
	otkPart := &Output{
		Const: OutputConst{
			Internal: OutputInternal{
				PartitionTable: pt,
			},
			KernelOptsList: kernelOptions,
			PartitionMap:   makePartMap(pt),
		},
	}

	return otkPart, nil
}

func run(r io.Reader, w io.Writer) error {
	rngSeed, err := cmdutil.SeedArgFor(&buildconfig.BuildConfig{}, "", "", "")
	if err != nil {
		return err
	}
	// math/rand is good enough in this case
	/* #nosec G404 */
	rng := rand.New(rand.NewSource(rngSeed))

	var genPartInput Input
	if err := json.NewDecoder(r).Decode(&genPartInput); err != nil {
		return err
	}
	// XXX: validate inputs, right now an empty "type" is not an error
	// but it should either be an error or we should set a default

	output, err := genPartitionTable(&genPartInput, rng)
	if err != nil {
		return fmt.Errorf("cannot generate partition table: %w", err)
	}
	// there is no need to output "nice" json, but it does make testing
	// simpler
	outputJson, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot marshal response: %w", err)
	}
	fmt.Fprintf(w, "%s\n", outputJson)
	return nil
}

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v", err.Error())
		os.Exit(1)
	}
}
