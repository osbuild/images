package osbuild_test

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/pkg/osbuild"
)

const osbuildMonitorLines_curl = `{"message": "source/org.osbuild.curl (org.osbuild.curl): Downloaded https://rpmrepo.osbuild.org/v2/mirror/public/f39/f39-x86_64-fedora-20231109/Packages/k/kpartx-0.9.5-2.fc39.x86_64.rpm\n", "context": {"origin": "org.osbuild", "pipeline": {"name": "source org.osbuild.curl", "id": "598849389c35f93efe2412446f5ca6919434417b9bcea040ea5f9203de81db2c", "stage": {}}, "id": "7355d3857aa5c7b3a0c476c13d4b242a625fe190f2e7796df2335f3a34429db3"}, "progress": {"name": "pipelines/sources", "total": 4, "done": 0}, "timestamp": 1731589338.8252223}
{"message": "source/org.osbuild.curl (org.osbuild.curl): Downloaded https://rpmrepo.osbuild.org/v2/mirror/public/f39/f39-x86_64-fedora-20231109/Packages/l/langpacks-fonts-en-4.0-9.fc39.noarch.rpm\n", "context": {"id": "7355d3857aa5c7b3a0c476c13d4b242a625fe190f2e7796df2335f3a34429db3"}, "progress": {"name": "pipelines/sources", "total": 4, "done": 0}, "timestamp": 1731589338.8256931}
{"message": "Starting pipeline build", "context": {"origin": "osbuild.monitor", "pipeline": {"name": "build", "id": "32e87da44d9a519e89770723a33b7ecdd4ab85b872ae6ab8aaa94bdef9a275c7", "stage": {}}, "id": "0020bdf60135d4a03d8db333f66d40386278bf55b39fd06ed18839da11d98f96"}, "progress": {"name": "pipelines/sources", "total": 4, "done": 1, "progress": {"name": "pipeline: build", "total": 2, "done": 0}}, "timestamp": 1731589407.0338647}`

func TestScannerSimple(t *testing.T) {
	ts1 := 1731589338.8252223 * 1000
	ts2 := 1731589338.8256931 * 1000
	ts3 := 1731589407.0338647 * 1000

	r := bytes.NewBufferString(osbuildMonitorLines_curl)
	scanner := osbuild.NewStatusScanner(r)
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

const osbuildMontiorLines_subprogress = `{"message": "Starting module org.osbuild.rpm", "context": {"origin": "osbuild.monitor", "pipeline": {"name": "build", "id": "32e87da44d9a519e89770723a33b7ecdd4ab85b872ae6ab8aaa94bdef9a275c7", "stage": {"name": "org.osbuild.rpm", "id": "bf00d0e1e216ffb796de06a1a7e9bb947d5a357f3f18ffea41a5611ee3ee0eac"}}, "id": "04c5aad63ba70bc39df10ad208cff66a108e44458e44eea41b305aee7a533877"}, "progress": {"name": "pipelines/sources", "total": 4, "done": 1, "progress": {"name": "pipeline: build", "total": 8, "done": 2, "progress": {"name": "sub-sub-progress", "total": 16, "done": 4}}}, "timestamp": 1731600115.148399}
`

func TestScannerSubprogress(t *testing.T) {
	ts1 := 1731600115.14839 * 1000

	r := bytes.NewBufferString(osbuildMontiorLines_subprogress)
	scanner := osbuild.NewStatusScanner(r)
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
