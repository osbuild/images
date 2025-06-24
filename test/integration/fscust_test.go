package integration_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var fsCustConfigFmt = `{
  "name": "fs-customizations from local file",
  "blueprint": {
    "customizations": {
      "files": [
	{
	  "path": "/etc/file-from-host",
	  "uri": "file://%s"
	}
      ]
    }
  }
}
`

func TestFileCustomizationFromURI(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("this test needs to run as root")
	}

	tmpdir := t.TempDir()
	canaryString := "testdata"
	canaryPath := filepath.Join(tmpdir, "canary.txt")
	err := os.WriteFile(canaryPath, []byte(canaryString), 0644)
	assert.NoError(t, err)
	buildConfigPath := filepath.Join(tmpdir, "buildConfig.json")
	err = os.WriteFile(buildConfigPath, []byte(fmt.Sprintf(fsCustConfigFmt, canaryPath)), 0644)
	assert.NoError(t, err)

	cmd := exec.Command(
		"go", "run", "./cmd/build",
		"-distro", "centos-10",
		"-type", "tar",
		"-config", buildConfigPath,
		"--output", tmpdir,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = "../.."
	err = cmd.Run()
	assert.NoError(t, err)

	// using ibcli would make this easier as we have --output-name here
	l, err := filepath.Glob(tmpdir + "/*/archive/*.tar.xz")
	assert.NoError(t, err)
	require.Equal(t, 1, len(l))

	// ensure we get the expected content from the locally added file
	output, err := exec.Command("tar", "xOf", l[0], "./etc/file-from-host").CombinedOutput()
	assert.NoError(t, err, "tar output: %s", output)
	assert.Equal(t, canaryString, string(output))
}
