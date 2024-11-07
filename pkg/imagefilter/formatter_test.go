package imagefilter_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/pkg/distrofactory"
	"github.com/osbuild/images/pkg/imagefilter"
)

func newFakeResult(t *testing.T, resultSpec string) imagefilter.Result {
	fac := distrofactory.NewTestDefault()

	l := strings.Split(resultSpec, ":")
	require.Equal(t, len(l), 3)

	// XXX: it would be nice if TestDistro would support constructing
	// like GetDistro("rhel-8.1:i386,amd64:ami,qcow2") that then
	// creates test distro/type/arch on the fly instead of the current
	// very static setup
	di := fac.GetDistro(l[0])
	require.NotNil(t, di)
	ar, err := di.GetArch(l[2])
	require.NoError(t, err)
	im, err := ar.GetImageType(l[1])
	require.NoError(t, err)
	return imagefilter.Result{di, ar, im}
}

func TestResultsFormatter(t *testing.T) {

	for _, tc := range []struct {
		formatter     string
		fakeResults   []string
		expectsOutput string
	}{
		{
			"",
			[]string{"test-distro-1:qcow2:test_arch3"},
			"test-distro-1 type:qcow2 arch:test_arch3\n",
		},
		{
			"text",
			[]string{"test-distro-1:qcow2:test_arch3"},
			"test-distro-1 type:qcow2 arch:test_arch3\n",
		},
		{
			"text",
			[]string{
				"test-distro-1:qcow2:test_arch3",
				"test-distro-1:test_type:test_arch",
			},
			"test-distro-1 type:qcow2 arch:test_arch3\n" +
				"test-distro-1 type:test_type arch:test_arch\n",
		},
		{
			"json",
			[]string{
				"test-distro-1:qcow2:test_arch3",
				"test-distro-1:test_type:test_arch",
			},
			`[{"distro":{"name":"test-distro-1"},"arch":{"name":"test_arch3"},"image_type":{"name":"qcow2"}},{"distro":{"name":"test-distro-1"},"arch":{"name":"test_arch"},"image_type":{"name":"test_type"}}]` + "\n",
		},
	} {
		res := make([]imagefilter.Result, len(tc.fakeResults))
		for i, resultSpec := range tc.fakeResults {
			res[i] = newFakeResult(t, resultSpec)
		}

		var buf bytes.Buffer
		fmter, err := imagefilter.NewResultsFormatter(imagefilter.OutputFormat(tc.formatter))
		require.NoError(t, err)
		err = fmter.Output(&buf, res)
		assert.NoError(t, err)
		assert.Equal(t, tc.expectsOutput, buf.String(), tc)
	}
}
