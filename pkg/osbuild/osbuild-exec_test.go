package osbuild_test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/pkg/datasizes"
	"github.com/osbuild/images/pkg/osbuild"
)

func mockOsbuildCmd(s string) (restore func()) {
	saved := osbuild.OSBuildCmd
	osbuild.OSBuildCmd = s
	return func() {
		osbuild.OSBuildCmd = saved
	}
}

func makeFakeOsbuild(t *testing.T, content string) string {
	p := filepath.Join(t.TempDir(), "fake-osbuild")
	err := os.WriteFile(p, []byte("#!/bin/sh\n"+content), 0755)
	assert.NoError(t, err)
	return p
}

func TestNewOSBuildCmdNilOptions(t *testing.T) {
	mf := []byte(`{"real": "manifest"}`)
	cmd := osbuild.NewOSBuildCmd(mf, nil, nil, nil)
	assert.NotNil(t, cmd)

	assert.Equal(
		t,
		[]string{
			"osbuild",
			"--store",
			"",
			"--output-directory",
			"",
			fmt.Sprintf("--cache-max-size=%d", int64(20*datasizes.GiB)),
			"-",
		},
		cmd.Args,
	)

	stdin, err := io.ReadAll(cmd.Stdin)
	assert.NoError(t, err)
	assert.Equal(t, mf, stdin)
}

func TestNewOSBuildCmdFullOptions(t *testing.T) {
	mf := []byte(`{"real": "manifest"}`)
	cmd := osbuild.NewOSBuildCmd(
		mf,
		[]string{
			"export-1",
			"export-2",
		},
		[]string{
			"checkpoint-1",
			"checkpoint-2",
		},
		&osbuild.OSBuildOptions{
			StoreDir:     "store",
			OutputDir:    "output",
			ExtraEnv:     []string{"EXTRA_ENV_1=1", "EXTRA_ENV_2=2"},
			Monitor:      osbuild.MonitorLog,
			MonitorFD:    10,
			JSONOutput:   true,
			CacheMaxSize: 10 * datasizes.GiB,
		},
	)
	assert.NotNil(t, cmd)

	assert.Equal(
		t,
		[]string{
			"osbuild",
			"--store",
			"store",
			"--output-directory",
			"output",
			fmt.Sprintf("--cache-max-size=%d", int64(10*datasizes.GiB)),
			"-",
			"--export",
			"export-1",
			"--export",
			"export-2",
			"--checkpoint",
			"checkpoint-1",
			"--checkpoint",
			"checkpoint-2",
			"--monitor=LogMonitor",
			"--monitor-fd=10",
			"--json",
		},
		cmd.Args,
	)

	assert.Contains(t, cmd.Env, "EXTRA_ENV_1=1")
	assert.Contains(t, cmd.Env, "EXTRA_ENV_2=2")

	stdin, err := io.ReadAll(cmd.Stdin)
	assert.NoError(t, err)
	assert.Equal(t, mf, stdin)
}

func TestRunOSBuild(t *testing.T) {
	fakeOsbuildBinary := makeFakeOsbuild(t, `
if [ "$1" = "--version" ]; then
    echo '90000.0'
else
    echo '{"success": true}'
fi
`)
	restore := mockOsbuildCmd(fakeOsbuildBinary)
	defer restore()

	opts := &osbuild.OSBuildOptions{
		JSONOutput: true,
	}
	result, err := osbuild.RunOSBuild([]byte(`{"fake":"manifest"}`), nil, nil, nil, opts)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
}
