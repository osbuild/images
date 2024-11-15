package osbuildmonitor_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/pkg/osbuildmonitor"
)

const osbuildMonitorLines_curl = `{"message": "source/org.osbuild.curl (org.osbuild.curl): Downloaded https://rpmrepo.osbuild.org/v2/mirror/public/f39/f39-x86_64-fedora-20231109/Packages/k/kpartx-0.9.5-2.fc39.x86_64.rpm\n", "context": {"origin": "org.osbuild", "pipeline": {"name": "source org.osbuild.curl", "id": "598849389c35f93efe2412446f5ca6919434417b9bcea040ea5f9203de81db2c", "stage": {}}, "id": "7355d3857aa5c7b3a0c476c13d4b242a625fe190f2e7796df2335f3a34429db3"}, "progress": {"name": "pipelines/sources", "total": 4, "done": 0}, "timestamp": 1731589338.8252223}
{"message": "source/org.osbuild.curl (org.osbuild.curl): Downloaded https://rpmrepo.osbuild.org/v2/mirror/public/f39/f39-x86_64-fedora-20231109/Packages/l/langpacks-fonts-en-4.0-9.fc39.noarch.rpm\n", "context": {"id": "7355d3857aa5c7b3a0c476c13d4b242a625fe190f2e7796df2335f3a34429db3"}, "progress": {"name": "pipelines/sources", "total": 4, "done": 0}, "timestamp": 1731589338.8256931}`

func TestScannerSimple(t *testing.T) {
	r := bytes.NewBufferString(osbuildMonitorLines_curl)
	scanner := osbuildmonitor.NewStatusScanner(r)
	// first line
	st, err := scanner.Status()
	assert.NoError(t, err)
	assert.Equal(t, &osbuildmonitor.Status{
		Trace: "source/org.osbuild.curl (org.osbuild.curl): Downloaded https://rpmrepo.osbuild.org/v2/mirror/public/f39/f39-x86_64-fedora-20231109/Packages/k/kpartx-0.9.5-2.fc39.x86_64.rpm",
		Progress: &osbuildmonitor.Progress{
			Done:  0,
			Total: 4,
		},
	}, st)
	// second line
	st, err = scanner.Status()
	assert.NoError(t, err)
	assert.Equal(t, &osbuildmonitor.Status{
		Trace: "source/org.osbuild.curl (org.osbuild.curl): Downloaded https://rpmrepo.osbuild.org/v2/mirror/public/f39/f39-x86_64-fedora-20231109/Packages/l/langpacks-fonts-en-4.0-9.fc39.noarch.rpm",
		Progress: &osbuildmonitor.Progress{
			Done:  0,
			Total: 4,
		},
	}, st)
	// end
	st, err = scanner.Status()
	assert.NoError(t, err)
	assert.Nil(t, st)
}

const osbuildMontiorLines_subprogress = `{"message": "Starting module org.osbuild.rpm", "context": {"origin": "osbuild.monitor", "pipeline": {"name": "build", "id": "32e87da44d9a519e89770723a33b7ecdd4ab85b872ae6ab8aaa94bdef9a275c7", "stage": {"name": "org.osbuild.rpm", "id": "bf00d0e1e216ffb796de06a1a7e9bb947d5a357f3f18ffea41a5611ee3ee0eac"}}, "id": "04c5aad63ba70bc39df10ad208cff66a108e44458e44eea41b305aee7a533877"}, "progress": {"name": "pipelines/sources", "total": 4, "done": 1, "progress": {"name": "pipeline: build", "total": 8, "done": 2, "progress": {"name": "sub-sub-progress", "total": 16, "done": 4}}}, "timestamp": 1731600115.148399}
`

func TestScannerSubprogress(t *testing.T) {
	r := bytes.NewBufferString(osbuildMontiorLines_subprogress)
	scanner := osbuildmonitor.NewStatusScanner(r)
	st, err := scanner.Status()
	assert.NoError(t, err)
	assert.Equal(t, &osbuildmonitor.Status{
		Trace: "Starting module org.osbuild.rpm",
		Progress: &osbuildmonitor.Progress{
			Done:  1,
			Total: 4,
			SubProgress: &osbuildmonitor.Progress{
				Done:  2,
				Total: 8,
				SubProgress: &osbuildmonitor.Progress{
					Done:  4,
					Total: 16,
				},
			},
		},
	}, st)
}

func TestScannerSmoke(t *testing.T) {
	f, err := os.Open("testdata/osbuild-monitor-output.json")
	require.NoError(t, err)
	defer f.Close()

	scanner := osbuildmonitor.NewStatusScanner(f)
	for {
		st, err := scanner.Status()
		assert.NoError(t, err)
		if st == nil {
			break
		}
	}
}
