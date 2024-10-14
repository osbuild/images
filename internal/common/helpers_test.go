package common

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPanicOnError(t *testing.T) {
	err := errors.New("Error message")
	assert.PanicsWithValue(t, err, func() { PanicOnError(err) })
}

func TestIsStringInSortedSlice(t *testing.T) {
	assert.True(t, IsStringInSortedSlice([]string{"bart", "homer", "lisa", "marge"}, "homer"))
	assert.False(t, IsStringInSortedSlice([]string{"bart", "lisa", "marge"}, "homer"))
	assert.False(t, IsStringInSortedSlice([]string{"bart", "lisa", "marge"}, ""))
	assert.False(t, IsStringInSortedSlice([]string{}, "homer"))
}

func TestSystemdMountUnit(t *testing.T) {
	for _, tc := range []struct {
		mountpoint   string
		expectedName string
	}{
		{"/", "-.mount"},
		{"/boot", "boot.mount"},
		{"/boot/efi", "boot-efi.mount"},
	} {
		name, err := MountUnitNameFor(tc.mountpoint)
		assert.NoError(t, err)
		assert.Equal(t, tc.expectedName, name)
	}
}
