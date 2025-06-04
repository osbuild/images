package types_test

import (
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/internal/types"
)

type s1 struct {
	Name string `toml:"name"`
	Sub  s2     `toml:"sub"`
}

type s2 struct {
	ValBool    types.Option[bool]   `toml:"val-bool"`
	ValString  types.Option[string] `toml:"val-string"`
	ValString2 types.Option[string] `toml:"val-string2"`
}

func TestTomlParseOption(t *testing.T) {
	testTomlStr := `
name = "some-name"
[sub]
val-bool = true
val-string = "opt-string"
`

	var bp s1
	err := toml.Unmarshal([]byte(testTomlStr), &bp)
	assert.NoError(t, err)
	assert.Equal(t, bp.Name, "some-name")
	assert.Equal(t, types.Some(true), bp.Sub.ValBool)
	assert.Equal(t, types.Some("opt-string"), bp.Sub.ValString)
	assert.EqualValues(t, types.None[string](), bp.Sub.ValString2)
}

func TestTomlParseOptionBad(t *testing.T) {
	testTomlStr := `
[sub]
val-bool = 1234
`

	var bp s1
	err := toml.Unmarshal([]byte(testTomlStr), &bp)
	assert.ErrorContains(t, err, "cannot use 1234 (int64) as bool")
}

// taken from https://github.com/moznion/go-optional
func TestOption_IsNone(t *testing.T) {
	assert.True(t, types.None[int]().IsNone())
	assert.False(t, types.Some[int](123).IsNone())

	var nilValue types.Option[int] = nil
	assert.True(t, nilValue.IsNone())
}

func TestOption_IsSome(t *testing.T) {
	assert.False(t, types.None[int]().IsSome())
	assert.True(t, types.Some[int](123).IsSome())

	var nilValue types.Option[int] = nil
	assert.False(t, nilValue.IsSome())
}

func TestOption_Unwrap(t *testing.T) {
	assert.Equal(t, "foo", types.Some[string]("foo").Unwrap())
	assert.Equal(t, "", types.None[string]().Unwrap())
	assert.Equal(t, false, types.None[bool]().Unwrap())
}

func TestOption_TakeOr(t *testing.T) {
	v := types.Some[int](123).TakeOr(666)
	assert.Equal(t, 123, v)

	v = types.None[int]().TakeOr(666)
	assert.Equal(t, 666, v)
}
