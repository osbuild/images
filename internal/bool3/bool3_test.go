package bool3_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/internal/bool3"
)

func TestBasic(t *testing.T) {
	var b3 bool3.Bool3
	assert.Equal(t, b3, bool3.Unset)

	b3 = bool3.New(true)
	assert.Equal(t, b3, bool3.True)

	b3 = bool3.New(false)
	assert.Equal(t, b3, bool3.False)
}

func TestUnmarshal(t *testing.T) {
	var b3 bool3.Bool3

	err := json.Unmarshal([]byte(`true`), &b3)
	assert.NoError(t, err)
	assert.Equal(t, b3, bool3.True)

	err = json.Unmarshal([]byte(`false`), &b3)
	assert.NoError(t, err)
	assert.Equal(t, b3, bool3.False)
}

func TestMarshalUnmarshal(t *testing.T) {
	type b3struct struct {
		B bool3.Bool3 `json:"B"`
	}

	var t1 b3struct
	jsonOutput, err := json.Marshal(&t1)
	assert.NoError(t, err)
	assert.Equal(t, `{"B":null}`, string(jsonOutput))

	var t2 b3struct
	err = json.Unmarshal(jsonOutput, &t2)
	assert.NoError(t, err)
	assert.Equal(t, t2.B, bool3.Unset)
}

func TestUnmarshalBad(t *testing.T) {
	var b3 bool3.Bool3

	err := json.Unmarshal([]byte(`"foo"`), &b3)
	assert.EqualError(t, err, `cannot parse "foo" as Bool3`)

	err = json.Unmarshal([]byte(`3`), &b3)
	assert.EqualError(t, err, `cannot unmarshal float64 to Bool3`)
}
