package common_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/internal/common"
)

func TestShlexQuote(t *testing.T) {
	assert.Equal(t, `''`, common.ShlexQuote(""))
	assert.Equal(t, `'test file name'`, common.ShlexQuote(`test file name`))
	unsafe := `Robert'); DROP TABLE`
	assert.Equal(t, `'Robert'"'"'); DROP TABLE'`, common.ShlexQuote(unsafe))
}
