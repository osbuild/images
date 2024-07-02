package disk

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func (p *Partition) MarshalJSON() ([]byte, error) {
	type partAlias Partition

	var entityName string
	switch p.Payload.(type) {
	case nil:
		entityName = "no-payload"
	case *Filesystem:
		entityName = "filesystem"
	case *LUKSContainer:
		entityName = "luks"
	case *LVMVolumeGroup:
		entityName = "lvm"
	case *Btrfs:
		entityName = "btrfs"
	default:
		return nil, fmt.Errorf("unsupported payload type %T", p.Payload)
	}

	partWithPayloadType := struct {
		partAlias
		PayloadType string
	}{
		partAlias(*p),
		entityName,
	}

	return json.Marshal(partWithPayloadType)
}

func (p *Partition) UnmarshalJSON(data []byte) error {
	type partAlias Partition
	var partWithoutPayload struct {
		partAlias
		Payload     json.RawMessage
		PayloadType string
	}

	dec := json.NewDecoder(bytes.NewBuffer(data))
	if err := dec.Decode(&partWithoutPayload); err != nil {
		return fmt.Errorf("cannot build partition from %q: %w", data, err)
	}
	*p = Partition(partWithoutPayload.partAlias)
	// no payload, e.g. bios partiton
	if partWithoutPayload.PayloadType == "no-payload" {
		return nil
	}

	switch partWithoutPayload.PayloadType {
	case "":
		return fmt.Errorf("cannot build partition from %q", data)
	case "no-payload":
		return nil
	case "filesystem":
		var ent *Filesystem
		if err := json.Unmarshal(partWithoutPayload.Payload, &ent); err != nil {
			return err
		}
		p.Payload = ent
	case "luks":
		var ent *LUKSContainer
		if err := json.Unmarshal(partWithoutPayload.Payload, &ent); err != nil {
			return err
		}
		p.Payload = ent
	case "lvm":
		var ent *LVMVolumeGroup
		if err := json.Unmarshal(partWithoutPayload.Payload, &ent); err != nil {
			return err
		}
		p.Payload = ent
	case "btrfs":
		var ent *Btrfs
		if err := json.Unmarshal(partWithoutPayload.Payload, &ent); err != nil {
			return err
		}
		p.Payload = ent

	}

	return nil
}
