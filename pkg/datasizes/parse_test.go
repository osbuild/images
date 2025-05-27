package datasizes_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/pkg/datasizes"
)

func TestDataSizeToUint64(t *testing.T) {
	cases := []struct {
		input   string
		success bool
		output  uint64
	}{
		{"123", true, 123},
		{"123 kB", true, 123000},
		{"123 KiB", true, 123 * 1024},
		{"123 MB", true, 123 * 1000 * 1000},
		{"123 MiB", true, 123 * 1024 * 1024},
		{"123 GB", true, 123 * 1000 * 1000 * 1000},
		{"123 GiB", true, 123 * 1024 * 1024 * 1024},
		{"123 TB", true, 123 * 1000 * 1000 * 1000 * 1000},
		{"123 TiB", true, 123 * 1024 * 1024 * 1024 * 1024},
		{"123kB", true, 123000},
		{"123KiB", true, 123 * 1024},
		{" 123  ", true, 123},
		{"  123kB  ", true, 123000},
		{"  123KiB  ", true, 123 * 1024},
		{"123 KB", false, 0},
		{"123 mb", false, 0},
		{"123 PB", false, 0},
		{"123 PiB", false, 0},
	}

	for _, c := range cases {
		result, err := datasizes.Parse(c.input)
		if c.success {
			require.Nil(t, err)
			assert.EqualValues(t, c.output, result)
		} else {
			assert.NotNil(t, err)
		}
	}
}

func TestParseSizeInJSONMapping(t *testing.T) {
	testCases := []struct {
		name      string
		sizeField string
		input     []byte
		expected  []byte
		err       error
	}{
		{
			name:      "no size field",
			sizeField: "size",
			input:     []byte(`{"name": "test"}`),
			expected:  []byte(`{"name": "test"}`),
			err:       nil,
		},
		{
			name:      "uint size field",
			sizeField: "size",
			input:     []byte(`{"size": 123, "name": "test"}`),
			expected:  []byte(`{"size": 123, "name": "test"}`),
			err:       nil,
		},
		{
			name:      "string size field",
			sizeField: "size",
			input:     []byte(`{"size": "123 MiB", "name": "test"}`),
			expected:  []byte(`{"size": 128974848, "name": "test"}`),
			err:       nil,
		},
		{
			name:      "string size field without unit",
			sizeField: "size",
			input:     []byte(`{"size": "123", "name": "test"}`),
			expected:  []byte(`{"size": 123, "name": "test"}`),
			err:       nil,
		},
		{
			name:      "invalid size field",
			sizeField: "size",
			input:     []byte(`{"size": "123 GazillionBytes", "name": "test"}`),
			expected:  nil,
			err:       fmt.Errorf("failed to parse size field named \"size\" to bytes: unknown data size units in string: 123 GazillionBytes"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := datasizes.ParseSizeInJSONMapping(tc.sizeField, tc.input)
			if tc.err != nil {
				assert.Error(t, err)
				assert.ErrorContains(t, tc.err, err.Error())
				return
			}

			require.NoError(t, err)
			assert.JSONEq(t, string(tc.expected), string(got))
		})
	}
}
