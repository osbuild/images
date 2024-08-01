package disk_test

import (
	"encoding/json"
	"fmt"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/internal/testdisk"
	"github.com/osbuild/images/pkg/disk"

	"golang.org/x/tools/go/packages"
)

func TestMarshalUnmarshalSimple(t *testing.T) {
	fakePt := testdisk.MakeFakePartitionTable("/", "/boot", "/boot/efi")

	js, err := json.Marshal(fakePt)
	assert.NoError(t, err)

	var ptFromJS disk.PartitionTable
	err = json.Unmarshal(js, &ptFromJS)
	assert.NoError(t, err)
	assert.Equal(t, fakePt, &ptFromJS)
}

func TestMarshalUnmarshalSad(t *testing.T) {
	var part disk.Partition
	err := json.Unmarshal([]byte(`{"randon": "json"}`), &part)
	assert.ErrorContains(t, err, `cannot build partition from "{`)
}

func TestMarshalUnmarshalPartitionHappy(t *testing.T) {
	part := &disk.Partition{}

	for _, ent := range []disk.PayloadEntity{
		&disk.Filesystem{Type: "ext2"},
		&disk.LUKSContainer{Passphrase: "secret"},
		&disk.Btrfs{Label: "foo"},
		&disk.LVMVolumeGroup{Name: "bar"},
	} {
		part.Payload = ent
		js, err := json.Marshal(part)
		assert.NoError(t, err)

		var partFromJS disk.Partition
		err = json.Unmarshal(js, &partFromJS)
		assert.NoError(t, err)
		assert.Equal(t, part, &partFromJS)
	}
}

func TestUnmarshalNullPayload(t *testing.T) {
	part := &disk.Partition{}
	part.Payload = nil

	js, err := json.Marshal(part)
	assert.NoError(t, err)

	var partFromJS disk.Partition
	err = json.Unmarshal(js, &partFromJS)
	assert.NoError(t, err)
	assert.Equal(t, part, &partFromJS)
}

func TestAllPayloadEntityExported(t *testing.T) {
	modulePath := "."
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedTypesInfo,
	}
	pkgs, err := packages.Load(cfg, modulePath)
	assert.NoError(t, err)

	var entityNameImpl []string
	for _, pkg := range pkgs {
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			if obj.Exported() {
				named, ok := obj.Type().(*types.Named)
				if !ok {
					continue
				}
				for i := 0; i < named.NumMethods(); i++ {
					method := named.Method(i)
					if method.Name() == "EntityName" {
						entityNameImpl = append(entityNameImpl, obj.Name())
					}
				}
			}
		}
	}
	// precondition check, ensure the test is working
	assert.True(t, len(entityNameImpl) >= 4)
	assert.Contains(t, entityNameImpl, "Btrfs")
	assert.Contains(t, entityNameImpl, "Filesystem")
	// check that when a new PayloadEntity is created it is part of the
	// payloadEntityMap so that the json marshaling will work
	assert.Equal(t, len(entityNameImpl), len(disk.PayloadEntityMap), fmt.Sprintf("the EntityName() function is implemented by %q but only %v are registered in %v, was a new PayloadEntity added but not registered?", entityNameImpl, len(disk.PayloadEntityMap), disk.PayloadEntityMap))
}

func TestImplementsInterfacesCompileTimeCheckPartition(t *testing.T) {
	var _ = disk.Container(&disk.Partition{})
	var _ = disk.Sizeable(&disk.Partition{})
}
