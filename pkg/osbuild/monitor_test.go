package osbuild_test

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	_ "embed"

	"github.com/osbuild/images/pkg/osbuild"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/monitor-simple.seq.json
var osbuildMonitorLines_curl []byte

func TestScannerSimple(t *testing.T) {
	ts1 := 1731589338.8252223 * 1000
	ts2 := 1731589338.8256931 * 1000
	ts3 := 1731589407.0338647 * 1000

	scanner := osbuild.NewStatusScanner(bytes.NewBuffer(osbuildMonitorLines_curl))
	// first line
	st, err := scanner.Status()
	assert.NoError(t, err)
	assert.Equal(t, &osbuild.Status{
		Trace: "source/org.osbuild.curl (org.osbuild.curl): Downloaded https://rpmrepo.osbuild.org/v2/mirror/public/f39/f39-x86_64-fedora-20231109/Packages/k/kpartx-0.9.5-2.fc39.x86_64.rpm",
		Progress: &osbuild.Progress{
			Done:    0,
			Total:   4,
			Message: "Pipeline source org.osbuild.curl",
		},
		Timestamp: time.UnixMilli(int64(ts1)),
	}, st)
	// second line
	st, err = scanner.Status()
	assert.NoError(t, err)
	assert.Equal(t, &osbuild.Status{
		Trace: "source/org.osbuild.curl (org.osbuild.curl): Downloaded https://rpmrepo.osbuild.org/v2/mirror/public/f39/f39-x86_64-fedora-20231109/Packages/l/langpacks-fonts-en-4.0-9.fc39.noarch.rpm",
		Progress: &osbuild.Progress{
			Done:    0,
			Total:   4,
			Message: "Pipeline source org.osbuild.curl",
		},
		Timestamp: time.UnixMilli(int64(ts2)),
	}, st)
	// third line
	st, err = scanner.Status()
	assert.NoError(t, err)
	assert.Equal(t, &osbuild.Status{
		Message: "Starting pipeline build",
		Progress: &osbuild.Progress{
			Done:    1,
			Total:   4,
			Message: "Pipeline build",
			SubProgress: &osbuild.Progress{
				Message: "Stage ",
				Done:    0,
				Total:   2,
			},
		},
		Timestamp: time.UnixMilli(int64(ts3)),
	}, st)
	// end
	st, err = scanner.Status()
	assert.NoError(t, err)
	assert.Nil(t, st)
}

//go:embed testdata/monitor-subprogress.seq.json
var osbuildMontiorLines_subprogress []byte

func TestScannerSubprogress(t *testing.T) {
	ts1 := 1731600115.14839 * 1000

	scanner := osbuild.NewStatusScanner(bytes.NewBuffer(osbuildMontiorLines_subprogress))
	st, err := scanner.Status()
	assert.NoError(t, err)
	assert.Equal(t, &osbuild.Status{
		Message: "Starting module org.osbuild.rpm",
		Progress: &osbuild.Progress{
			Done:    1,
			Total:   4,
			Message: "Pipeline build",
			SubProgress: &osbuild.Progress{
				Done:    2,
				Total:   8,
				Message: "Stage org.osbuild.rpm",
				SubProgress: &osbuild.Progress{
					Done:    4,
					Total:   16,
					Message: "Stage org.osbuild.rpm",
				},
			},
		},
		Timestamp: time.UnixMilli(int64(ts1)),
	}, st)
}

func TestScannerSmoke(t *testing.T) {
	f, err := os.Open("../../test/data/osbuild-monitor-output.json")
	require.NoError(t, err)
	defer f.Close()

	scanner := osbuild.NewStatusScanner(f)
	for {
		st, err := scanner.Status()
		assert.NoError(t, err)
		if st == nil {
			break
		}
		assert.NotEqual(t, time.Time{}, st.Timestamp)
	}
}

func TestScannerVeryLongLines(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	fmt.Fprint(buf, `{"message": "`)
	fmt.Fprint(buf, strings.Repeat("1", 128_000))
	fmt.Fprint(buf, `"}`)

	r := bytes.NewBufferString(buf.String())
	scanner := osbuild.NewStatusScanner(r)
	st, err := scanner.Status()
	assert.NoError(t, err)
	require.NotNil(t, st)
	assert.Equal(t, 128_000, len(st.Trace))
}

//go:embed testdata/monitor-duration.seq.json
var osbuildMonitorDuration_selinux []byte

func TestScannerDuration(t *testing.T) {
	ts1 := 1757401310.172594 * 1000
	dur1 := 0.5353707351023331

	scanner := osbuild.NewStatusScanner(bytes.NewBuffer(osbuildMonitorDuration_selinux))
	st, err := scanner.Status()
	assert.NoError(t, err)
	assert.Equal(t, &osbuild.Status{
		Trace: "Finished module org.osbuild.selinux",
		Progress: &osbuild.Progress{
			Total:   2,
			Done:    1,
			Message: "Pipeline ",
			SubProgress: &osbuild.Progress{
				Total:   3,
				Done:    3,
				Message: "Stage ",
			},
		},
		Timestamp: time.UnixMilli(int64(ts1)),
		Duration:  time.Duration(dur1 * float64(time.Second)),
	}, st)
}
