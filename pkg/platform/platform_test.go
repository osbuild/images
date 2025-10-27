package platform_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v3"

	"github.com/osbuild/images/pkg/platform"
)

func TestImageFormatString(t *testing.T) {
	assert.Equal(t, "unset", platform.FORMAT_UNSET.String())
	assert.Equal(t, "ova", platform.FORMAT_OVA.String())
}

func TestImageFormatStringUnknown(t *testing.T) {
	assert.PanicsWithError(t, "unknown image format 999", func() {
		_ = platform.ImageFormat(999).String()
	})
}

func TestImageFormatUnmarshal(t *testing.T) {
	ifmts := []platform.ImageFormat{
		platform.FORMAT_UNSET,
		platform.FORMAT_RAW,
		platform.FORMAT_ISO,
		platform.FORMAT_QCOW2,
		platform.FORMAT_VMDK,
		platform.FORMAT_VHD,
		platform.FORMAT_GCE,
		platform.FORMAT_OVA,
	}
	for _, ifmt := range ifmts {
		inpJSON := fmt.Sprintf("%q", ifmt.String())
		inpYAML := ifmt.String()
		// json
		var f platform.ImageFormat
		err := json.Unmarshal([]byte(inpJSON), &f)
		assert.NoError(t, err)
		assert.Equal(t, ifmt, f)
		// now YAML
		err = yaml.Unmarshal([]byte(inpYAML), &f)
		assert.NoError(t, err)
		assert.Equal(t, ifmt, f)
	}
}
