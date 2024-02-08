package osbuild

import (
	"strings"

	"github.com/osbuild/images/pkg/disk"
)

// GenMkfsStages generates a list of org.mkfs.* stages based on a
// partition table description for a single device node
func GenMkfsStages(pt *disk.PartitionTable, device *Device) []*Stage {
	stages := make([]*Stage, 0, len(pt.Partitions))

	// assume loopback device for simplicity since it's the only one currently supported
	// panic if the conversion fails
	devOptions, ok := device.Options.(*LoopbackDeviceOptions)
	if !ok {
		panic("GenMkfsStages: failed to convert device options to loopback options")
	}

	genStage := func(e disk.Entity, path []disk.Entity) error {
		stageDevices, lastName := getDevices(path, devOptions.Filename, true)

		// the last device on the PartitionTable must be named "device"
		lastDevice := stageDevices[lastName]
		delete(stageDevices, lastName)
		stageDevices["device"] = lastDevice

		// firstly, handle btrfs (disk.Btrfs isn't disk.Mountable)
		if btrfs, isBtrfs := e.(*disk.Btrfs); isBtrfs {
			options := &MkfsBtrfsStageOptions{
				UUID:  btrfs.UUID,
				Label: btrfs.Label,
			}
			stage := NewMkfsBtrfsStage(options, stageDevices)
			stages = append(stages, stage)

			return nil
		}

		mnt, isMountable := e.(disk.Mountable)
		if !isMountable {
			// if the thing is not mountable nor btrfs, there's no fs to be created
			return nil
		}

		var stage *Stage
		t := mnt.GetFSType()
		fsSpec := mnt.GetFSSpec()
		switch t {
		case "xfs":
			options := &MkfsXfsStageOptions{
				UUID:  fsSpec.UUID,
				Label: fsSpec.Label,
			}
			stage = NewMkfsXfsStage(options, stageDevices)
		case "vfat":
			options := &MkfsFATStageOptions{
				VolID: strings.Replace(fsSpec.UUID, "-", "", -1),
			}
			stage = NewMkfsFATStage(options, stageDevices)
		case "btrfs":
			// A mountable btrfs entity means that it's a subvolume, no need to mkfs it.
			// Subvolumes are handled separately
			return nil
		case "ext4":
			options := &MkfsExt4StageOptions{
				UUID:  fsSpec.UUID,
				Label: fsSpec.Label,
			}
			stage = NewMkfsExt4Stage(options, stageDevices)
		default:
			panic("unknown fs type " + t)
		}
		stages = append(stages, stage)

		return nil
	}

	_ = pt.ForEachEntity(genStage) // genStage always returns nil
	return stages
}
