package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsEmptyStruct(t *testing.T) {
	type aStruct struct {
		Slice   []string
		Pointer *bool
		Literal int
	}

	emptyStruct := aStruct{}
	r, err := IsEmptyStruct(emptyStruct)
	assert.NoError(t, err)
	assert.Equal(t, true, r)

	notEmptyStruct := aStruct{Literal: 1}
	r, err = IsEmptyStruct(notEmptyStruct)
	assert.NoError(t, err)
	assert.Equal(t, false, r)

	notEmptyStructWithEmptySlice := aStruct{Slice: []string{}}
	r, err = IsEmptyStruct(notEmptyStructWithEmptySlice)
	assert.NoError(t, err)
	assert.Equal(t, false, r)

	notAStruct := "not a struct"
	r, err = IsEmptyStruct(notAStruct)
	assert.Error(t, err)
	assert.Equal(t, false, r)
}
