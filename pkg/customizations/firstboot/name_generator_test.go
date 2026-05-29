package firstboot

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScriptNameGenerator(t *testing.T) {
	var ng scriptNameGenerator

	assert.Equal(t, "osbuild-first-my-script", ng.generate("my-script", "custom"))
	assert.Equal(t, "osbuild-first-custom-1", ng.generate("", "custom"))
	assert.Equal(t, "osbuild-first-custom-2", ng.generate("custom-42", "custom"))
	assert.Equal(t, "osbuild-first-test", ng.generate("test", "custom"))
	assert.Equal(t, "osbuild-first-custom-3", ng.generate("test", "custom"))
}
