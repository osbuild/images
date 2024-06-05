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
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/osbuild"
)

type OtkGenPartitionInput struct {
	Options    *OtkPartOptions `json:"options"`
	Partitions []*OtkPartition `json:"partitions"`
}

type OtkPartOptions struct {
	UEFI *OtkPartUEFI `json:"uefi"`
	BIOS bool         `json:"bios"`
	Type string       `json:"type"`
	Size string       `json:"size"`
	UUID string       `json:"uuid"`

	SectorSize uint64 `json:"sector_size"`
}

type OtkPartUEFI struct {
	Size string `json:"size"`
}

type OtkPartition struct {
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

// XXX: review all struct names and make them consistent (OtkOutput*?)
type OtkGenPartitionsOutput struct {
	Const OtkGenPartConstOutput `json:"const"`
}

type OtkGenPartitionsInternal struct {
	PartitionTable *disk.PartitionTable `json:"partition-table"`
}

// "exported" view of partitions, this is an API so only add things here
// that are really needed and unlikely to change
type OtkPublicPartition struct {
	// not a UUID type because fat UUIDs are not compliant
	UUID string `json:"uuid"`
}

type OtkGenPartConstOutput struct {
	KernelOptsList []string `json:"kernel_opts_list"`
	// we generate this for convenience for otk users, so that they
	// can write, e.g. "filesystem.partition_map.boot.uuid"
	PartitionMap map[string]OtkPublicPartition `json:"partition_map"`
	Internal     OtkGenPartitionsInternal      `json:"internal"`
}

func makePartMap(pt *disk.PartitionTable) map[string]OtkPublicPartition {
	pm := make(map[string]OtkPublicPartition, len(pt.Partitions))
	// TODO: think about exposing more partitions, if we do, what labels
	// would we use? OtkPartition.Name? what about clashes with
	// "{r,b}oot" then?
	for _, part := range pt.Partitions {
		switch pl := part.Payload.(type) {
		case *disk.Filesystem:
			switch pl.Mountpoint {
			case "/":
				pm["root"] = OtkPublicPartition{
					UUID: pl.UUID,
				}
			case "/boot":
				pm["boot"] = OtkPublicPartition{
					UUID: pl.UUID,
				}
			}
		}
	}

	return pm
}

func makePartitionTableFromOtkInput(input *OtkGenPartitionInput) (*disk.PartitionTable, error) {
	pt := &disk.PartitionTable{
		UUID:       input.Options.UUID,
		Type:       input.Options.Type,
		SectorSize: input.Options.SectorSize,
	}
	if input.Options.BIOS {
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
	if input.Options.UEFI.Size != "" {
		uintSize, err := common.DataSizeToUint64(input.Options.UEFI.Size)
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

// Missing:
// 1. customizations^Wmodifications, e.g. extra partiton tables
// 2. refactor, make this nicer, it sucks a bit right now
func run(r io.Reader, rng *rand.Rand) (*OtkGenPartitionsOutput, error) {
	var genPartInput OtkGenPartitionInput
	if err := json.NewDecoder(r).Decode(&genPartInput); err != nil {
		return nil, err
	}

	basePt, err := makePartitionTableFromOtkInput(&genPartInput)
	if err != nil {
		return nil, err
	}

	pt, err := disk.NewPartitionTable(basePt, nil, 0, disk.DefaultPartitioningMode, nil, rng)
	if err != nil {
		return nil, err
	}

	kernelOptions := osbuild.GenImageKernelOptions(pt)
	otkPart := &OtkGenPartitionsOutput{
		Const: OtkGenPartConstOutput{
			Internal: OtkGenPartitionsInternal{
				PartitionTable: pt,
			},
			KernelOptsList: kernelOptions,
			PartitionMap:   makePartMap(pt),
		},
	}

	return otkPart, nil
}

func main() {
	rngSeed, err := cmdutil.SeedArgFor(&buildconfig.BuildConfig{}, "", "", "")
	if err != nil {
		// XXX: FIXME! helper
		panic(err)
	}
	source := rand.NewSource(rngSeed)
	// math/rand is good enough in this case
	/* #nosec G404 */
	rng := rand.New(source)

	output, err := run(os.Stdin, rng)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v", err.Error())
		os.Exit(1)
	}

	outputJson, err := json.Marshal(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v", err.Error())
		os.Exit(1)
	}
	fmt.Print(string(outputJson))
}
