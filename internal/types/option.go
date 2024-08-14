package types

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/osbuild/images/internal/common"
)

// Option is a more constrained subset of the code in
//     https://github.com/moznion/go-optional
//
// It is not using an external go-optional lib directly because none
// has toml unmarshal support and also because there is no support for
// complex types in UnmarshalTOML (it only gives a single
// reflect.Value of type any) so our optional "lib" must come with
// limitations for this.
//
// Unfortunately there is no way I could find to import the code and
// limit the supported types. So this is a copy of the subset.
// Fortunatly the code is small, targeted and easy to follow so it
// should not be too bad.

type OptionTomlTypes interface {
	~int | ~bool | ~string
}

// Option is a subset of github.com/moznion/go-optional for use with toml

// Option is a data type that must be Some (i.e. having a value) or None (i.e. doesn't have a value).
// This type implements database/sql/driver.Valuer and database/sql.Scanner.
type Option[T OptionTomlTypes] []T

const (
	value = iota
)

// Some is a function to make an Option type value with the actual value.
func Some[T OptionTomlTypes](v T) Option[T] {
	return Option[T]{
		value: v,
	}
}

// None is a function to make an Option type value that doesn't have a value.
func None[T OptionTomlTypes]() Option[T] {
	return nil
}

// IsNone returns True if the Option *doesn't* have a value
func (o Option[T]) IsNone() bool {
	return o == nil
}

// IsSome returns whether the Option has a value or not.
func (o Option[T]) IsSome() bool {
	return o != nil
}

// Unwrap returns the value regardless of Some/None status.
// If the Option value is Some, this method returns the actual value.
// On the other hand, if the Option value is None, this method returns the *default* value according to the type.
func (o Option[T]) Unwrap() T {
	if o.IsNone() {
		var defaultValue T
		return defaultValue
	}
	return o[value]
}

// TakeOr returns the actual value if the Option has a value.
// On the other hand, this returns fallbackValue.
func (o Option[T]) TakeOr(fallbackValue T) T {
	if o.IsNone() {
		return fallbackValue
	}
	return o[value]
}

var jsonNull = []byte("null")

func (o Option[T]) MarshalJSON() ([]byte, error) {
	if o.IsNone() {
		return jsonNull, nil
	}

	marshal, err := json.Marshal(o.Unwrap())
	if err != nil {
		return nil, err
	}
	return marshal, nil
}

func (o *Option[T]) UnmarshalJSON(data []byte) error {
	if len(data) <= 0 || bytes.Equal(data, jsonNull) {
		*o = None[T]()
		return nil
	}

	var v T
	err := json.Unmarshal(data, &v)
	if err != nil {
		return err
	}
	*o = Some(v)

	return nil
}

// not part of github.com/moznion/go-optional
func (o *Option[T]) UnmarshalTOML(data any) error {
	b, ok := data.(T)
	if !ok {
		return fmt.Errorf("cannot use %[1]v (%[1]T) as bool", data)
	}
	*o = Some(b)
	return nil
}

func (o *Option[T]) UnmarshalYAML(unmarshal func(any) error) error {
	return common.UnmarshalYAMLviaJSON(o, unmarshal)
}
