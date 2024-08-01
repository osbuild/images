package bool3

import (
	"encoding/json"
	"fmt"
)

type Bool3 int

const (
	Unset Bool3 = iota
	True
	False
)

func New(v bool) Bool3 {
	if v {
		return True
	}
	return False
}

func (b Bool3) String() string {
	switch b {
	case True:
		return "true"
	case False:
		return "false"
	default:
		return "unset"
	}
}

func parseBool3(v interface{}) (Bool3, error) {
	switch val := v.(type) {
	case nil:
		return Unset, nil
	case bool:
		switch val {
		case true:
			return True, nil
		case false:
			return False, nil
		}
	case string:
		switch val {
		case "true":
			return True, nil
		case "false":
			return False, nil
		case "unset":
			return Unset, nil
		default:
			return Unset, fmt.Errorf("cannot parse %q as Bool3", val)
		}
	}
	return Unset, fmt.Errorf("cannot unmarshal %T to Bool3", v)
}

func (b Bool3) MarshalJSON() ([]byte, error) {
	switch b {
	case True, False:
		return json.Marshal(b.String())
	}
	// XXX: or should we just remove this special case and use "unset"?
	return []byte("null"), nil
}

func (b *Bool3) UnmarshalJSON(data []byte) error {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	bb, err := parseBool3(v)
	*b = bb
	return err
}

func (b Bool3) MarshalTOML() ([]byte, error) {
	// TODO: return "null" here for unset too?
	return []byte(fmt.Sprintf("\"%s\"", b.String())), nil
}

func (b *Bool3) UnmarshalTOML(v interface{}) error {
	bb, err := parseBool3(v)
	*b = bb
	return err
}

func (b Bool3) MarshalYAML() (interface{}, error) {
	// TODO: return null here for unset too?
	return b.String(), nil
}

func (b *Bool3) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var v interface{}
	if err := unmarshal(&v); err != nil {
		return err
	}

	bb, err := parseBool3(v)
	*b = bb
	return err
}
