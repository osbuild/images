package progress_test

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/bootc-image-builder/bib/internal/progress"
)

func TestProgressNew(t *testing.T) {
	restore := progress.MockIsattyIsTerminal(true)
	defer restore()

	for _, tc := range []struct {
		typ         string
		expected    interface{}
		expectedErr string
	}{
		{"term", &progress.TerminalProgressBar{}, ""},
		{"debug", &progress.DebugProgressBar{}, ""},
		{"plain", &progress.PlainProgressBar{}, ""},
		{"bad", nil, `unknown progress type: "bad"`},
	} {
		pb, err := progress.New(tc.typ)
		if tc.expectedErr == "" {
			assert.NoError(t, err)
			assert.Equal(t, reflect.TypeOf(pb), reflect.TypeOf(tc.expected), fmt.Sprintf("[%v] %T not the expected %T", tc.typ, pb, tc.expected))
		} else {
			assert.EqualError(t, err, tc.expectedErr)
		}
	}
}

func TestPlainProgress(t *testing.T) {
	var buf bytes.Buffer
	restore := progress.MockOsStderr(&buf)
	defer restore()

	// plain progress never generates progress output
	pbar, err := progress.NewPlainProgressBar()
	assert.NoError(t, err)
	err = pbar.SetProgress(0, "set-progress", 1, 100)
	assert.NoError(t, err)
	assert.Equal(t, "", buf.String())

	// but it shows the messages
	pbar.SetPulseMsgf("pulse")
	assert.Equal(t, "pulse", buf.String())
	buf.Reset()

	pbar.SetMessagef("message")
	assert.Equal(t, "message", buf.String())
	buf.Reset()

	err = pbar.Start()
	assert.NoError(t, err)
	assert.Equal(t, "", buf.String())
	err = pbar.Stop()
	assert.NoError(t, err)
	assert.Equal(t, "", buf.String())
}

func TestDebugProgress(t *testing.T) {
	var buf bytes.Buffer
	restore := progress.MockOsStderr(&buf)
	defer restore()

	pbar, err := progress.NewDebugProgressBar()
	assert.NoError(t, err)
	err = pbar.SetProgress(0, "set-progress-msg", 1, 100)
	assert.NoError(t, err)
	assert.Equal(t, "[1 / 100] set-progress-msg\n", buf.String())
	buf.Reset()

	pbar.SetPulseMsgf("pulse-msg")
	assert.Equal(t, "pulse: pulse-msg\n", buf.String())
	buf.Reset()

	pbar.SetMessagef("some-message")
	assert.Equal(t, "msg: some-message\n", buf.String())
	buf.Reset()

	err = pbar.Start()
	assert.NoError(t, err)
	assert.Equal(t, "Start progressbar\n", buf.String())
	buf.Reset()

	err = pbar.Stop()
	assert.NoError(t, err)
	assert.Equal(t, "Stop progressbar\n", buf.String())
	buf.Reset()
}

func TestTermProgressNoTerm(t *testing.T) {
	var buf bytes.Buffer
	restore := progress.MockOsStderr(&buf)
	defer restore()

	// TODO: use something like "github.com/creack/pty" to create
	// a real pty to test this for real
	_, err := progress.NewTerminalProgressBar()
	assert.EqualError(t, err, "cannot use *os.File as a terminal")
}
