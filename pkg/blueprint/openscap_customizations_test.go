package blueprint

import (
	"encoding/json"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
)

func TestGetOpenSCAPConfig(t *testing.T) {

	expectedOscap := OpenSCAPCustomization{
		DataStream: "test-data-stream.xml",
		ProfileID:  "test_profile",
		Tailoring: &OpenSCAPTailoringCustomizations{
			Selected:   []string{"quick_rule"},
			Unselected: []string{"very_slow_rule"},
			Overrides: []OpenSCAPTailoringOverride{
				OpenSCAPTailoringOverride{
					Var:   "rule_id",
					Value: 50,
				},
			},
		},
	}

	TestCustomizations := Customizations{
		OpenSCAP: &expectedOscap,
	}

	retOpenSCAPCustomiztions := TestCustomizations.GetOpenSCAP()

	assert.EqualValues(t, expectedOscap, *retOpenSCAPCustomiztions)
}

func TestOpenSCAPOverrideTOMLUnmarshaler(t *testing.T) {
	tests := []struct {
		name    string
		TOML    string
		want    *OpenSCAPTailoringOverride
		wantErr bool
	}{
		{
			name: "string based rule",
			TOML: `
var = "sshd_idle_timeout_value"
value = "600"
			`,
			want: &OpenSCAPTailoringOverride{
				Var:   "sshd_idle_timeout_value",
				Value: "600",
			},
			wantErr: false,
		},
		{
			name: "integer based rule",
			TOML: `
var = "sshd_idle_timeout_value"
value = 600
			`,
			want: &OpenSCAPTailoringOverride{
				Var:   "sshd_idle_timeout_value",
				Value: uint64(600),
			},
			wantErr: false,
		},
		{
			name: "invalid rule",
			TOML: `
var = "sshd_idle_timeout_value"
			`,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		var override OpenSCAPTailoringOverride
		err := toml.Unmarshal([]byte(tt.TOML), &override)
		if tt.wantErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.NotNil(t, override)
			assert.Equal(t, tt.want, &override)
		}
	}
}

func TestOpenSCAPOverrideJSONUnmarshaler(t *testing.T) {
	tests := []struct {
		name    string
		JSON    string
		want    *OpenSCAPTailoringOverride
		wantErr bool
	}{
		{
			name: "string based rule",
			JSON: `{
				"var": "sshd_idle_timeout_value",
				"value":  "600"
			}`,
			want: &OpenSCAPTailoringOverride{
				Var:   "sshd_idle_timeout_value",
				Value: "600",
			},
			wantErr: false,
		},
		{
			name: "integer based rule",
			JSON: `{
				"var": "sshd_idle_timeout_value",
				"value":  600
			}`,
			want: &OpenSCAPTailoringOverride{
				Var:   "sshd_idle_timeout_value",
				Value: uint64(600),
			},
			wantErr: false,
		},
		{
			name: "invalid rule",
			JSON: `{
				"var": "sshd_idle_timeout_value"
			}`,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		var override OpenSCAPTailoringOverride
		err := json.Unmarshal([]byte(tt.JSON), &override)
		if tt.wantErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.NotNil(t, override)
			assert.Equal(t, tt.want, &override)
		}
	}
}
