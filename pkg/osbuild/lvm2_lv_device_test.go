package osbuild_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/osbuild"
)

func TestLVM2LVDeviceMarshal(t *testing.T) {
	opts := &osbuild.LVM2LVDeviceOptions{Volume: "some-volume"}
	dev := osbuild.NewLVM2LVDevice("some-parent", opts)
	b, err := json.MarshalIndent(dev, "", " ")
	assert.NoError(t, err)
	expectedJSON := `{
 "type": "org.osbuild.lvm2.lv",
 "parent": "some-parent",
 "options": {
  "volume": "some-volume"
 }
}`
	assert.Equal(t, expectedJSON, string(b))
}

func TestLVM2LVDeviceMarshalWithDetectpv(t *testing.T) {
	opts := &osbuild.LVM2LVDeviceOptions{
		Volume:   "some-volume",
		Detectpv: common.ToPtr(true),
	}
	dev := osbuild.NewLVM2LVDevice("some-parent", opts)
	b, err := json.MarshalIndent(dev, "", " ")
	assert.NoError(t, err)
	expectedJSON := `{
 "type": "org.osbuild.lvm2.lv",
 "parent": "some-parent",
 "options": {
  "volume": "some-volume",
  "detectpv": true
 }
}`
	assert.Equal(t, expectedJSON, string(b))
}
