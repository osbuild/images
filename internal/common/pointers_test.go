package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToPtr(t *testing.T) {
	var valueInt int = 42
	gotInt := ToPtr(valueInt)
	assert.Equal(t, valueInt, *gotInt)

	var valueBool bool = true
	gotBool := ToPtr(valueBool)
	assert.Equal(t, valueBool, *gotBool)

	var valueUint64 uint64 = 1
	gotUint64 := ToPtr(valueUint64)
	assert.Equal(t, valueUint64, *gotUint64)

	var valueStr string = "the-greatest-test-value"
	gotStr := ToPtr(valueStr)
	assert.Equal(t, valueStr, *gotStr)

}

func TestValueOrEmpty(t *testing.T) {
	var ptrInt *int
	valueInt := ValueOrEmpty(ptrInt)
	assert.Equal(t, 0, valueInt)
	helperInt := 20
	ptrInt = &helperInt
	valueInt = ValueOrEmpty(ptrInt)
	assert.Equal(t, 20, valueInt)

	var ptrBool *bool
	valueBool := ValueOrEmpty(ptrBool)
	assert.Equal(t, false, valueBool)
	helperBool := true
	ptrBool = &helperBool
	valueBool = ValueOrEmpty(ptrBool)
	assert.Equal(t, true, valueBool)

	var ptrUint64 *uint64
	valueUint64 := ValueOrEmpty(ptrUint64)
	assert.Equal(t, uint64(0), valueUint64)
	helperUint64 := uint64(20)
	ptrUint64 = &helperUint64
	valueUint64 = ValueOrEmpty(ptrUint64)
	assert.Equal(t, uint64(20), valueUint64)

	var ptrString *string
	valueString := ValueOrEmpty(ptrString)
	assert.Equal(t, "", valueString)
	helperString := "the-greatest-test-value"
	ptrString = &helperString
	valueString = ValueOrEmpty(ptrString)
	assert.Equal(t, "the-greatest-test-value", valueString)
}
