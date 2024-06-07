package disk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
)

type Partition struct {
	Start    uint64 // Start of the partition in bytes
	Size     uint64 // Size of the partition in bytes
	Type     string // Partition type, e.g. 0x83 for MBR or a UUID for gpt
	Bootable bool   // `Legacy BIOS bootable` (GPT) or `active` (DOS) flag

	// ID of the partition, dos doesn't use traditional UUIDs, therefore this
	// is just a string.
	UUID string

	// If nil, the partition is raw; It doesn't contain a payload.
	Payload Entity
}

func (p *Partition) IsContainer() bool {
	return true
}

func (p *Partition) Clone() Entity {
	if p == nil {
		return nil
	}

	partition := &Partition{
		Start:    p.Start,
		Size:     p.Size,
		Type:     p.Type,
		Bootable: p.Bootable,
		UUID:     p.UUID,
	}

	if p.Payload != nil {
		partition.Payload = p.Payload.Clone()
	}

	return partition
}

func (pt *Partition) GetItemCount() uint {
	if pt == nil || pt.Payload == nil {
		return 0
	}
	return 1
}

func (p *Partition) GetChild(n uint) Entity {
	if n != 0 {
		panic(fmt.Sprintf("invalid child index for Partition: %d != 0", n))
	}
	return p.Payload
}

func (p *Partition) GetSize() uint64 {
	return p.Size
}

// Ensure the partition has at least the given size. Will do nothing
// if the partition is already larger. Returns if the size changed.
func (p *Partition) EnsureSize(s uint64) bool {
	if s > p.Size {
		p.Size = s
		return true
	}
	return false
}

func (p *Partition) IsBIOSBoot() bool {
	if p == nil {
		return false
	}

	return p.Type == BIOSBootPartitionGUID
}

func (p *Partition) IsPReP() bool {
	if p == nil {
		return false
	}

	return p.Type == "41" || p.Type == PRePartitionGUID
}

func (p *Partition) UnmarshalJSON(data []byte) error {
	// golang make this complicated: "Payload" is an interface that
	// can be either: "Filesystem", "LVMVolumeGroup", "Btrfs" or
	// "LUKSContainer". So we use a type registry and try to dynamically
	// detect the type
	type partAlias Partition
	var partWithoutPayload struct {
		partAlias
		Payload json.RawMessage
	}
	dec := json.NewDecoder(bytes.NewBuffer(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&partWithoutPayload); err != nil {
		return fmt.Errorf("cannot build partition from %q: %w", data, err)
	}
	*p = Partition(partWithoutPayload.partAlias)
	// check if payload is null first, that is valid for e.g. BIOS
	// partitions and if it's null no need to decode further
	var val interface{}
	if err := json.Unmarshal(partWithoutPayload.Payload, &val); err != nil {
		return fmt.Errorf("cannot decode payload: %w", err)
	}
	if val == nil {
		return nil
	}

	// now resolve payload, it's an interface
	for i := range entityTypes {
		// entValP is of type reflect.Value and points to a
		// struct that implements the Entity interface
		entValP := reflect.New(entityTypes[i]).Elem().Addr()
		// ent is the concrete type/value
		// (e.g. *disk.Filesysem), it's required so that
		// json.Decode() can introspect the fields
		ent := entValP.Interface()

		// try to decode, by disallowing unknown fields this
		// will only work for matching types
		dec := json.NewDecoder(bytes.NewBuffer(partWithoutPayload.Payload))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&ent); err != nil {
			continue
		}
		// the right payload is found and needs to be assigned/converted now
		p.Payload = entValP.Interface().(Entity)
		return nil
	}

	return fmt.Errorf("cannot build partition from: %q", data)
}
