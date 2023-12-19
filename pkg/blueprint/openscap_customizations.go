package blueprint

import (
	"encoding/json"
	"fmt"
)

type OpenSCAPCustomization struct {
	Datastream string                           `json:"datastream,omitempty" toml:"datastream,omitempty"`
	ProfileID  string                           `json:"profile_id,omitempty" toml:"profile_id,omitempty"`
	Tailoring  *OpenSCAPTailoringCustomizations `json:"tailoring,omitempty" toml:"tailoring,omitempty"`
}

type OpenSCAPTailoringCustomizations struct {
	Selected   []string                    `json:"selected,omitempty" toml:"selected,omitempty"`
	Unselected []string                    `json:"unselected,omitempty" toml:"unselected,omitempty"`
	Overrides  []OpenSCAPTailoringOverride `json:"overrides,omitempty" toml:"overrides,omitempty"`
}

type OpenSCAPTailoringOverride struct {
	Var   string      `json:"var,omitempty" toml:"var,omitempty"`
	Value interface{} `json:"value,omitempty" toml:"value,omitempty"`
}

func (c *Customizations) GetOpenSCAP() *OpenSCAPCustomization {
	if c == nil {
		return nil
	}
	return c.OpenSCAP
}

func (ot *OpenSCAPTailoringOverride) UnmarshalTOML(data interface{}) error {
	d, _ := data.(map[string]interface{})

	switch d["var"].(type) {
	case string:
		ot.Var = d["var"].(string)
	default:
		return fmt.Errorf("TOML unmarshal: override var must be string, got %v of type %T", d["var"], d["var"])
	}

	switch d["value"].(type) {
	case int64:
		ot.Value = uint64(d["value"].(int64))
	case string:
		ot.Value = d["value"].(string)
	default:
		return fmt.Errorf("TOML unmarshal: override value must be integer or string, got %v of type %T", d["value"], d["value"])
	}

	return nil
}

func (ot *OpenSCAPTailoringOverride) UnmarshalJSON(data []byte) error {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	d, _ := v.(map[string]interface{})

	switch d["var"].(type) {
	case string:
		ot.Var = d["var"].(string)
	default:
		return fmt.Errorf("JSON unmarshal: override var must be string, got %v of type %T", d["var"], d["var"])
	}

	switch d["value"].(type) {
	case float64:
		ot.Value = uint64(d["value"].(float64))
	case string:
		ot.Value = d["value"].(string)
	default:
		return fmt.Errorf("JSON unmarshal: override var must be float64 number or string, got %v of type %T", d["value"], d["value"])
	}

	return nil
}
