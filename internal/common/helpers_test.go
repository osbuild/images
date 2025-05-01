package common

import (
	"encoding/base64"
	"errors"
	"fmt"
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

func TestMustHappy(t *testing.T) {
	var mustTesterRet string
	var mustTesterErr error
	mustTester := func() (string, error) {
		return mustTesterRet, mustTesterErr
	}

	mustTesterRet = "happy"
	mustTesterErr = nil
	res := Must(mustTester())
	assert.Equal(t, res, "happy")

	mustTesterRet = "unhappy"
	mustTesterErr = fmt.Errorf("some error")
	assert.PanicsWithError(t, "some error", func() {
		Must(mustTester())
	})
}

func TestEncodeUTF16le(t *testing.T) {
	// the function was written to replicate 'iconv -f ASCII -t UTF-16LE' so these
	// test cases were generated using that command
	type testCase struct {
		src   string
		exp64 string // generate with 'echo -e -n '<src>' | iconv -f ASCII -t UTF-16LE | base64 -w0'
	}

	testCases := []testCase{
		{
			src:   "test",
			exp64: "dABlAHMAdAA=",
		},
		{
			src:   "this\nis\na\nmultiline\nstring\n\n",
			exp64: "dABoAGkAcwAKAGkAcwAKAGEACgBtAHUAbAB0AGkAbABpAG4AZQAKAHMAdAByAGkAbgBnAAoACgA=",
		},
		{
			src:   "shimx64.efi,redhat,\\EFI\\Linux\\ffffffffffffffffffffffffffffffff-5.14.0-528.el9.x86_64.efi ,UKI bootentry",
			exp64: "cwBoAGkAbQB4ADYANAAuAGUAZgBpACwAcgBlAGQAaABhAHQALABcAEUARgBJAFwATABpAG4AdQB4AFwAZgBmAGYAZgBmAGYAZgBmAGYAZgBmAGYAZgBmAGYAZgBmAGYAZgBmAGYAZgBmAGYAZgBmAGYAZgBmAGYAZgBmAC0ANQAuADEANAAuADAALQA1ADIAOAAuAGUAbAA5AC4AeAA4ADYAXwA2ADQALgBlAGYAaQAgACwAVQBLAEkAIABiAG8AbwB0AGUAbgB0AHIAeQA=",
		},
	}

	assert := assert.New(t)
	for _, tc := range testCases {
		exp, err := base64.StdEncoding.DecodeString(tc.exp64)
		assert.NoError(err)
		assert.Equal(exp, EncodeUTF16le(tc.src))
	}
}
