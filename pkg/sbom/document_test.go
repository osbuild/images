package sbom

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStandardTypeJSONUnmarhsall(t *testing.T) {
	type testStruct struct {
		Type     StandardType `json:"type"`
		TypeOmit StandardType `json:"type_omit,omitempty"`
	}

	tests := []struct {
		name string
		data []byte
		want testStruct
	}{
		{
			name: "StandardTypeNone",
			data: []byte(`{"type":""}`),
			want: testStruct{
				Type:     StandardTypeNone,
				TypeOmit: StandardTypeNone,
			},
		},
		{
			name: "StandardTypeSpdx",
			data: []byte(`{"type":"spdx","type_omit":"spdx"}`),
			want: testStruct{
				Type:     StandardTypeSpdx,
				TypeOmit: StandardTypeSpdx,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ts testStruct
			err := json.Unmarshal(tt.data, &ts)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, ts)
		})
	}
}

func TestStandardTypeJSONMarhsall(t *testing.T) {
	type TestStruct struct {
		Type     StandardType `json:"type"`
		TypeOmit StandardType `json:"type_omit,omitempty"`
	}

	tests := []struct {
		name string
		want []byte
		data TestStruct
	}{
		{
			name: "StandardTypeNone",
			want: []byte(`{"type":""}`),
			data: TestStruct{
				Type:     StandardTypeNone,
				TypeOmit: StandardTypeNone,
			},
		},
		{
			name: "StandardTypeSpdx",
			want: []byte(`{"type":"spdx","type_omit":"spdx"}`),
			data: TestStruct{
				Type:     StandardTypeSpdx,
				TypeOmit: StandardTypeSpdx,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []byte
			got, err := json.Marshal(tt.data)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
