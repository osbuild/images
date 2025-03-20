package disk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/osbuild/images/pkg/datasizes"
)

// Default physical extent size in bytes: logical volumes
// created inside the VG will be aligned to this.
const LVMDefaultExtentSize = 4 * datasizes.MebiByte

type LVMVolumeGroup struct {
	Name        string
	Description string

	LogicalVolumes []LVMLogicalVolume `json:"logical_volumes"`
}

func init() {
	payloadEntityMap["lvm"] = reflect.TypeOf(LVMVolumeGroup{})
}

func (vg *LVMVolumeGroup) EntityName() string {
	return "lvm"
}

func (vg *LVMVolumeGroup) Clone() Entity {
	if vg == nil {
		return nil
	}

	clone := &LVMVolumeGroup{
		Name:           vg.Name,
		Description:    vg.Description,
		LogicalVolumes: make([]LVMLogicalVolume, len(vg.LogicalVolumes)),
	}

	for idx, lv := range vg.LogicalVolumes {
		ent := lv.Clone()

		// lv.Clone() will return nil only if the logical volume is nil
		if ent == nil {
			panic(fmt.Sprintf("logical volume %d in a LVM volume group is nil; this is a programming error", idx))
		}

		lv, cloneOk := ent.(*LVMLogicalVolume)
		if !cloneOk {
			panic("LVMLogicalVolume.Clone() returned an Entity that cannot be converted to *LVMLogicalVolume; this is a programming error")
		}

		clone.LogicalVolumes[idx] = *lv
	}

	return clone
}

func (vg *LVMVolumeGroup) GetItemCount() uint {
	if vg == nil {
		return 0
	}
	return uint(len(vg.LogicalVolumes))
}

func (vg *LVMVolumeGroup) GetChild(n uint) Entity {
	if vg == nil {
		panic("LVMVolumeGroup.GetChild: nil entity")
	}
	return &vg.LogicalVolumes[n]
}

func (vg *LVMVolumeGroup) CreateMountpoint(mountpoint string, size uint64) (Entity, error) {

	filesystem := Filesystem{
		Type:         "xfs",
		Mountpoint:   mountpoint,
		FSTabOptions: "defaults",
		FSTabFreq:    0,
		FSTabPassNo:  0,
	}

	// leave lv name empty to autogenerate based on mountpoint
	return vg.CreateLogicalVolume("", size, &filesystem)
}

// genLVName generates a valid logical volume name from a mountpoint or base
// that does not conflict with existing ones.
func (vg *LVMVolumeGroup) genLVName(base string) (string, error) {
	names := make(map[string]bool, len(vg.LogicalVolumes))
	for _, lv := range vg.LogicalVolumes {
		names[lv.Name] = true
	}

	base = lvname(base) // if the mountpoint is used (i.e. if the base contains /), sanitize it and append 'lv'

	// Make sure that we don't collide with an existing volume, e.g.
	// 'home/test' and /home_test would collide.
	return genUniqueString(base, names)
}

// CreateLogicalVolume creates a new logical volume on the volume group. If a
// name is not provided, a valid one is generated based on the payload
// mountpoint. If a name is provided, it is used directly without validating.
func (vg *LVMVolumeGroup) CreateLogicalVolume(lvName string, size uint64, payload Entity) (*LVMLogicalVolume, error) {
	if vg == nil {
		panic("LVMVolumeGroup.CreateLogicalVolume: nil entity")
	}

	if lvName == "" {
		// generate a name based on the payload's mountpoint
		switch ent := payload.(type) {
		case Mountable:
			lvName = ent.GetMountpoint()
		case *Swap:
			lvName = "swap"
		default:
			return nil, fmt.Errorf("could not create logical volume: no name provided and payload %T is not mountable or swap", payload)
		}
		autoName, err := vg.genLVName(lvName)
		if err != nil {
			return nil, err
		}
		lvName = autoName
	}

	lv := LVMLogicalVolume{
		Name:    lvName,
		Size:    vg.AlignUp(size),
		Payload: payload,
	}

	vg.LogicalVolumes = append(vg.LogicalVolumes, lv)

	return &vg.LogicalVolumes[len(vg.LogicalVolumes)-1], nil
}

func alignUp(size uint64) uint64 {
	if size%LVMDefaultExtentSize != 0 {
		size += LVMDefaultExtentSize - size%LVMDefaultExtentSize
	}

	return size
}

func (vg *LVMVolumeGroup) AlignUp(size uint64) uint64 {
	return alignUp(size)
}

func (vg *LVMVolumeGroup) MetadataSize() uint64 {
	if vg == nil {
		return 0
	}

	// LVM2 allows for a lot of customizations that will affect the size
	// of the metadata and its location and thus the start of the physical
	// extent. For now we assume the default which results in a start of
	// the physical extent 1 MiB
	return 1 * datasizes.MiB
}

func (vg *LVMVolumeGroup) minSize(size uint64) uint64 {
	var lvsum uint64
	for _, lv := range vg.LogicalVolumes {
		lvsum += lv.Size
	}
	minSize := lvsum + vg.MetadataSize()

	if minSize > size {
		size = minSize
	}

	return vg.AlignUp(size)
}

type LVMLogicalVolume struct {
	Name    string
	Size    uint64
	Payload Entity
}

func (lv *LVMLogicalVolume) Clone() Entity {
	if lv == nil {
		return nil
	}
	return &LVMLogicalVolume{
		Name:    lv.Name,
		Size:    lv.Size,
		Payload: lv.Payload.Clone(),
	}
}

func (lv *LVMLogicalVolume) GetItemCount() uint {
	if lv == nil || lv.Payload == nil {
		return 0
	}
	return 1
}

func (lv *LVMLogicalVolume) GetChild(n uint) Entity {
	if n != 0 || lv == nil {
		panic(fmt.Sprintf("invalid child index for LVMLogicalVolume: %d != 0", n))
	}
	return lv.Payload
}

func (lv *LVMLogicalVolume) GetSize() uint64 {
	if lv == nil {
		return 0
	}
	return lv.Size
}

func (lv *LVMLogicalVolume) EnsureSize(s uint64) bool {
	if lv == nil {
		panic("LVMLogicalVolume.EnsureSize: nil entity")
	}
	if s > lv.Size {
		lv.Size = alignUp(s)
		return true
	}
	return false
}

// lvname returns a name for a logical volume based on the mountpoint.
func lvname(path string) string {
	if path == "/" {
		return "rootlv"
	}

	path = strings.TrimLeft(path, "/")
	return strings.ReplaceAll(path, "/", "_") + "lv"
}

func (lv *LVMLogicalVolume) UnmarshalJSON(data []byte) error {
	// XXX: duplicated accross the Partition,LUKS,LVM :(
	type Alias LVMLogicalVolume
	var withoutPayload struct {
		Alias
		Payload     json.RawMessage
		PayloadType string `json:"payload_type"`
	}

	dec := json.NewDecoder(bytes.NewBuffer(data))
	if err := dec.Decode(&withoutPayload); err != nil {
		return fmt.Errorf("cannot build partition from %q: %w", data, err)
	}
	*lv = LVMLogicalVolume(withoutPayload.Alias)
	// no payload, e.g. bios partiton
	if withoutPayload.PayloadType == "no-payload" || withoutPayload.PayloadType == "" {
		return nil
	}

	entType := payloadEntityMap[withoutPayload.PayloadType]
	if entType == nil {
		return fmt.Errorf("cannot build partition from %q", data)
	}
	entValP := reflect.New(entType).Elem().Addr()
	ent := entValP.Interface()
	if err := json.Unmarshal(withoutPayload.Payload, &ent); err != nil {
		return err
	}
	lv.Payload = ent.(PayloadEntity)
	return nil
}

func (lv *LVMLogicalVolume) UnmarshalYAML(unmarshal func(any) error) error {
	return unmarshalYAMLviaJSON(lv, unmarshal)
}
