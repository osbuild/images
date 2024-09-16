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
	"github.com/osbuild/images/internal/otkdisk"
	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/disk"
	"github.com/osbuild/images/pkg/osbuild"
)

// Tree is the wrapper aroudn the "real" input
type Tree struct {
	Tree Input `json:"tree"`
}

// Input represents the user provided inputs that will be used
// to generate the partition table
type Input struct {
	Properties InputProperties   `json:"properties"`
	Partitions []*InputPartition `json:"partitions"`

	Modifications InputModifications `json:"modifications"`
}

// InputProperties contains global properties of the partition table
type InputProperties struct {
	Type        otkdisk.PartType `json:"type"`
	DefaultSize string           `json:"default_size"`
	UUID        string           `json:"uuid"`
	StartOffset string           `json:"start_offset"`
	Create      InputCreate      `json:"create"`

	SectorSize uint64 `json:"sector_size"`
}

// InputCreate contains details what partitions to auto-generate
type InputCreate struct {
	BIOSBootPartition bool   `json:"bios_boot_partition"`
	EspPartition      bool   `json:"esp_partition"`
	EspPartitionSize  string `json:"esp_partition_size"`
}

// InputPartition represents a single user provided partition input
type InputPartition struct {
	Name       string `json:"name"`
	Mountpoint string `json:"mountpoint"`
	Label      string `json:"label"`
	Size       string `json:"size"`
	Type       string `json:"type"`
	PartUUID   string `json:"part_uuid"`
	PartType   string `json:"part_type"`
	FsUUID     string `json:"fs_uuid"`
	FSMntOps   string `json:"fs_mntops"`
	FSFreq     uint64 `json:"fs_freq"`
	FSPassNo   uint64 `json:"fs_passno"`

	// TODO: add sectorlvm,luks, see https://github.com/achilleas-k/images/pull/2#issuecomment-2136025471
}

// InputModifications allow modifiying the partition generation to e.g.
// increase the default disk size
type InputModifications struct {
	PartitionMode disk.PartitioningMode               `json:"partition_mode"`
	Filesystems   []blueprint.FilesystemCustomization `json:"filesystems"`
	MinDiskSize   string                              `json:"min_disk_size"`
	Filename      string                              `json:"filename"`
}

// Output contains a full description of a disk, this can be consumed
// by other tools like otk-make-*
type Output = otkdisk.Data

func makePartMap(pt *disk.PartitionTable) map[string]otkdisk.Partition {
	pm := make(map[string]otkdisk.Partition, len(pt.Partitions))
	// TODO: think about exposing more partitions, if we do, what labels
	// would we use? ition.Name? what about clashes with
	// "{r,b}oot" then?
	for _, part := range pt.Partitions {
		switch pl := part.Payload.(type) {
		case *disk.Filesystem:
			switch pl.Mountpoint {
			case "/":
				pm["root"] = otkdisk.Partition{
					UUID: pl.UUID,
				}
			case "/boot":
				pm["boot"] = otkdisk.Partition{
					UUID: pl.UUID,
				}
			}
		}
	}

	return pm
}

func validateInput(input *Input) error {
	// TODO: validate more
	if err := input.Properties.Type.Validate(); err != nil {
		return err
	}
	return nil
}

func makePartitionTableFromOtkInput(input *Input) (*disk.PartitionTable, error) {
	var err error

	if err := validateInput(input); err != nil {
		return nil, fmt.Errorf("cannot validate inputs: %w", err)
	}

	var startOffset uint64
	if input.Properties.StartOffset != "" {
		startOffset, err = common.DataSizeToUint64(input.Properties.StartOffset)
		if err != nil {
			return nil, err
		}
	}

	pt := &disk.PartitionTable{
		UUID:        input.Properties.UUID,
		Type:        string(input.Properties.Type),
		SectorSize:  input.Properties.SectorSize,
		StartOffset: startOffset,
	}
	if input.Properties.Create.BIOSBootPartition {
		if len(pt.Partitions) > 0 {
			return nil, fmt.Errorf("internal error: bios partition *must* go first")
		}
		pt.Partitions = append(pt.Partitions, disk.Partition{
			Size:     1 * common.MebiByte,
			Bootable: true,
			Type:     disk.BIOSBootPartitionGUID,
			UUID:     disk.BIOSBootPartitionUUID,
		})
	}
	if input.Properties.Create.EspPartition {
		size := input.Properties.Create.EspPartitionSize
		if size == "" {
			size = "1 GiB"
		}
		uintSize, err := common.DataSizeToUint64(size)
		if err != nil {
			return nil, err
		}
		if uintSize > 0 {
			part := disk.Partition{
				Size: uintSize,
				Payload: &disk.Filesystem{
					Type:         "vfat",
					UUID:         disk.EFIFilesystemUUID,
					Mountpoint:   "/boot/efi",
					Label:        "EFI-SYSTEM",
					FSTabOptions: "defaults,uid=0,gid=0,umask=077,shortname=winnt",
					FSTabFreq:    0,
					FSTabPassNo:  2,
				},
			}
			switch input.Properties.Type {
			case otkdisk.PartTypeGPT:
				part.Type = disk.EFISystemPartitionGUID
				part.UUID = disk.EFISystemPartitionUUID
			case otkdisk.PartTypeDOS:
				part.Bootable = true
				part.Type = disk.DosFat16B
			default:
				return nil, fmt.Errorf("unsupported partition type: %v", input)
			}
			pt.Partitions = append(pt.Partitions, part)
		}
	}

	for _, part := range input.Partitions {
		uintSize, err := common.DataSizeToUint64(part.Size)
		if err != nil {
			return nil, err
		}
		pt.Partitions = append(pt.Partitions, disk.Partition{
			Size: uintSize,
			UUID: part.PartUUID,
			Type: part.PartType,
			// XXX: support lvm,luks here
			Payload: &disk.Filesystem{
				Label:        part.Label,
				Type:         part.Type,
				Mountpoint:   part.Mountpoint,
				UUID:         part.FsUUID,
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
	fname := "disk.img"
	if genPartInput.Modifications.Filename != "" {
		fname = genPartInput.Modifications.Filename
	}

	kernelOptions := osbuild.GenImageKernelOptions(pt)
	otkPart := &Output{
		Const: otkdisk.Const{
			Internal: otkdisk.Internal{
				PartitionTable: pt,
			},
			KernelOptsList: kernelOptions,
			PartitionMap:   makePartMap(pt),
			Filename:       fname,
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

	var genPartInputTree Tree
	if err := json.NewDecoder(r).Decode(&genPartInputTree); err != nil {
		return err
	}
	// XXX: validate inputs, right now an empty "type" is not an error
	// but it should either be an error or we should set a default

	generated, err := genPartitionTable(&genPartInputTree.Tree, rng)
	if err != nil {
		return fmt.Errorf("cannot generate partition table: %w", err)
	}
	// there is no need to output "nice" json, but it does make testing
	// simpler
	output := map[string]interface{}{
		"tree": generated,
	}
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
