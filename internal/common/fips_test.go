package common

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFIPSEnabledHost(t *testing.T) {
	file, err := os.CreateTemp("/tmp", "fips_enabled")
	assert.NoError(t, err, "unable to create tmp file")
	defer file.Close()
	defer os.Remove(file.Name())
	FIPSEnabledFilePath = file.Name()

	fileContents := []string{
		"",
		"0\n",
		"1\n",
		"xxxxxx\n",
	}

	for _, fileContent := range fileContents {
		err = file.Truncate(0)
		assert.NoError(t, err, "truncating file: %s", file.Name())
		_, err = file.Seek(0, 0)
		assert.NoError(t, err, "seeking the begining of file: %s", file.Name())
		_, err = file.Write([]byte(fileContent))
		assert.NoError(t, err, "unable to write to file: %s", file.Name())
		if strings.TrimSpace(fileContent) == "1" {
			assert.Equal(t, IsBuildHostFIPSEnabled(), true)
		} else {
			assert.Equal(t, IsBuildHostFIPSEnabled(), false)
		}
	}
}
