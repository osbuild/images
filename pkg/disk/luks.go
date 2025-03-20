package disk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"

	"github.com/google/uuid"

	"github.com/osbuild/images/pkg/datasizes"
)

// Argon2id defines parameters for the key derivation function for LUKS.
type Argon2id struct {
	// Number of iterations to perform.
	Iterations uint

	// Amount of memory to use (in KiB).
	Memory uint

	// Degree of parallelism (i.e. number of threads).
	Parallelism uint
}

// ClevisBind defines parameters for binding a LUKS device with a given policy.
type ClevisBind struct {
	Pin    string
	Policy string

	// If enabled, the passphrase will be removed from the LUKS device at the
	// end of the build (using the org.osbuild.luks2.remove-key stage).
	RemovePassphrase bool `yaml:"remove_passphrase"`
}

// LUKSContainer represents a LUKS encrypted volume.
type LUKSContainer struct {
	Passphrase string
	UUID       string
	Cipher     string
	Label      string
	Subsystem  string
	SectorSize uint64

	// The password-based key derivation function's parameters.
	PBKDF Argon2id

	// Parameters for binding the LUKS device.
	Clevis *ClevisBind

	Payload Entity
}

func init() {
	payloadEntityMap["luks"] = reflect.TypeOf(LUKSContainer{})
}

func (lc *LUKSContainer) EntityName() string {
	return "luks"
}

func (lc *LUKSContainer) GetItemCount() uint {
	if lc.Payload == nil {
		return 0
	}
	return 1
}

func (lc *LUKSContainer) GetChild(n uint) Entity {
	if n != 0 {
		panic(fmt.Sprintf("invalid child index for LUKSContainer: %d != 0", n))
	}
	return lc.Payload
}

func (lc *LUKSContainer) Clone() Entity {
	if lc == nil {
		return nil
	}
	clc := &LUKSContainer{
		Passphrase: lc.Passphrase,
		UUID:       lc.UUID,
		Cipher:     lc.Cipher,
		Label:      lc.Label,
		Subsystem:  lc.Subsystem,
		SectorSize: lc.SectorSize,
		PBKDF: Argon2id{
			Iterations:  lc.PBKDF.Iterations,
			Memory:      lc.PBKDF.Memory,
			Parallelism: lc.PBKDF.Parallelism,
		},
		Payload: lc.Payload.Clone(),
	}
	if lc.Clevis != nil {
		clc.Clevis = &ClevisBind{
			Pin:              lc.Clevis.Pin,
			Policy:           lc.Clevis.Policy,
			RemovePassphrase: lc.Clevis.RemovePassphrase,
		}
	}
	return clc
}

func (lc *LUKSContainer) GenUUID(rng *rand.Rand) {
	if lc == nil {
		return
	}

	if lc.UUID == "" {
		lc.UUID = uuid.Must(newRandomUUIDFromReader(rng)).String()
	}
}

func (lc *LUKSContainer) MetadataSize() uint64 {
	if lc == nil {
		return 0
	}

	// 16 MiB is the default size for the LUKS2 header
	return 16 * datasizes.MiB
}

func (lc *LUKSContainer) minSize(size uint64) uint64 {
	// since a LUKS container can contain pretty much any payload, but we only
	// care about the ones that have a size, or contain children with sizes
	minSize := lc.MetadataSize()
	switch payload := lc.Payload.(type) {
	case VolumeContainer:
		minSize += payload.minSize(size)
	case Sizeable:
		minSize += payload.GetSize()
	}
	return minSize
}

func (lc *LUKSContainer) UnmarshalJSON(data []byte) error {
	// XXX: duplicated accross the Partition,LUKS,LVM :(
	type Alias LUKSContainer
	var withoutPayload struct {
		Alias
		Payload     json.RawMessage
		PayloadType string `json:"payload_type"`
	}

	dec := json.NewDecoder(bytes.NewBuffer(data))
	if err := dec.Decode(&withoutPayload); err != nil {
		return fmt.Errorf("cannot build partition from %q: %w", data, err)
	}
	*lc = LUKSContainer(withoutPayload.Alias)
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
	lc.Payload = ent.(PayloadEntity)
	return nil
}

func (lc *LUKSContainer) UnmarshalYAML(unmarshal func(any) error) error {
	return unmarshalYAMLviaJSON(lc, unmarshal)
}
